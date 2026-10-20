// Package widgets holds the controls Hoppscotch has and MyGo does not, so
// the views read like the original's components.
package widgets

import (
	"fmt"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// Row is one line of a key/value list: a query parameter, a request header,
// a form field or an environment variable.
type Row struct {
	Enabled bool
	Key     string
	Value   string
}

func (r Row) filled() bool { return r.Key != "" || r.Value != "" }

// State holds the rows of a key/value list, kept by the app as its other
// state. View edits it; the list keeps an empty row at the end so that
// typing starts a new one, as Hoppscotch's does. OnChange runs when a row is
// edited, added or removed.
type State struct {
	Rows     []Row
	OnChange func()
	// Compact narrows the key column, for lists shown in a sidebar.
	Compact bool

	add    bool
	remove int // one more than the row to remove; 0 removes none
}

// Flush applies the edits the last frame asked for, before the rows are
// bound again: a row is appended or removed, and an empty row ends the
// list so there is always a place to type the next one.
func (s *State) Flush() {
	changed := false
	if s.remove > 0 && s.remove <= len(s.Rows) {
		at := s.remove - 1
		s.Rows = append(s.Rows[:at], s.Rows[at+1:]...)
		changed = true
	}
	s.remove = 0
	if s.add {
		s.Rows = append(s.Rows, Row{Enabled: true})
		s.add = false
		changed = true
	}
	if changed {
		s.notify()
	}
	for len(s.Rows) > 1 && !s.Rows[len(s.Rows)-1].filled() && !s.Rows[len(s.Rows)-2].filled() {
		s.Rows = s.Rows[:len(s.Rows)-1]
	}
	if len(s.Rows) == 0 || s.Rows[len(s.Rows)-1].filled() {
		s.Rows = append(s.Rows, Row{Enabled: true})
	}
}

// Enabled returns the rows a request sends: enabled ones with a key.
func (s *State) Enabled() []Row {
	rows := make([]Row, 0, len(s.Rows))
	for _, r := range s.Rows {
		if r.Enabled && r.Key != "" {
			rows = append(rows, r)
		}
	}
	return rows
}

// View draws the rows and the button that starts another one.
func (s *State) View(c *ui.Context, p design.Palette) {
	s.Flush()
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		for i := range s.Rows {
			s.row(c, p, i)
		}
		if ui.ButtonBase(c).Label("添加行").Justify(ui.Start).Padding(4, 8).Children(func() {
			ui.Icon(c, icons.Plus).FontSize(12).TextColor(p.TextMuted)
			ui.Text(c, "添加行").FontSize(design.FontSizeBody).TextColor(p.TextMuted).SingleLine()
		}).Clicked() {
			s.add = true
		}
	})
}

func (s *State) row(c *ui.Context, p design.Palette, i int) {
	r := &s.Rows[i]
	if s.Compact {
		s.compactRow(c, p, i, r)
		return
	}
	ui.Row(c).FillWidth().Height(32).AlignItems(ui.Center).Gap(6).Children(func() {
		checked := ui.Checkbox(c, &r.Enabled, "").Label(fmt.Sprintf("启用第 %d 行", i+1))
		key := ui.TextInput(c, &r.Key).Placeholder("键").Width(200).
			Label(fmt.Sprintf("键第 %d 行", i+1))
		value := ui.TextInput(c, &r.Value).Placeholder("值").Grow(1).
			Label(fmt.Sprintf("值第 %d 行", i+1))
		if checked.Changed() || key.Changed() || value.Changed() {
			s.notify()
		}
		if ui.ButtonBase(c).Width(24).Height(24).Label("删除").Children(func() {
			ui.Icon(c, icons.Trash).FontSize(12).TextColor(p.TextMuted)
		}).Clicked() {
			s.remove = i + 1
		}
	})
}

// compactRow stacks a row's key over its value, as a sidebar has no room
// for both side by side.
func (s *State) compactRow(c *ui.Context, p design.Palette, i int, r *Row) {
	changed := false
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).Gap(6).Children(func() {
			checked := ui.Checkbox(c, &r.Enabled, "").Label(fmt.Sprintf("启用第 %d 行", i+1))
			key := ui.TextInput(c, &r.Key).Placeholder("键").Grow(1).MinWidth(0).
				Label(fmt.Sprintf("键第 %d 行", i+1))
			if checked.Changed() || key.Changed() {
				changed = true
			}
			if ui.ButtonBase(c).Width(20).Height(20).Label("删除").Children(func() {
				ui.Icon(c, icons.Trash).FontSize(11).TextColor(p.TextMuted)
			}).Clicked() {
				s.remove = i + 1
			}
		})
		ui.Row(c).FillWidth().Padding(0, 0, 0, 26).Children(func() {
			value := ui.TextInput(c, &r.Value).Placeholder("值").Grow(1).MinWidth(0).
				Label(fmt.Sprintf("值第 %d 行", i+1))
			if value.Changed() {
				changed = true
			}
		})
	})
	if changed {
		s.notify()
	}
}

// notify tells the owner that a row was edited, added or removed.
func (s *State) notify() {
	if s.OnChange != nil {
		s.OnChange()
	}
}

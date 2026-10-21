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
	// File is the path of an uploaded file relative to the collection, and
	// FileName the name it had when it was chosen: a form-data field holds one
	// or the other.
	File     string
	FileName string
}

func (r Row) filled() bool { return r.Key != "" || r.Value != "" || r.File != "" }

// State holds the rows of a key/value list, kept by the app as its other
// state. View edits it; the list keeps an empty row at the end so that
// typing starts a new one, as Hoppscotch's does. OnChange runs when a row is
// edited, added or removed.
type State struct {
	Rows     []Row
	OnChange func()
	// Compact narrows the key column, for lists shown in a sidebar.
	Compact bool

	// ChooseFile, when set, makes each row's value an uploaded file instead of
	// text: it asks for one, keeps it where the collection does, and returns
	// that path with the name the file had. FileDropped runs with a file a row
	// no longer holds, so the app deletes it.
	ChooseFile  func() (path, name string, err error)
	FileDropped func(path string)

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
		s.dropFile(s.Rows[at])
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
	})
}

func (s *State) row(c *ui.Context, p design.Palette, i int) {
	r := &s.Rows[i]
	if s.Compact {
		s.compactRow(c, p, i, r)
		return
	}
	ui.Row(c).FillWidth().Height(32).AlignItems(ui.Center).Gap(6).Children(func() {
		checked := ui.Checkbox(c, &r.Enabled, "").Label(fmt.Sprintf("启用第 %d 行", i+1)).
			Cursor(ui.CursorPointer)
		key := ui.TextInput(c, &r.Key).Placeholder("键").Width(200).
			Label(fmt.Sprintf("键第 %d 行", i+1))
		if s.ChooseFile != nil {
			changed := checked.Changed() || key.Changed()
			if s.fileCell(c, p, i, r) {
				changed = true
			}
			if changed {
				s.notify()
			}
		} else {
			value := ui.TextInput(c, &r.Value).Placeholder("值").Grow(1).
				Label(fmt.Sprintf("值第 %d 行", i+1))
			if checked.Changed() || key.Changed() || value.Changed() {
				s.notify()
			}
		}
		if ui.ButtonBase(c).Width(24).Height(24).Label("删除").Cursor(ui.CursorPointer).
			Children(func() {
				ui.Icon(c, icons.Trash).FontSize(12).TextColor(p.TextMuted)
			}).Clicked() {
			s.remove = i + 1
		}
	})
}

// fileCell draws a row's value: text it is typed into, or the file it holds
// with the button that chooses another one and the name the file had. It
// reports whether the row changed.
func (s *State) fileCell(c *ui.Context, p design.Palette, i int, r *Row) bool {
	changed := false
	ui.Row(c).Grow(1).Height(28).AlignItems(ui.Center).Gap(8).Children(func() {
		if r.File == "" {
			value := ui.TextInput(c, &r.Value).Placeholder("值").Grow(1).
				Label(fmt.Sprintf("值第 %d 行", i+1))
			if value.Changed() {
				changed = true
			}
		} else {
			ui.Row(c).Height(24).AlignItems(ui.Center).Gap(6).Padding(0, 8).
				Radius(design.Radius).Background(p.SurfaceHover).Children(func() {
				ui.Icon(c, icons.File).FontSize(12).TextColor(p.TextMuted)
				ui.Text(c, r.FileName).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
			})
			// Clearing the file puts the row back to text, and deletes the file.
			if ui.ButtonBase(c).Width(18).Height(18).Label(fmt.Sprintf("清除文件第 %d 行", i+1)).
				Cursor(ui.CursorPointer).Children(func() {
				ui.Icon(c, icons.Close).FontSize(9).TextColor(p.TextMuted)
			}).Clicked() {
				s.dropFile(*r)
				r.File, r.FileName = "", ""
				changed = true
			}
		}
		if ui.ButtonBase(c).Height(24).Padding(0, 8).Radius(design.Radius).
			Label(fmt.Sprintf("选择文件第 %d 行", i+1)).Cursor(ui.CursorPointer).Children(func() {
			ui.Text(c, "选择文件").FontSize(design.FontSizeBody).
				TextColor(p.TextSecondary).SingleLine()
		}).Clicked() {
			if s.chooseFile(r) {
				changed = true
			}
		}
		if r.FileName != "" {
			ui.Text(c, r.FileName).FontSize(design.FontSizeTiny).
				TextColor(p.TextMuted).SingleLine()
		}
	})
	return changed
}

// chooseFile asks the app for a file and keeps it on the row, dropping the one
// it held. It reports whether the row changed.
func (s *State) chooseFile(r *Row) bool {
	path, name, err := s.ChooseFile()
	if err != nil || path == "" {
		return false
	}
	if old := r.File; old != "" && old != path {
		s.dropFile(Row{File: old})
	}
	r.File, r.FileName, r.Value = path, name, ""
	return true
}

// dropFile deletes the file a row held, through the app that keeps it.
func (s *State) dropFile(r Row) {
	if r.File != "" && s.FileDropped != nil {
		s.FileDropped(r.File)
	}
}

// compactRow stacks a row's key over its value, as a sidebar has no room
// for both side by side.
func (s *State) compactRow(c *ui.Context, p design.Palette, i int, r *Row) {
	changed := false
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).Gap(6).Children(func() {
			checked := ui.Checkbox(c, &r.Enabled, "").Label(fmt.Sprintf("启用第 %d 行", i+1)).
				Cursor(ui.CursorPointer)
			key := ui.TextInput(c, &r.Key).Placeholder("键").Grow(1).MinWidth(0).
				Label(fmt.Sprintf("键第 %d 行", i+1))
			if checked.Changed() || key.Changed() {
				changed = true
			}
			if ui.ButtonBase(c).Width(20).Height(20).Label("删除").Cursor(ui.CursorPointer).
				Children(func() {
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

// RemoveAt asks for the row at i to be removed by the next Flush, as the cross
// of a row a list draws itself does.
func (s *State) RemoveAt(i int) { s.remove = i + 1 }

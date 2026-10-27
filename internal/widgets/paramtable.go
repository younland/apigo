package widgets

import (
	"fmt"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// Param is one row of a parameters table: what it takes, the value it is sent
// with, and what it is for.
type Param struct {
	Enabled     bool
	Key         string
	Value       string
	Description string
}

// The columns of a parameters table, as the widths the header and the rows share
// so the two line up: the name and the value take the room left over, and the
// note takes the most.
const (
	paramCellGap    float32 = 6
	paramCheckWidth float32 = 22
	paramRowHeight  float32 = 34
	paramNameGrow   float32 = 3
	paramValueGrow  float32 = 3
	paramNoteGrow   float32 = 4
	paramDelete     float32 = 24
)

// Params is the state of a parameters table: its rows, and whether they are the
// parameters of a path, whose names come from the address itself, so the table
// neither adds nor deletes rows.
type Params struct {
	Rows []Param
	// Fixed makes the names read-only and the rows the address's own: a path
	// takes the names its placeholders give it.
	Fixed    bool
	OnChange func()

	add    bool
	remove int // one more than the row to remove; 0 removes none
}

// Flush applies the edits the last frame asked for, before the rows are bound
// again.
func (s *Params) Flush() {
	changed := false
	if s.remove > 0 && s.remove <= len(s.Rows) {
		at := s.remove - 1
		s.Rows = append(s.Rows[:at], s.Rows[at+1:]...)
		changed = true
	}
	s.remove = 0
	if s.add && !s.Fixed {
		s.Rows = append(s.Rows, Param{Enabled: true})
		s.add = false
		changed = true
	}
	// A table has a row to type in even when it holds nothing: the place the
	// first parameter starts.
	if !s.Fixed && len(s.Rows) == 0 {
		s.Rows = append(s.Rows, Param{Enabled: true})
		changed = true
	}
	if changed && s.OnChange != nil {
		s.OnChange()
	}
}

// Enabled returns the rows a request sends: the ones turned on that name
// something.
func (s *Params) Enabled() []Param {
	out := make([]Param, 0, len(s.Rows))
	for _, row := range s.Rows {
		if row.Enabled && row.Key != "" {
			out = append(out, row)
		}
	}
	return out
}

// View draws a table of parameters under its title: the columns it holds, the
// rows themselves, and the line another row starts from.
func (s *Params) View(c *ui.Context, p design.Palette, title string) {
	s.Flush()
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Text(c, title).Bold().FontSize(design.FontSizeBody).TextColor(p.Text)
		s.header(c, p)
		for i := range s.Rows {
			s.row(c, p, i)
		}
		if !s.Fixed {
			s.addRow(c, p)
		}
	})
}

// header names the columns of the table, in the widths the rows below use.
func (s *Params) header(c *ui.Context, p design.Palette) {
	ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).Gap(paramCellGap).Children(func() {
		ui.Row(c).Width(paramCheckWidth)
		paramColumn(c, p, "参数名", paramNameGrow)
		paramColumn(c, p, "参数值", paramValueGrow)
		paramColumn(c, p, "说明", paramNoteGrow)
		ui.Row(c).Width(paramDelete)
	})
}

// paramColumn is one heading of the table.
func paramColumn(c *ui.Context, p design.Palette, text string, grow float32) {
	ui.Text(c, text).Grow(grow).FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
}

// row draws one parameter: whether it is sent, its name and value, and what it
// is for.
func (s *Params) row(c *ui.Context, p design.Palette, i int) {
	r := &s.Rows[i]
	changed := false
	ui.Row(c).FillWidth().Height(paramRowHeight).AlignItems(ui.Center).
		Gap(paramCellGap).Children(func() {
		check := ui.Checkbox(c, &r.Enabled, "").Width(paramCheckWidth).
			Label(fmt.Sprintf("启用%s参数第 %d 行", s.prefix(), i+1)).Cursor(ui.CursorPointer)
		if s.nameCell(c, p, i, r) || check.Changed() {
			changed = true
		}
		value := ui.TextInput(c, &r.Value).Grow(paramValueGrow).MinWidth(0).
			Placeholder("参数值").Label(fmt.Sprintf("%s参数值第 %d 行", s.prefix(), i+1))
		note := ui.TextInput(c, &r.Description).Grow(paramNoteGrow).MinWidth(0).
			Placeholder("说明").Label(fmt.Sprintf("%s参数说明第 %d 行", s.prefix(), i+1))
		if value.Changed() || note.Changed() {
			changed = true
		}
		s.deleteCell(c, p, i)
	})
	if changed && s.OnChange != nil {
		s.OnChange()
	}
}

// prefix names a row's controls: the path table reads the same as the query one,
// so its labels say which of the two they belong to.
func (s *Params) prefix() string {
	if s.Fixed {
		return "路径"
	}
	return ""
}

// nameCell draws the name of a parameter: an input in both tables, as the two
// read alike. A path's name is the one its address gives it, so the field is
// read-only: it can be selected and copied, not renamed.
func (s *Params) nameCell(c *ui.Context, p design.Palette, i int, r *Param) bool {
	input := ui.TextInput(c, &r.Key).Grow(paramNameGrow).MinWidth(0).
		Placeholder("参数名").Label(fmt.Sprintf("%s参数名第 %d 行", s.prefix(), i+1))
	if s.Fixed {
		input.ReadOnly(true).TextColor(p.TextSecondary)
		return false
	}
	return input.Changed()
}

// deleteCell draws the cross that drops a row: a path's rows belong to its
// address, so the column keeps its room and holds nothing.
func (s *Params) deleteCell(c *ui.Context, p design.Palette, i int) {
	if s.Fixed {
		ui.Row(c).Width(paramDelete)
		return
	}
	if ui.ButtonBase(c).Width(paramDelete).Height(24).
		Label(fmt.Sprintf("删除参数第 %d 行", i+1)).Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Trash).FontSize(12).TextColor(p.TextMuted)
	}).Clicked() {
		s.remove = i + 1
	}
}

// addRow is the line a table starts another parameter from, as Hoppscotch's
// 添加参数 reads under its rows.
func (s *Params) addRow(c *ui.Context, p design.Palette) {
	ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).Gap(paramCellGap).Children(func() {
		ui.Row(c).Width(paramCheckWidth)
		if ui.ButtonBase(c).Height(24).Justify(ui.Start).Padding(0, 2).
			Label("添加参数").Cursor(ui.CursorPointer).Children(func() {
			ui.Text(c, "添加参数").FontSize(design.FontSizeBody).TextColor(p.TextMuted).
				SingleLine()
		}).Clicked() {
			s.add = true
		}
	})
}

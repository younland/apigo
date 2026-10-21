package main

import (
	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// menuAction is one line of a popover menu: its icon, the text it shows and
// what it runs. A zero action draws the rule between two groups. danger colors
// what cannot be taken back, as deleting does.
type menuAction struct {
	icon   *ui.SVG
	text   string
	danger bool
	run    func()
}

// menuRule is the rule between two groups of a popover menu.
func menuRule() menuAction { return menuAction{} }

// menuPanel styles the panel a popover menu draws in, with the border every
// control of the app draws with.
func menuPanel(panel ui.Element, p design.Palette, width float32) {
	panel.Width(width).Padding(4).Radius(design.Radius).
		Background(p.Surface).Border(design.BorderWidth, p.Border)
}

// menuRow spans a row of a popover menu over the panel and tints the one the
// pointer is on, so a menu says which line a click would run.
func menuRow(row ui.Element, p design.Palette) ui.Element {
	row.FillWidth().Radius(design.Radius)
	if row.Hovered() {
		row.Background(p.SurfaceHover)
	}
	return row
}

// popoverMenu draws a menu under an anchor the way Hoppscotch draws its own:
// an icon and a label per line, with rules between groups. It stands in for
// the platform menu, which has no icons.
func (a *app) popoverMenu(c *ui.Context, p design.Palette, anchor ui.Element, open *bool, width float32, actions []menuAction) {
	ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
		menuPanel(panel, p, width)
		for _, action := range actions {
			if action.run == nil {
				ui.Row(c).Height(design.BorderWidth).FillWidth().Background(p.Divider)
				continue
			}
			// What cannot be taken back reads in the danger color, icon and all.
			color := p.TextSecondary
			if action.danger {
				color = p.Danger
			}
			row := menuRow(ui.ButtonBase(c).Height(28).Label(action.text).Justify(ui.Start).
				Padding(0, 8).Gap(8).Cursor(ui.CursorPointer).Children(func() {
				ui.Icon(c, action.icon).FontSize(13).TextColor(color)
				ui.Text(c, action.text).FontSize(design.FontSizeBody).
					TextColor(color).SingleLine()
			}), p)
			if row.Clicked() {
				*open = false
				action.run()
			}
		}
	})
}

// menuFlag is the open state of one popover menu, keyed by what it belongs
// to: "folder/<path>", "collection/<id>" or "tab/<id>".
func (a *app) menuFlag(key string) *bool {
	if a.menus == nil {
		a.menus = map[string]*bool{}
	}
	flag := a.menus[key]
	if flag == nil {
		flag = new(bool)
		a.menus[key] = flag
	}
	return flag
}

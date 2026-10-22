package app

import (
	"apigo/internal/design"
	"apigo/internal/icons"

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

// selectSpec is the trigger of a list of choices: the name it is found by, the
// text it shows, and how wide it is.
type selectSpec struct {
	name  string
	label string
	width float32
}

// selectTrigger draws the control that opens a list of choices. Every picker of
// the app draws this way — the label at the left, the chevron at the right, the
// border of the other fields and the page behind it — so they read as one.
func selectTrigger(c *ui.Context, p design.Palette, spec selectSpec) ui.Element {
	btn := ui.ButtonBase(c).Width(spec.width).Height(design.TabStripHeight-12).
		Padding(0, 10).MinWidth(0).Background(p.Surface).Radius(design.Radius).
		Border(design.BorderWidth, p.Border).Justify(ui.SpaceBetween).
		Label(spec.name).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(p.SurfaceHover)
	}
	btn.Children(func() {
		ui.Text(c, spec.label).FontSize(design.FontSizeBody).TextColor(p.TextSecondary).SingleLine()
		ui.Icon(c, icons.Chevron).FontSize(12).TextColor(p.TextMuted)
	})
	return btn
}

// selectChoice is one line of a list of choices: the name it is found by, the
// text it shows, whether it is the one in force, and what picking it does.
type selectChoice struct {
	name   string
	label  string
	chosen bool
	run    func()
}

// selectChoiceRow draws one choice. The one in force is ticked, reads in the
// accent and is tinted; the others keep the tick's room, so labels line up.
func selectChoiceRow(c *ui.Context, p design.Palette, choice selectChoice) {
	row := menuRow(ui.ButtonBase(c).Height(28).Label(choice.name).Justify(ui.Start).
		Gap(8).Padding(0, 8).Cursor(ui.CursorPointer).Children(func() {
		text := p.TextSecondary
		if choice.chosen {
			ui.Icon(c, icons.Check).FontSize(12).TextColor(p.Accent)
			text = p.Accent
		} else {
			ui.Row(c).Width(12)
		}
		ui.Text(c, choice.label).FontSize(design.FontSizeBody).TextColor(text).SingleLine()
	}), p)
	if choice.chosen {
		row.Background(p.Accent.Alpha(accentTint))
	}
	if row.Clicked() {
		choice.run()
	}
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

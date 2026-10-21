package app

import (
	"apigo/internal/design"
	"apigo/internal/icons"
	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// kindIcon is the icon of a request kind.
func kindIcon(k kind) *ui.SVG {
	switch k {
	case kindGraphQL:
		return icons.Triangle
	case kindWebSocket:
		return icons.Plug
	case kindSSE:
		return icons.Bolt
	case kindSocketIO:
		return icons.Stack
	case kindMQTT:
		return icons.Radio
	default:
		return icons.Braces
	}
}

// kindRow is one line of a kind menu: the icon, the name, and the choice. The
// menu it fills draws the panel around it.
func (a *app) kindMenu(c *ui.Context, p design.Palette, choose func(kind)) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		for _, k := range store.RequestKinds {
			row := menuRow(ui.ButtonBase(c).Height(28).Label(store.KindName(k)).Justify(ui.Start).
				Padding(0, 8).Gap(8).Cursor(ui.CursorPointer).Children(func() {
				ui.Icon(c, kindIcon(k)).FontSize(13).TextColor(p.TextMuted)
				ui.Text(c, store.KindName(k)).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
			}), p)
			if row.Clicked() {
				choose(k)
			}
		}
	})
}

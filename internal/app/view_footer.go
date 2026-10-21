package app

import (
	"apigo/internal/design"
	"apigo/internal/store"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// footer draws the status bar: the window or collection shown, the kind of
// the request being edited, and the version.
func (a *app) footer(c *ui.Context, p design.Palette) {
	name, kind := "主窗口", ""
	if a.active != "" {
		name = a.activeCollection().Name
		if t := a.current(); t != nil {
			kind = store.KindName(t.kind)
		}
	}
	ui.Column(c).FillWidth().Children(func() {
		a.divider(c, p)
		ui.Row(c).Height(design.FooterHeight).FillWidth().
			Padding(0, design.RowGap).Gap(8).AlignItems(ui.Center).
			Children(func() {
				ui.Text(c, name).FontSize(design.FontSizeTiny).
					TextColor(p.TextMuted).SingleLine()
				if kind != "" {
					ui.Text(c, kind).FontSize(design.FontSizeTiny).
						TextColor(p.TextMuted).SingleLine()
				}
				ui.Row(c).Grow(1)
				ui.Text(c, appVersion()).FontSize(design.FontSizeTiny).
					TextColor(p.TextMuted).SingleLine()
			})
	})
}

// appVersion is the version mygo.json gives the build.
func appVersion() string {
	if v := mygo.App.Version(); v != "" {
		return "apigo " + v
	}
	return "apigo"
}

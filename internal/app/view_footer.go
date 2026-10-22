package app

import (
	"apigo/internal/design"
	"apigo/internal/icons"
	"apigo/internal/store"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// footer draws the status bar: the toggle of the collection's tree, the window
// or collection shown, the kind of the request being edited, and the version.
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
				// The tree of a collection folds away from the corner of the
				// window: a plain icon, with no box around it, as the status bar
				// is a line of text rather than a row of controls.
				if a.active != "" {
					a.sidebarToggle(c, p)
				}
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

// sidebarToggle folds the tree of a collection away and brings it back: the
// panel takes a quarter of a narrow window from the request. The icon of each
// state points the way the tree goes.
func (a *app) sidebarToggle(c *ui.Context, p design.Palette) {
	icon, name := icons.SidebarClose, "收起接口面板"
	if a.sidebarCollapsed {
		icon, name = icons.SidebarOpen, "展开接口面板"
	}
	btn := ui.ButtonBase(c).Width(20).Height(20).Radius(design.Radius).
		Label(name).Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(p.TextMuted)
	})
	if btn.Hovered() {
		btn.Background(p.SurfaceHover)
	}
	if btn.Clicked() {
		a.sidebarCollapsed = !a.sidebarCollapsed
	}
}

// appName is the name the app shows: its window's title, and the first word of
// the version line in the status bar. The files it writes keep the old lower
// case name, as the data directory and the meta file of a collection are.
const appName = "ApiGo"

// appVersion is the name of the app and the version mygo.json gives the build.
func appVersion() string {
	if v := mygo.App.Version(); v != "" {
		return appName + " " + v
	}
	return appName
}

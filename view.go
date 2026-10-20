package main

import (
	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// view builds the window: the collection tabs, then either the home window
// or the active collection's request tabs, panel and workspace, with the
// status bar under them.
func (a *app) view(c *ui.Context) {
	if t := a.themeFor(c); t != nil {
		c.SetTheme(t)
	}
	p := a.palette(c)
	// A write that fails while the workspace moves on is reported once.
	if a.saveErr != "" {
		c.Toast(a.saveErr)
		a.saveErr = ""
	}

	ui.Column(c).Fill().Background(p.Background).Children(func() {
		a.topBar(c, p)
		a.divider(c, p)
		if a.active == "" {
			a.homeView(c, p)
		} else {
			a.collectionWorkspace(c, p)
		}
		a.footer(c, p)
	})
	// Dialogs are overlays of the window, not children of the layout, so
	// their backdrop covers the whole window.
	a.collectionModal(c, p)
	a.envModal(c, p)
	a.settingsModal(c, p)
}

// collectionWorkspace draws the active collection: the interfaces tree on
// the left, and the request panel on the right, whose top row is the tabs
// of the open requests over the request and response panes.
func (a *app) collectionWorkspace(c *ui.Context, p design.Palette) {
	ws := a.space()
	ui.Column(c).Grow(1).FillWidth().Padding(0, design.WorkspacePadding).Children(func() {
		ui.Row(c).FillWidth().Grow(1).AlignItems(ui.Start).Children(func() {
			a.requestsPanel(c, p)
			ui.Row(c).Width(design.BorderWidth).FillHeight().Background(p.Divider)
			ui.Column(c).Grow(1).FillHeight().Children(func() {
				a.requestTabs(c, p, ws)
				a.divider(c, p)
				a.workspace(c, p)
			})
		})
	})
}

// workspace splits the room into the request pane above the response pane,
// as Hoppscotch does.
func (a *app) workspace(c *ui.Context, p design.Palette) {
	t := a.current()
	if t == nil {
		return
	}
	ui.SplitVerticalPercent(c, &a.percent, func() {
		if t.kind != kindREST {
			a.emptyKind(c, p, t.kind)
			return
		}
		a.request(c, p)
	}, func() {
		a.response(c, p)
	}).Grow(1)
}

// emptyKind stands in for a request kind the app does not send yet.
func (a *app) emptyKind(c *ui.Context, p design.Palette, k kind) {
	ui.Column(c).FillWidth().FillHeight().Center().Gap(design.RowGap).Children(func() {
		ui.Icon(c, kindIcon(k)).FontSize(28).TextColor(p.TextMuted)
		ui.Text(c, kindName(k)+" 请求尚未实现").FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
	})
}

// divider is the one-DIP line between regions.
func (a *app) divider(c *ui.Context, p design.Palette) {
	ui.Row(c).Height(design.BorderWidth).FillWidth().Background(p.Divider)
}

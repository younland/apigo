package app

import (
	"apigo/internal/design"
	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// view builds the window: the collection tabs, then either the home window
// or the active collection's request tabs, panel and workspace, with the
// status bar under them.
func (a *app) view(c *ui.Context) {
	c.SetTheme(a.themeFor(c))
	p := a.palette()
	// A write that fails while the workspace moves on is reported once.
	if a.saveErr != "" {
		c.Toast(a.saveErr)
		a.saveErr = ""
	}
	// The tree follows the files another program changes in the open
	// collection's directory.
	a.followFiles()

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
	a.importModal(c, p)
	a.envModal(c, p)
	a.settingsModal(c, p)
	a.confirmModal(c, p)
	a.folderModal(c, p)
}

// collectionWorkspace draws the active collection: the interfaces tree on
// the left, and the request panel on the right, whose top row is the tabs of
// the open requests over the request and response panes — or, with no tab
// open at all, a pane saying what to do. The status bar folds the tree away,
// for the room a narrow window needs.
func (a *app) collectionWorkspace(c *ui.Context, p design.Palette) {
	ws := a.space()
	// Ctrl+S saves the request the workspace shows, whatever its kind.
	c.OnShortcut(ui.Cmd, ui.KeyS, func() { a.saveCurrent(c) })
	ui.Column(c).Grow(1).FillWidth().Padding(0, design.WorkspacePadding).Children(func() {
		ui.Row(c).FillWidth().Grow(1).AlignItems(ui.Start).Children(func() {
			if !a.sidebarCollapsed {
				a.requestsPanel(c, p)
				ui.Row(c).Width(design.BorderWidth).FillHeight().Background(p.Divider)
			}
			ui.Column(c).Grow(1).FillHeight().Children(func() {
				// A collection opens with no request tab at all: the pane says
				// what to do rather than showing an empty request, and the
				// strip of tabs comes with the first tab.
				if ws.empty() {
					a.startHint(c, p)
					return
				}
				a.requestTabs(c, p, ws)
				a.divider(c, p)
				a.workspace(c, p)
			})
		})
	})
}

// workspace splits the room into the request pane above the response pane, as
// Hoppscotch does; a socket shows its log under the message instead.
func (a *app) workspace(c *ui.Context, p design.Palette) {
	t := a.current()
	if t == nil {
		return
	}
	ui.SplitVerticalPercent(c, &a.percent, func() {
		switch t.kind {
		case kindREST:
			a.request(c, p)
		case kindWebSocket:
			a.socketPane(c, p, t)
		default:
			a.emptyKind(c, p, t.kind)
		}
	}, func() {
		if t.kind == kindWebSocket {
			a.socketLogPane(c, p, t)
			return
		}
		a.response(c, p)
	}).Grow(1)
}

// emptyKind stands in for a request kind the app does not send yet.
func (a *app) emptyKind(c *ui.Context, p design.Palette, k kind) {
	ui.Column(c).FillWidth().FillHeight().Center().Gap(design.RowGap).Children(func() {
		ui.Icon(c, kindIcon(k)).FontSize(28).TextColor(p.TextMuted)
		ui.Text(c, store.KindName(k)+" 请求尚未实现").FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
	})
}

// divider is the one-DIP line between regions.
func (a *app) divider(c *ui.Context, p design.Palette) {
	ui.Row(c).Height(design.BorderWidth).FillWidth().Background(p.Divider)
}

// noScrollbars keeps a scroll container scrolling by the wheel, the trackpad
// and Shift+wheel, and draws no bar of its own: the insets move both bars out
// past any window. The strips of the app have no room for a bar of their own.
func noScrollbars(e ui.Element) ui.Element {
	const away float32 = 1 << 14
	return e.ScrollbarInsets(away, 0, 0, away)
}

// topBackdrop dims the window and puts the dialog over its top, centered:
// what a dialog asks stays in view of the list it is about, rather than
// covering the middle of it.
func topBackdrop(backdrop ui.Element) {
	backdrop.Background(ui.RGBA(0, 0, 0, 0.4)).Justify(ui.Start).
		Padding(design.PanelPadding*3, 0, 0, 0)
}

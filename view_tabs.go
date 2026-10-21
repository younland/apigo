package main

import (
	"fmt"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// requestTabs draws the collection's request tabs and the button that
// opens another one, with the environment picker of the right edge. The row
// spans the window: the tabs scroll before the picker as they grow, with no
// bar of their own, as the strip has no room under its tabs for one.
func (a *app) requestTabs(c *ui.Context, p design.Palette, ws *workspace) {
	ui.Row(c).Height(design.TabStripHeight).FillWidth().
		Background(p.Surface).AlignItems(ui.Start).Children(func() {
		noScrollbars(ui.ScrollHorizontal(c)).Grow(1).Height(design.TabStripHeight).Children(func() {
			ui.Row(c).Height(design.TabStripHeight).AlignItems(ui.Start).Children(func() {
				// Built from a snapshot: a tab's own buttons close tabs while
				// the strip is still being drawn.
				for _, t := range append([]*tab(nil), ws.tabs...) {
					a.tabItem(c, p, ws, t)
				}
				a.newTabButton(c, p, ws)
			})
		})
		a.environmentPicker(c, p)
	})
}

// requestTabMenuKey is the menu a request tab's popover keeps open, keyed by
// the tab itself: its index moves as tabs open and close.
func requestTabMenuKey(t *tab) string { return fmt.Sprintf("request-tab/%p", t) }

// tabIndex returns where a tab is kept, or -1 once it is closed: an index
// taken before a button ran is stale.
func (a *app) tabIndex(ws *workspace, t *tab) int {
	for i, other := range ws.tabs {
		if other == t {
			return i
		}
	}
	return -1
}

// tabItem draws one request tab: its method, title, edit dot and close
// button, with the menu Hoppscotch offers on a right click.
func (a *app) tabItem(c *ui.Context, p design.Palette, ws *workspace, t *tab) {
	i := a.tabIndex(ws, t)
	if i < 0 {
		return // closed while the strip was drawn
	}
	active := i == ws.active
	bg, line := p.Surface, p.Surface
	color := p.TextMuted
	if active {
		bg, line, color = p.Background, p.Accent, p.Text
	}

	tab := ui.Row(c).Height(design.TabStripHeight).Background(bg).AlignItems(ui.Start)
	open := a.menuFlag(requestTabMenuKey(t))
	tab.Children(func() {
		ui.Column(c).Height(design.TabStripHeight).Children(func() {
			ui.Row(c).Height(2).FillWidth().Background(line)
			ui.Row(c).Height(design.TabStripHeight - 2).AlignItems(ui.Center).Children(func() {
				a.tabContent(c, p, ws, t, active, color)
			})
		})
		if i < len(ws.tabs)-1 {
			ui.Row(c).Width(design.BorderWidth).FillHeight().Background(p.Divider)
		}
	})
	a.popoverMenu(c, p, tab, open, 180, []menuAction{
		{icon: icons.Pencil, text: "重命名请求", run: func() {
			if at := a.tabIndex(ws, t); at >= 0 {
				a.selectTab(ws, at)
			}
			t.startRename()
		}},
		{icon: icons.Copy, text: "复制当前标签页", run: func() { a.duplicateTabAt(ws, t) }},
		{icon: icons.Send, text: "分享标签页请求", run: func() { c.Toast("分享功能尚未实现") }},
		menuRule(),
		{icon: icons.Close, text: "关闭标签页", run: func() { a.closeTabAt(ws, t) }},
		{icon: icons.Stack, text: "关闭所有其他标签页", run: func() { a.closeTabsExcept(ws, t) }},
	})
}

// tabContent draws the method and the title, or the field that renames it.
// The field stands on its own rather than inside the button that selects
// the tab, which would take its clicks and its focus.
func (a *app) tabContent(c *ui.Context, p design.Palette, ws *workspace, t *tab, active bool, color ui.Color) {
	if t.renaming {
		ui.Row(c).Height(design.TabStripHeight-2).Padding(0, 10).Gap(6).
			AlignItems(ui.Center).Children(func() {
			ui.Text(c, t.method).Bold().FontSize(design.FontSizeTiny).
				TextColor(p.Method(t.method)).SingleLine()
			a.renameField(c, p, t)
		})
		return
	}
	// The tab takes the width its pieces need, the title shortened to keep it
	// inside the cap. The body's own width is a minimum rather than a width:
	// the toolkit measures the texts of an embedded font short, so left alone
	// the tab clamps to its minimum and the title eats the method beside it
	// ("PO…"); an explicit width would instead put the cross outside the tab's
	// box, where clicks miss it.
	chrome := requestTabChrome(c, t, active)
	label := tabName(c, t.title, design.RequestTabMaxWidth-chrome-tabWiggle)
	body := ui.ButtonBase(c).Height(design.TabStripHeight-2).
		Padding(0, 14).Gap(6).MinWidth(max(design.TabMinWidth, chrome+textWidth(c, label)+tabWiggle)).
		Label("标签页 " + t.title).Cursor(ui.CursorPointer)
	body.Children(func() {
		ui.Text(c, t.method).Bold().FontSize(design.FontSizeTiny).
			TextColor(p.Method(t.method)).SingleLine()
		ui.Text(c, label).FontSize(design.FontSizeBody).TextColor(color).SingleLine().
			MinWidth(textWidth(c, label) + tabWiggle)
		// The dot says the request is not on disk: a new tab, or one edited
		// since it was written. Ctrl+S writes it.
		if t.unsaved() {
			ui.Row(c).Size(6, 6).Radius(3).Background(p.TextMuted).
				Label("未保存 " + t.title)
		}
	})
	if body.RightClicked() {
		menu := a.menuFlag(requestTabMenuKey(t))
		*menu = !*menu
	}
	if body.Clicked() {
		if i := a.tabIndex(ws, t); i >= 0 {
			a.selectTab(ws, i)
		}
	}
	if active {
		close := ui.ButtonBase(c).Width(20).Height(20).Label("关闭标签页 " + t.title).
			Cursor(ui.CursorPointer).Children(func() {
			ui.Icon(c, icons.Close).FontSize(10).TextColor(p.TextMuted)
		})
		if close.Clicked() {
			a.closeTabAt(ws, t)
		}
	}
}

// renameField edits the tab's title in place: Enter or the check button
// keeps it, Escape or the cross puts the old title back.
func (a *app) renameField(c *ui.Context, p design.Palette, t *tab) {
	input := ui.TextInput(c, &t.title).Width(120).AutoFocus()
	input.OnShortcut(0, ui.KeyEscape, t.cancelRename)
	if input.Submitted() {
		a.commitTabRename(c, t)
		return
	}
	if ui.ButtonBase(c).Width(20).Height(20).Label("完成重命名").Cursor(ui.CursorPointer).
		Children(func() {
			ui.Icon(c, icons.Check).FontSize(11).TextColor(p.Accent)
		}).Clicked() {
		a.commitTabRename(c, t)
	}
	if ui.ButtonBase(c).Width(20).Height(20).Label("取消重命名").Cursor(ui.CursorPointer).
		Children(func() {
			ui.Icon(c, icons.Close).FontSize(10).TextColor(p.TextMuted)
		}).Clicked() {
		t.cancelRename()
	}
}

// newTabButton is the "+" that opens a request tab.
func (a *app) newTabButton(c *ui.Context, p design.Palette, ws *workspace) {
	if ui.ButtonBase(c).Width(32).Height(design.TabStripHeight).Label("新建标签页").
		Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Plus).FontSize(14).TextColor(p.TextMuted)
	}).Clicked() {
		a.addTab(ws)
	}
}

// environmentPicker lists the active collection's environments and opens
// the manager; the one picked here is the one requests send with. The trigger
// is a control of its own, bordered as the other fields are, and the active
// environment reads with the accent and a tick.
func (a *app) environmentPicker(c *ui.Context, p design.Palette) {
	label := "选择环境"
	if env := a.activeEnvironment(); env != nil {
		label = env.Name
	}
	btn := ui.ButtonBase(c).Width(120).Height(design.TabStripHeight-12).Padding(0, 10).
		MinWidth(0).Background(p.Surface).Radius(design.Radius).
		Border(design.BorderWidth, p.Border).Justify(ui.SpaceBetween).
		Label("环境：" + label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(p.SurfaceHover)
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(design.FontSizeBody).TextColor(p.TextSecondary).SingleLine()
		ui.Icon(c, icons.Chevron).FontSize(12).TextColor(p.TextMuted)
	})
	if btn.Clicked() {
		a.envMenuOpen = !a.envMenuOpen
	}
	ui.PopoverBase(c, btn, &a.envMenuOpen, func(panel ui.Element) {
		menuPanel(panel, p, 180)
		envs := a.collectionEnvironments()
		for _, env := range envs {
			row := menuRow(ui.ButtonBase(c).Height(28).Label("切换到 "+env.Name).Justify(ui.Start).
				Gap(8).Padding(0, 8).Cursor(ui.CursorPointer).Children(func() {
				text := p.TextSecondary
				if env.Active {
					// The one requests send with is ticked and reads in the
					// accent; the others keep its room, so names line up.
					ui.Icon(c, icons.Check).FontSize(12).TextColor(p.Accent)
					text = p.Accent
				} else {
					ui.Row(c).Width(12)
				}
				ui.Text(c, env.Name).FontSize(design.FontSizeBody).TextColor(text).SingleLine()
			}), p)
			if env.Active {
				row.Background(p.Accent.Alpha(accentTint))
			}
			if row.Clicked() {
				a.envMenuOpen = false
				if err := a.store.activateEnvironment(env.ID); err != nil {
					c.Toast("激活环境失败：" + err.Error())
				}
			}
		}
		if len(envs) == 0 {
			ui.Text(c, "还没有环境").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		}
		ui.Row(c).Height(design.BorderWidth).FillWidth().Background(p.Divider)
		manage := menuRow(ui.ButtonBase(c).Height(28).Label("管理环境").Justify(ui.Start).
			Padding(0, 8).Cursor(ui.CursorPointer).Children(func() {
			ui.Text(c, "管理环境…").FontSize(design.FontSizeBody).
				TextColor(p.TextSecondary).SingleLine()
		}), p)
		if manage.Clicked() {
			a.envMenuOpen = false
			a.openEnvManager()
		}
	})
}

// tabWiggle is the hair the toolkit rounds away when it lays a line out, kept
// so a title that just fits is not ellipsized to 「PO…」.
const tabWiggle float32 = 2

// requestTabChrome is how wide a request tab draws beside its title: both
// paddings, the method, the gap after each piece, the edit dot and the cross.
func requestTabChrome(c *ui.Context, t *tab, active bool) float32 {
	const (
		pad, gap   = 14 + 14, 6
		dot, cross = 6, 20
	)
	chrome := pad + measureText(c, t.method, design.FontSizeTiny, 700) + gap
	if t.unsaved() {
		chrome += gap + dot
	}
	if active {
		chrome += gap + cross
	}
	return chrome
}

package main

import (
	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// requestTabs draws the collection's request tabs and the button that
// opens another one, with the environment picker of the right edge. The row
// spans the window: the tabs scroll before the picker as they grow.
func (a *app) requestTabs(c *ui.Context, p design.Palette, ws *workspace) {
	ui.Row(c).Height(design.TabStripHeight).FillWidth().
		Background(p.Surface).AlignItems(ui.Start).Children(func() {
		ui.ScrollHorizontal(c).Grow(1).Height(design.TabStripHeight).Children(func() {
			ui.Row(c).Height(design.TabStripHeight).AlignItems(ui.Start).Children(func() {
				for i := range ws.tabs {
					a.tabItem(c, p, ws, i)
				}
				a.newTabButton(c, p, ws)
			})
		})
		a.environmentPicker(c, p)
		a.visibilityButton(c, p)
	})
}

// tabItem draws one request tab: its method, title, edit dot and close
// button, with the context menu Hoppscotch offers on a right click.
func (a *app) tabItem(c *ui.Context, p design.Palette, ws *workspace, i int) {
	t := ws.tabs[i]
	active := i == ws.active
	bg, line := p.Surface, p.Surface
	color := p.TextMuted
	if active {
		bg, line, color = p.Background, p.Accent, p.Text
	}

	tab := ui.Row(c).Height(design.TabStripHeight).Background(bg).AlignItems(ui.Start).
		ContextMenu(func(m *ui.Menu) { a.tabMenu(c, m, ws, i) })
	tab.Children(func() {
		ui.Column(c).Height(design.TabStripHeight).Children(func() {
			ui.Row(c).Height(2).FillWidth().Background(line)
			ui.Row(c).Height(design.TabStripHeight - 2).AlignItems(ui.Center).Children(func() {
				a.tabContent(c, p, ws, t, i, active, color)
			})
		})
		if i < len(ws.tabs)-1 {
			ui.Row(c).Width(design.BorderWidth).FillHeight().Background(p.Divider)
		}
	})
}

// tabContent draws the method and the title, or the field that renames it.
// The field stands on its own rather than inside the button that selects
// the tab, which would take its clicks and its focus.
func (a *app) tabContent(c *ui.Context, p design.Palette, ws *workspace, t *tab, i int, active bool, color ui.Color) {
	if t.renaming {
		ui.Row(c).Height(design.TabStripHeight-2).Padding(0, 10).Gap(6).
			AlignItems(ui.Center).Children(func() {
			ui.Text(c, t.method).Bold().FontSize(design.FontSizeTiny).
				TextColor(p.Method(t.method)).SingleLine()
			a.renameField(c, p, t)
		})
		return
	}
	body := ui.ButtonBase(c).Height(design.TabStripHeight-2).
		Padding(0, 10).Gap(6).Label("标签页 " + t.title)
	body.Children(func() {
		ui.Text(c, t.method).Bold().FontSize(design.FontSizeTiny).
			TextColor(p.Method(t.method)).SingleLine()
		ui.Text(c, t.title).FontSize(design.FontSizeBody).TextColor(color).SingleLine()
		if !active && t.marked() {
			ui.Row(c).Size(6, 6).Radius(3).Background(p.TextMuted)
		}
	})
	if body.Clicked() {
		a.selectTab(ws, i)
	}
	if active {
		close := ui.ButtonBase(c).Width(20).Height(20).Label("关闭标签页").
			Children(func() {
				ui.Icon(c, icons.Close).FontSize(10).TextColor(p.TextMuted)
			})
		if close.Clicked() {
			a.closeTab(ws, i)
		}
	}
}

// renameField edits the tab's title in place: Enter or the check button
// keeps it, Escape or the cross puts the old title back.
func (a *app) renameField(c *ui.Context, p design.Palette, t *tab) {
	input := ui.TextInput(c, &t.title).Width(120).AutoFocus()
	input.OnShortcut(0, ui.KeyEscape, t.cancelRename)
	if input.Submitted() {
		t.commitRename()
		return
	}
	if ui.ButtonBase(c).Width(20).Height(20).Label("完成重命名").Children(func() {
		ui.Icon(c, icons.Check).FontSize(11).TextColor(p.Accent)
	}).Clicked() {
		t.commitRename()
	}
	if ui.ButtonBase(c).Width(20).Height(20).Label("取消重命名").Children(func() {
		ui.Icon(c, icons.Close).FontSize(10).TextColor(p.TextMuted)
	}).Clicked() {
		t.cancelRename()
	}
}

// tabMenu is the context menu of a tab.
func (a *app) tabMenu(c *ui.Context, m *ui.Menu, ws *workspace, i int) {
	if m.Item("重命名请求").Shortcut(0, ui.KeyR).Chosen() {
		a.selectTab(ws, i)
		ws.tabs[i].startRename()
	}
	if m.Item("复制当前标签页").Shortcut(0, ui.KeyD).Chosen() {
		a.duplicateTab(ws, i)
	}
	if m.Item("分享标签页请求").Shortcut(0, ui.KeyS).Chosen() {
		c.Toast("分享功能尚未实现")
	}
	m.Separator()
	if m.Item("关闭标签页").Shortcut(0, ui.KeyW).Chosen() {
		a.closeTab(ws, i)
	}
	if m.Item("关闭所有其他标签页").Shortcut(0, ui.KeyX).Chosen() {
		a.closeOtherTabs(ws, i)
	}
}

// newTabButton is the "+" that opens a request tab.
func (a *app) newTabButton(c *ui.Context, p design.Palette, ws *workspace) {
	if ui.ButtonBase(c).Width(32).Height(design.TabStripHeight).Label("新建标签页").
		Children(func() {
			ui.Icon(c, icons.Plus).FontSize(14).TextColor(p.TextMuted)
		}).Clicked() {
		a.addTab(ws)
	}
}

// environmentPicker lists the active collection's environments and opens
// the manager; the one picked here is the one requests send with.
func (a *app) environmentPicker(c *ui.Context, p design.Palette) {
	label := "选择环境"
	if env := a.activeEnvironment(); env != nil {
		label = env.Name
	}
	btn := ui.ButtonBase(c).Height(design.TabStripHeight).Padding(0, 10).Gap(6).
		Label("环境：" + label)
	btn.Children(func() {
		ui.Text(c, label).FontSize(design.FontSizeBody).TextColor(p.TextSecondary).SingleLine()
		ui.Icon(c, icons.Chevron).FontSize(12).TextColor(p.TextMuted)
	})
	if btn.Clicked() {
		a.envMenuOpen = !a.envMenuOpen
	}
	ui.PopoverBase(c, btn, &a.envMenuOpen, func(panel ui.Element) {
		panel.Width(180).Padding(4).Radius(design.Radius).
			Background(p.Surface).Border(design.BorderWidth, p.Divider)
		envs := a.collectionEnvironments()
		for _, env := range envs {
			mark := "  "
			if env.Active {
				mark = "● "
			}
			if ui.ButtonBase(c).Height(28).Label("切换到 "+env.Name).Justify(ui.Start).
				Padding(0, 8).Children(func() {
				ui.Text(c, mark+env.Name).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
			}).Clicked() {
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
		if ui.ButtonBase(c).Height(28).Label("管理环境").Justify(ui.Start).
			Padding(0, 8).Children(func() {
			ui.Text(c, "管理环境…").FontSize(design.FontSizeBody).
				TextColor(p.TextSecondary).SingleLine()
		}).Clicked() {
			a.envMenuOpen = false
			a.openEnvManager()
		}
	})
}

// visibilityButton is the eye of the right edge; it will hide the
// environment's values in the panes.
func (a *app) visibilityButton(c *ui.Context, p design.Palette) {
	if ui.ButtonBase(c).Width(32).Height(design.TabStripHeight).Label("隐藏环境值").
		Children(func() {
			ui.Icon(c, icons.Eye).FontSize(14).TextColor(p.TextMuted)
		}).Clicked() {
		c.Toast("隐藏环境值功能尚未实现")
	}
}

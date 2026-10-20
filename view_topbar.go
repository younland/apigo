package main

import (
	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// topBar draws the home button and the open collections, which switch the
// whole workspace: a collection's request tabs, panel and environment go
// with it.
func (a *app) topBar(c *ui.Context, p design.Palette) {
	ui.Row(c).Height(design.TabStripHeight).FillWidth().
		Background(p.Surface).AlignItems(ui.Start).Padding(0, 4).Children(func() {
		a.homeButton(c, p)
		for _, id := range a.open {
			a.collectionTab(c, p, id)
		}
		a.moreButton(c, p)
		ui.Row(c).Grow(1)
		a.settingsButton(c, p)
	})
}

func (a *app) homeButton(c *ui.Context, p design.Palette) {
	selected := a.active == ""
	bg, color := p.Surface, p.TextMuted
	if selected {
		bg, color = p.Background, p.Text
	}
	btn := ui.ButtonBase(c).Height(design.TabStripHeight-8).Padding(0, 10).Gap(6).
		Label("主窗口").Background(bg).Children(func() {
		ui.Icon(c, icons.Home).FontSize(13).TextColor(color)
		ui.Text(c, "主窗口").FontSize(design.FontSizeBody).TextColor(color).SingleLine()
	})
	if btn.Clicked() {
		a.openHome()
	}
}

// collectionTab is one open collection: its name, and the cross that closes
// the workspace.
func (a *app) collectionTab(c *ui.Context, p design.Palette, id string) {
	i := a.collectionIndex(id)
	if i < 0 {
		return
	}
	name := a.store.collections[i].Name
	selected := a.active == id
	bg, color := p.Surface, p.TextMuted
	if selected {
		bg, color = p.Background, p.Text
	}
	tab := ui.Row(c).Height(design.TabStripHeight - 8).Background(bg).Radius(design.Radius).
		AlignItems(ui.Center).ContextMenu(func(m *ui.Menu) {
		if m.Item("关闭集合").Chosen() {
			a.closeCollection(id)
		}
		if m.Item("关闭其他集合").Chosen() {
			a.closeOtherCollections(id)
		}
	})
	tab.Children(func() {
		body := ui.ButtonBase(c).Height(design.TabStripHeight-8).Padding(0, 8).Gap(6).
			Label("集合标签 " + name).Children(func() {
			ui.Icon(c, icons.Folder).FontSize(12).TextColor(color)
			ui.Text(c, name).FontSize(design.FontSizeBody).TextColor(color).SingleLine()
		})
		if body.Clicked() {
			a.openCollection(id)
		}
		close := ui.ButtonBase(c).Width(20).Height(20).Label("关闭集合 " + name).
			Children(func() {
				ui.Icon(c, icons.Close).FontSize(10).TextColor(p.TextMuted)
			})
		if close.Clicked() {
			a.closeCollection(id)
		}
	})
}

// moreButton lists the recently closed collections; the collection tabs
// themselves answer a right click with close and close others. It holds no
// settings: the gear at the right end opens those.
func (a *app) moreButton(c *ui.Context, p design.Palette) {
	more := ui.ButtonBase(c).Width(32).Height(design.TabStripHeight - 8).Label("更多").
		Children(func() {
			ui.Icon(c, icons.More).FontSize(14).TextColor(p.TextMuted)
		})
	if more.Clicked() {
		a.moreOpen = !a.moreOpen
	}
	ui.PopoverBase(c, more, &a.moreOpen, func(panel ui.Element) {
		panel.Width(200).Padding(4).Radius(design.Radius).
			Background(p.Surface).Border(design.BorderWidth, p.Divider)
		ui.Text(c, "最近关闭").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		if len(a.closed) == 0 {
			ui.Text(c, "暂无最近关闭的集合").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		}
		for _, id := range a.closed {
			i := a.collectionIndex(id)
			if i < 0 {
				continue
			}
			name := a.store.collections[i].Name
			if ui.ButtonBase(c).Height(28).Label("重新打开 "+name).Justify(ui.Start).
				Padding(0, 8).Gap(6).Children(func() {
				ui.Icon(c, icons.Folder).FontSize(12).TextColor(p.TextMuted)
				ui.Text(c, name).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
			}).Clicked() {
				a.moreOpen = false
				a.openCollection(id)
			}
		}
	})
}

// settingsButton opens the settings dialog.
func (a *app) settingsButton(c *ui.Context, p design.Palette) {
	if ui.ButtonBase(c).Width(32).Height(design.TabStripHeight - 8).Label("设置").
		Children(func() {
			ui.Icon(c, icons.Gear).FontSize(14).TextColor(p.TextMuted)
		}).Clicked() {
		a.settingsOpen = true
	}
}

package app

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
		// The tabs scroll inside a strip of their own, so a collection is
		// named whole rather than squeezed into an ellipsis.
		noScrollbars(ui.ScrollHorizontal(c)).Height(design.TabStripHeight).Children(func() {
			ui.Row(c).Height(design.TabStripHeight).AlignItems(ui.Start).Shrink(0).Children(func() {
				// Built from a snapshot: a collection's own buttons close it
				// while the bar is still being drawn.
				for _, id := range append([]string(nil), a.open...) {
					a.collectionTab(c, p, id)
				}
			})
		})
		a.moreButton(c, p)
		ui.Row(c).Grow(1)
		a.settingsButton(c, p)
	})
}

func (a *app) homeButton(c *ui.Context, p design.Palette) {
	selected := a.active == ""
	bg, color := p.Surface, p.TextMuted
	if selected {
		bg, color = p.Background, p.Accent
	}
	btn := ui.ButtonBase(c).Height(design.TabStripHeight-8).Padding(0, 10).Gap(6).
		Label("主窗口").Background(bg).Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Home).FontSize(13).TextColor(color)
		ui.Text(c, "主窗口").FontSize(design.FontSizeBody).TextColor(color).SingleLine()
	})
	if btn.Clicked() {
		a.openHome()
	}
}

// tabChrome is the width of a collection tab beside its name: the padding of
// the button, its folder icon and gap, and the cross.
const tabChrome float32 = 8 + 8 + 12 + 6 + 20

// textWidth is how wide s draws in the interface font, in DIPs.
func textWidth(c *ui.Context, s string) float32 {
	return measureText(c, s, design.FontSizeBody, 400)
}

// measureText is how wide s draws in the interface font, at a size and weight.
func measureText(c *ui.Context, s string, size float32, weight int) float32 {
	w, _ := c.MeasureText(0, ui.Span{Text: s, Font: design.FontFamily, Size: size, Weight: weight})
	return w
}

// tabName keeps a tab inside its cap the way the tab would: the whole name
// while it fits, and its beginning with an ellipsis after.
func tabName(c *ui.Context, name string, max float32) string {
	if textWidth(c, name) <= max {
		return name
	}
	const ellipsis = "…"
	runes := []rune(name)
	for n := len(runes) - 1; n > 0; n-- {
		if cut := string(runes[:n]) + ellipsis; textWidth(c, cut) <= max {
			return cut
		}
	}
	return ellipsis
}

// collectionTab is one open collection: its name, and the cross that closes
// the workspace.
func (a *app) collectionTab(c *ui.Context, p design.Palette, id string) {
	i := a.collectionIndex(id)
	if i < 0 || !containsString(a.open, id) {
		return // gone, or closed while the bar was drawn
	}
	name := a.store.Collections[i].Name
	selected := a.active == id
	bg, color := p.Surface, p.TextMuted
	if selected {
		// The open collection is the accent one: its icon and name say which
		// workspace the window shows.
		bg, color = p.Background, p.Accent
	}
	tab := ui.Row(c).Height(design.TabStripHeight - 8).Background(bg).Radius(design.Radius).
		AlignItems(ui.Center).Shrink(0)
	open := a.menuFlag("tab/" + id)
	tab.Children(func() {
		// The name keeps a pixel more than it measures: the toolkit lays a line
		// out a hair wider than it measures, and would otherwise ellipsize a
		// name that just fits.
		label := tabName(c, name, design.CollectionTabMaxWidth-tabChrome)
		body := ui.ButtonBase(c).Height(design.TabStripHeight-8).Padding(0, 8).Gap(6).
			Label("集合标签 " + name).
			Cursor(ui.CursorPointer).Children(func() {
			ui.Icon(c, icons.Folder).FontSize(12).TextColor(color)
			ui.Text(c, label).FontSize(design.FontSizeBody).TextColor(color).SingleLine().
				Width(textWidth(c, label) + 1)
		})
		if body.RightClicked() {
			*open = !*open
		}
		if body.Clicked() {
			a.openCollection(id)
		}
		close := ui.ButtonBase(c).Width(20).Height(20).Label("关闭集合 " + name).
			Cursor(ui.CursorPointer).Children(func() {
			ui.Icon(c, icons.Close).FontSize(10).TextColor(p.TextMuted)
		})
		if close.Clicked() {
			a.askToCloseCollection(id)
		}
	})
	a.popoverMenu(c, p, tab, open, 160, []menuAction{
		{icon: icons.Close, text: "关闭集合", run: func() { a.askToCloseCollection(id) }},
		{icon: icons.Stack, text: "关闭其他集合", run: func() { a.closeOtherCollections(id) }},
	})
}

// moreButton lists the recently closed collections; the collection tabs
// themselves answer a right click with close and close others. It holds no
// settings: the gear at the right end opens those.
func (a *app) moreButton(c *ui.Context, p design.Palette) {
	more := ui.ButtonBase(c).Width(32).Height(design.TabStripHeight - 8).Label("更多").
		Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.More).FontSize(14).TextColor(p.TextMuted)
	})
	if more.Clicked() {
		a.moreOpen = !a.moreOpen
	}
	ui.PopoverBase(c, more, &a.moreOpen, func(panel ui.Element) {
		menuPanel(panel, p, 200)
		ui.Text(c, "最近关闭").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		if len(a.closed) == 0 {
			ui.Text(c, "暂无最近关闭的集合").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		}
		for _, id := range append([]string(nil), a.closed...) {
			i := a.collectionIndex(id)
			if i < 0 || !containsString(a.closed, id) {
				continue // gone, or reopened while the menu was drawn
			}
			name := a.store.Collections[i].Name
			row := menuRow(ui.ButtonBase(c).Height(28).Label("重新打开 "+name).Justify(ui.Start).
				Padding(0, 8).Gap(6).Cursor(ui.CursorPointer).Children(func() {
				ui.Icon(c, icons.Folder).FontSize(12).TextColor(p.TextMuted)
				ui.Text(c, name).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
			}), p)
			if row.Clicked() {
				a.moreOpen = false
				a.openCollection(id)
			}
		}
	})
}

// settingsButton opens the settings.
func (a *app) settingsButton(c *ui.Context, p design.Palette) {
	if ui.ButtonBase(c).Width(32).Height(design.TabStripHeight - 8).Label("设置").
		Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Gear).FontSize(14).TextColor(p.TextMuted)
	}).Clicked() {
		a.settingsOpen = true
	}
}

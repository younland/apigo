package main

import (
	"strings"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// homeView lists the collections, and leads to making the first one when
// there is none: the app opens on this window, not on a request. The title and
// the list take two thirds of the width, centered, so a wide window leaves the
// rows reading as lines rather than as rules across the whole of it.
func (a *app) homeView(c *ui.Context, p design.Palette) {
	ui.Row(c).Grow(1).FillWidth().Padding(32).AlignItems(ui.Start).Children(func() {
		ui.Row(c).Grow(1)
		ui.Column(c).Grow(4).FillHeight().Gap(design.RowGap).Children(func() {
			a.homeHeader(c, p)
			if len(a.store.collections) == 0 {
				a.homeEmpty(c, p)
				return
			}
			// The list's bar is hidden: it scrolls by the wheel, and a bar of its
			// own would sit at the edge of the column.
			noScrollbars(ui.Scroll(c)).Grow(1).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Gap(2).Children(func() {
					// Built from a snapshot: a row's own menu deletes collections
					// while the list is still being drawn.
					for _, col := range append([]collection(nil), a.store.collections...) {
						if a.collectionIndex(col.ID) < 0 {
							continue // deleted while the list was drawn
						}
						a.homeCollectionRow(c, p, col)
					}
				})
			})
		})
		ui.Row(c).Grow(1)
	})
}

// homeHeader draws the title and the actions of the home window. The
// actions belong to a list that exists: the empty window leads with its own
// buttons instead.
func (a *app) homeHeader(c *ui.Context, p design.Palette) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(design.RowGap).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, "接口管理").Bold().FontSize(20).TextColor(p.Text)
			ui.Text(c, "每个集合分组一组接口，并管理它自己的环境与变量").
				FontSize(design.FontSizeBody).TextColor(p.TextMuted)
		})
		ui.Row(c).Grow(1)
		if len(a.store.collections) == 0 {
			return
		}
		if ui.Button(c, "导入集合").Cursor(ui.CursorPointer).Clicked() {
			c.Toast("导入集合尚未实现：后续支持 Postman、在线 Swagger、OpenAPI v2/v3")
		}
		if ui.PrimaryButton(c, "新建集合").Cursor(ui.CursorPointer).Clicked() {
			a.openCollectionModal()
		}
	})
	a.divider(c, p)
}

// homeEmpty leads the way when no collection exists yet: one clear action,
// and what the first collection comes with.
func (a *app) homeEmpty(c *ui.Context, p design.Palette) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(12).Children(func() {
		ui.Row(c).Size(64, 64).Radius(16).Background(p.Surface).Center().Children(func() {
			ui.Icon(c, icons.Folder).FontSize(28).TextColor(p.Accent)
		})
		ui.Text(c, "还没有集合").FontSize(design.FontSizeDisplay).Bold().TextColor(p.Text)
		ui.Text(c, "集合把接口分组在一起，并带上自己的环境与变量").
			FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
		ui.Row(c).Gap(design.RowGap).Children(func() {
			if ui.PrimaryButton(c, "新建集合").Cursor(ui.CursorPointer).Clicked() {
				a.openCollectionModal()
			}
			if ui.Button(c, "从集合文件导入").Cursor(ui.CursorPointer).Clicked() {
				c.Toast("导入集合尚未实现：后续支持 Postman、在线 Swagger、OpenAPI v2/v3")
			}
		})
		ui.Text(c, "新建的集合会自带「本地 / 测试 / 生产」三个环境和 BaseURL 变量").
			FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
	})
}

// collectionModal asks for the name of a collection: a new one, as the
// flow's first step, or the new name of the one being renamed. It opens at
// the top of the window, centered, rather than in the middle: the list it
// makes stays in view.
func (a *app) collectionModal(c *ui.Context, p design.Palette) {
	if !a.collectionOpen {
		return
	}
	title := "新建集合"
	if a.collectionEdit != "" {
		title = "重命名集合"
	}
	ui.DialogBase(c, &a.collectionOpen, func(backdrop, panel ui.Element) {
		topBackdrop(backdrop)
		panel.Width(420).Padding(20).Gap(12).Radius(8).
			Background(p.Background).Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1)
			ui.Text(c, title).Bold().FontSize(16).TextColor(p.Text)
			ui.Row(c).Grow(1)
			if ui.ButtonBase(c).Width(24).Height(24).Label("关闭").Cursor(ui.CursorPointer).Children(func() {
				ui.Icon(c, icons.Close).FontSize(12).TextColor(p.TextMuted)
			}).Clicked() {
				a.collectionOpen = false
			}
		})
		a.divider(c, p)
		ui.Text(c, "标签").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		input := ui.TextInput(c, &a.collectionName).Label("集合名称")
		if input.Submitted() {
			a.saveNewCollection(c)
		}
		ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1)
			if ui.PrimaryButton(c, "保存").Cursor(ui.CursorPointer).
				Disabled(strings.TrimSpace(a.collectionName) == "").Clicked() {
				a.saveNewCollection(c)
			}
			if ui.Button(c, "取消").Cursor(ui.CursorPointer).Clicked() {
				a.collectionOpen = false
			}
		})
	})
}

// homeCollectionRow is one collection of the list: its icon, name and the
// kinds it holds, with the menu that opens or deletes it. The row is the
// collection's own target: the pointer tints it, and a double click opens it,
// as a double click opens a file.
func (a *app) homeCollectionRow(c *ui.Context, p design.Palette, col collection) {
	row := ui.Row(c).FillWidth().Height(52).AlignItems(ui.Center).Gap(12).
		Padding(0, 8).Radius(design.Radius).Cursor(ui.CursorPointer)
	row.Children(func() { a.homeCollectionBody(c, p, col) })
	if row.Hovered() {
		row.Background(p.SurfaceHover)
	}
	if row.DoubleClicked() {
		a.openCollection(col.ID)
	}
	a.divider(c, p)
}

// homeCollectionBody draws what a collection row holds: its icon, name and the
// kinds it holds, with the click that opens it and the menu of its actions.
func (a *app) homeCollectionBody(c *ui.Context, p design.Palette, col collection) {
	ui.Row(c).Size(32, 32).Radius(8).Background(p.SurfaceHover).Center().Children(func() {
		ui.Icon(c, icons.Folder).FontSize(16).TextColor(p.Accent)
	})
	ui.Column(c).Gap(2).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, col.Name).FontSize(design.FontSizeTitle).TextColor(p.Text).SingleLine()
			ui.Text(c, "HTTP").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		})
		ui.Textf(c, "%d 个接口", countRequests(col.Root)).
			FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
	})
	ui.Row(c).Grow(1)
	open := ui.ButtonBase(c).Height(30).Padding(0, 10).Label("打开 " + col.Name).
		Cursor(ui.CursorPointer).Children(func() {
		ui.Text(c, "打开").FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
	})
	if open.Clicked() {
		a.openCollection(col.ID)
	}
	more := ui.ButtonBase(c).Width(28).Height(28).Label("集合操作 " + col.Name).
		Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.More).FontSize(14).TextColor(p.TextMuted)
	})
	menu := a.menuFlag("collection/" + col.ID)
	if more.Clicked() {
		*menu = !*menu
	}
	a.popoverMenu(c, p, more, menu, 160, []menuAction{
		{icon: icons.External, text: "打开文件夹", run: func() {
			if err := a.openFolder(a.store.collectionDir(col.ID)); err != nil {
				c.Toast("打开文件夹失败：" + err.Error())
			}
		}},
		{icon: icons.Folder, text: "打开集合", run: func() { a.openCollection(col.ID) }},
		{icon: icons.Pencil, text: "重命名集合", run: func() { a.openRenameModal(col.ID) }},
		menuRule(),
		{icon: icons.Trash, text: "删除集合", run: func() { a.askToDeleteCollection(col.ID) }},
	})
}

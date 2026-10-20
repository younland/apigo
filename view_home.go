package main

import (
	"strings"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// homeView lists the collections, and leads to making the first one when
// there is none: the app opens on this window, not on a request.
func (a *app) homeView(c *ui.Context, p design.Palette) {
	ui.Column(c).Grow(1).FillWidth().Padding(32).Gap(design.RowGap).Children(func() {
		a.homeHeader(c, p)
		if len(a.store.collections) == 0 {
			a.homeEmpty(c, p)
			return
		}
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				for i := range a.store.collections {
					a.homeCollectionRow(c, p, a.store.collections[i])
				}
			})
		})
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
		if ui.Button(c, "导入集合").Clicked() {
			c.Toast("导入集合尚未实现：后续支持 Postman、在线 Swagger、OpenAPI v2/v3")
		}
		if ui.PrimaryButton(c, "新建集合").Clicked() {
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
			if ui.PrimaryButton(c, "新建集合").Clicked() {
				a.openCollectionModal()
			}
			if ui.Button(c, "从集合文件导入").Clicked() {
				c.Toast("导入集合尚未实现：后续支持 Postman、在线 Swagger、OpenAPI v2/v3")
			}
		})
		ui.Text(c, "新建的集合会自带「本地 / 测试 / 生产」三个环境和 BaseURL 变量").
			FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
	})
}

// collectionModal asks for the name of a new collection, as the flow's
// first step, and saves it with 保存. It opens at the top of the window,
// centered, rather than in the middle: the list it makes stays in view.
func (a *app) collectionModal(c *ui.Context, p design.Palette) {
	if !a.collectionOpen {
		return
	}
	ui.DialogBase(c, &a.collectionOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4)).Justify(ui.Start).
			Padding(design.PanelPadding*3, 0, 0, 0)
		panel.Width(420).Padding(20).Gap(12).Radius(8).
			Background(p.Background).Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1)
			ui.Text(c, "新建集合").Bold().FontSize(16).TextColor(p.Text)
			ui.Row(c).Grow(1)
			if ui.ButtonBase(c).Width(24).Height(24).Label("关闭").Children(func() {
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
			if ui.PrimaryButton(c, "保存").
				Disabled(strings.TrimSpace(a.collectionName) == "").Clicked() {
				a.saveNewCollection(c)
			}
			if ui.Button(c, "取消").Clicked() {
				a.collectionOpen = false
			}
		})
	})
}

// homeCollectionRow is one collection of the list: its icon, name and the
// kinds it holds, with the menu that opens or deletes it.
func (a *app) homeCollectionRow(c *ui.Context, p design.Palette, col collection) {
	ui.Row(c).FillWidth().Height(52).AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Row(c).Size(32, 32).Radius(8).Background(p.SurfaceHover).Children(func() {
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
			Children(func() {
				ui.Text(c, "打开").FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
			})
		if open.Clicked() {
			a.openCollection(col.ID)
		}
		more := ui.ButtonBase(c).Width(28).Height(28).Label("集合操作 " + col.Name).
			Children(func() {
				ui.Icon(c, icons.More).FontSize(14).TextColor(p.TextMuted)
			})
		more.Menu(func(m *ui.Menu) {
			if m.Item("打开集合").Chosen() {
				a.openCollection(col.ID)
			}
			m.Separator()
			if m.Item("删除集合").Chosen() {
				if err := a.deleteCollection(col.ID); err != nil {
					c.Toast("删除失败：" + err.Error())
				}
			}
		})
	})
	a.divider(c, p)
}

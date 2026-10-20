package main

import (
	"strings"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// requestsPanel is the left panel of a collection: its interfaces as a tree
// of folders and requests, the search and kind filter over them, and the
// history.
func (a *app) requestsPanel(c *ui.Context, p design.Palette) {
	col := a.activeCollection()
	ui.Column(c).Width(design.SidebarWidth).FillHeight().Background(p.Background).Children(func() {
		ui.Row(c).Height(design.HeaderHeight).FillWidth().
			Padding(0, design.PanelPadding).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "接口管理").Bold().FontSize(design.FontSizeBody).TextColor(p.Text)
			ui.TextInput(c, &a.search).Grow(1).Height(26).Placeholder("搜索接口")
			a.panelModeButton(c, p)
		})
		a.divider(c, p)
		a.requestFilters(c, p)
		a.divider(c, p)
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(design.RowGap, design.PanelPadding).
				Gap(2).Children(func() {
				if a.panelMode == panelHistory {
					a.historyPanel(c, p)
					return
				}
				a.requestTree(c, p, col)
			})
		})
	})
}

// activeCollection is the collection the workspace belongs to, zero in the
// home window.
func (a *app) activeCollection() collection {
	if i := a.collectionIndex(a.active); i >= 0 {
		return a.store.collections[i]
	}
	return collection{}
}

// panelModeButton switches the panel between requests and history.
func (a *app) panelModeButton(c *ui.Context, p design.Palette) {
	icon, label := icons.Clock, "历史"
	if a.panelMode == panelHistory {
		icon, label = icons.Folder, "接口"
	}
	if ui.ButtonBase(c).Width(24).Height(24).Label(label).Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(p.TextMuted)
	}).Clicked() {
		if a.panelMode == panelHistory {
			a.panelMode = panelRequests
			return
		}
		a.panelMode = panelHistory
	}
}

// requestFilters draws the kind filter, the folder button and the button
// that adds an interface of a kind.
func (a *app) requestFilters(c *ui.Context, p design.Palette) {
	ui.Row(c).FillWidth().Padding(design.RowGap, design.PanelPadding).
		Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1)
		filter := ui.ButtonBase(c).Width(26).Height(26).Label("筛选类型").Children(func() {
			ui.Icon(c, icons.Sliders).FontSize(13).TextColor(p.TextMuted)
		})
		if filter.Clicked() {
			a.filterOpen = !a.filterOpen
		}
		ui.PopoverBase(c, filter, &a.filterOpen, func(panel ui.Element) {
			panel.Width(160).Padding(4).Radius(design.Radius).
				Background(p.Surface).Border(design.BorderWidth, p.Divider)
			if ui.ButtonBase(c).Height(28).Label("全部类型").Justify(ui.Start).
				Padding(0, 8).Children(func() {
				ui.Text(c, "全部类型").FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
			}).Clicked() {
				a.kindFilter, a.filterOpen = "", false
			}
			a.kindMenu(c, p, func(k kind) { a.kindFilter, a.filterOpen = k, false })
		})
		if ui.ButtonBase(c).Width(26).Height(26).Label("新建文件夹").Children(func() {
			ui.Icon(c, icons.Folder).FontSize(13).TextColor(p.TextMuted)
		}).Clicked() {
			if err := a.newFolder(); err != nil {
				c.Toast("新建文件夹失败：" + err.Error())
			}
		}
		add := ui.ButtonBase(c).Width(26).Height(26).Label("新建接口").Children(func() {
			ui.Icon(c, icons.Plus).FontSize(14).TextColor(p.Accent)
		})
		if add.Clicked() {
			a.kindMenuOpen = !a.kindMenuOpen
		}
		ui.PopoverBase(c, add, &a.kindMenuOpen, func(panel ui.Element) {
			panel.Width(160).Padding(4).Radius(design.Radius).
				Background(p.Surface).Border(design.BorderWidth, p.Divider)
			a.kindMenu(c, p, func(k kind) {
				a.kindMenuOpen = false
				if err := a.addRequest(k, ""); err != nil {
					c.Toast(err.Error())
				}
			})
		})
	})
}

// requestTree draws the interfaces of a collection: its 根目录, the folders
// inside it, and their requests.
func (a *app) requestTree(c *ui.Context, p design.Palette, col collection) {
	a.folderRow(c, p, col, col.Root, 0)
}

// folderRow draws one folder of the tree and, when open, what it holds: the
// folders inside it first, then its requests.
func (a *app) folderRow(c *ui.Context, p design.Palette, col collection, f folder, depth int) {
	open := !a.foldersClosed[f.Path]
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Row(c).Height(28).FillWidth().Padding(0, 0, 0, depthIndent(depth)).
			AlignItems(ui.Center).Gap(6).Children(func() {
			chevron := ui.Icon(c, icons.Chevron).FontSize(11).TextColor(p.TextMuted)
			if !open {
				chevron.Rotate(-90)
			}
			ui.Icon(c, icons.Folder).FontSize(13).TextColor(p.TextSecondary)
			if ui.ButtonBase(c).Grow(1).Height(28).Label("文件夹 " + f.Name).
				Justify(ui.Start).Gap(6).Children(func() {
				ui.Text(c, f.Name).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
				ui.Textf(c, "%d", countRequests(f)).FontSize(design.FontSizeTiny).
					TextColor(p.TextMuted)
			}).Clicked() {
				if a.foldersClosed == nil {
					a.foldersClosed = map[string]bool{}
				}
				a.foldersClosed[f.Path] = open
			}
			a.folderAddButton(c, p, f)
		})
		if !open {
			return
		}
		for _, sub := range f.Folders {
			a.folderRow(c, p, col, sub, depth+1)
		}
		for _, req := range f.Requests {
			if a.matchesFilter(req) {
				a.requestLineRow(c, p, col, req, depth+1)
			}
		}
		if depth == 0 && len(f.Folders) == 0 && len(f.Requests) == 0 {
			a.emptyTreeHint(c, p, depth+1)
		}
	})
}

// emptyTreeHint says how the first interface of the tree is made.
func (a *app) emptyTreeHint(c *ui.Context, p design.Palette, depth int) {
	ui.Row(c).FillWidth().Padding(0, 0, 0, depthIndent(depth)).Children(func() {
		ui.Text(c, "还没有接口，点上面的 + 新建").FontSize(design.FontSizeBody).
			TextColor(p.TextMuted).SingleLine()
	})
}

// depthIndent is how far one level of the tree moves a row right.
func depthIndent(depth int) float32 { return float32(depth) * design.PanelPadding }

// folderAddButton adds an interface of a kind inside a folder.
func (a *app) folderAddButton(c *ui.Context, p design.Palette, f folder) {
	open := a.folderMenuFlag(f.Path)
	add := ui.ButtonBase(c).Width(20).Height(20).Label("在" + f.Name + "里新建接口").
		Children(func() {
			ui.Icon(c, icons.Plus).FontSize(11).TextColor(p.TextMuted)
		})
	if add.Clicked() {
		*open = !*open
	}
	ui.PopoverBase(c, add, open, func(panel ui.Element) {
		panel.Width(160).Padding(4).Radius(design.Radius).
			Background(p.Surface).Border(design.BorderWidth, p.Divider)
		a.kindMenu(c, p, func(k kind) {
			*open = false
			if err := a.addRequest(k, f.Path); err != nil {
				c.Toast(err.Error())
			}
		})
	})
}

// folderMenuFlag is the open state of a folder's add menu.
func (a *app) folderMenuFlag(id string) *bool {
	if a.folderMenus == nil {
		a.folderMenus = map[string]*bool{}
	}
	flag := a.folderMenus[id]
	if flag == nil {
		flag = new(bool)
		a.folderMenus[id] = flag
	}
	return flag
}

// matchesFilter reports whether a request passes the search and the kind.
func (a *app) matchesFilter(req savedRequest) bool {
	if a.kindFilter != "" && req.Kind != a.kindFilter {
		return false
	}
	query := strings.TrimSpace(a.search)
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(req.Name), strings.ToLower(query))
}

// requestLineRow is one clickable request of the tree: its kind, method and
// name, indented under the folder it is in.
func (a *app) requestLineRow(c *ui.Context, p design.Palette, col collection, req savedRequest, depth int) {
	label, color := req.Method, p.Method(req.Method)
	if req.Kind != kindREST {
		label, color = kindName(req.Kind), p.TextMuted
	}
	ui.Row(c).Height(26).FillWidth().Padding(0, 0, 0, depthIndent(depth)).
		AlignItems(ui.Center).Gap(6).Children(func() {
		ui.Icon(c, kindIcon(req.Kind)).FontSize(11).TextColor(color)
		ui.Text(c, label).Bold().FontSize(design.FontSizeTiny).TextColor(color).SingleLine()
		if ui.ButtonBase(c).Grow(1).Height(26).Label("请求 " + req.Name).
			Justify(ui.Start).Children(func() {
			ui.Text(c, req.Name).FontSize(design.FontSizeBody).
				TextColor(p.TextSecondary).SingleLine()
		}).Clicked() {
			a.openRequest(col.ID, req)
		}
	})
}

// historyPanel lists the sent requests, newest first; a click opens one.
func (a *app) historyPanel(c *ui.Context, p design.Palette) {
	if len(a.store.history) == 0 {
		ui.Text(c, "还没有请求历史").FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
		return
	}
	for _, entry := range a.store.history {
		a.historyRow(c, p, entry)
	}
}

func (a *app) historyRow(c *ui.Context, p design.Palette, entry historyEntry) {
	ui.Row(c).Height(26).FillWidth().AlignItems(ui.Center).Gap(6).Children(func() {
		ui.Text(c, entry.Method).Bold().FontSize(design.FontSizeTiny).
			TextColor(p.Method(entry.Method)).SingleLine()
		if ui.ButtonBase(c).Grow(1).Height(26).Label("历史 " + entry.URL).
			Justify(ui.Start).Children(func() {
			ui.Text(c, entry.URL).FontSize(design.FontSizeBody).
				TextColor(p.TextSecondary).SingleLine()
		}).Clicked() {
			a.openHistory(entry)
		}
		if entry.Status != 0 {
			ui.Textf(c, "%d", entry.Status).FontSize(design.FontSizeTiny).
				TextColor(p.Status(entry.Status))
		}
	})
}

package main

import (
	"strings"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// requestsPanel is the left panel of a collection: the search and the
// history over its interfaces, which are a tree of folders and requests.
func (a *app) requestsPanel(c *ui.Context, p design.Palette) {
	col := a.activeCollection()
	ui.Column(c).Width(design.SidebarWidth).FillHeight().Background(p.Background).
		Label("接口面板").Children(func() {
		ui.Row(c).Height(design.HeaderHeight).FillWidth().
			Padding(0, design.PanelPadding).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &a.search).Grow(1).Height(26).Placeholder("搜索接口")
			a.panelModeButton(c, p)
		})
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
	if ui.ButtonBase(c).Width(24).Height(24).Label(label).Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(p.TextMuted)
	}).Clicked() {
		if a.panelMode == panelHistory {
			a.panelMode = panelRequests
			return
		}
		a.panelMode = panelHistory
	}
}

// kindFilterButton picks the kind the tree lists, "" for every kind.
func (a *app) kindFilterButton(c *ui.Context, p design.Palette) {
	filter := ui.ButtonBase(c).Width(22).Height(22).Label("筛选类型").Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Sliders).FontSize(15).TextColor(p.TextMuted)
	})
	if filter.Clicked() {
		a.filterOpen = !a.filterOpen
	}
	ui.PopoverBase(c, filter, &a.filterOpen, func(panel ui.Element) {
		menuPanel(panel, p, 160)
		row := menuRow(ui.ButtonBase(c).Height(28).Label("全部类型").Justify(ui.Start).
			Padding(0, 8).Cursor(ui.CursorPointer).Children(func() {
			ui.Text(c, "全部类型").FontSize(design.FontSizeBody).
				TextColor(p.TextSecondary).SingleLine()
		}), p)
		if row.Clicked() {
			a.kindFilter, a.filterOpen = "", false
		}
		a.kindMenu(c, p, func(k kind) { a.kindFilter, a.filterOpen = k, false })
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
	menu := a.menuFlag("folder-menu/" + f.Path)
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		row := ui.Row(c).Height(28).FillWidth().Padding(0, 0, 0, depthIndent(depth)).
			Radius(design.Radius).AlignItems(ui.Center).Gap(6).Cursor(ui.CursorPointer)
		row.Children(func() {
			// The whole row is the button: every part of it folds the folder,
			// and the + keeps its own clicks.
			body := ui.ButtonBase(c).Grow(1).Height(28).Label("文件夹 " + f.Name).
				Justify(ui.Start).Gap(4).Cursor(ui.CursorPointer)
			body.Children(func() {
				chevron := ui.Icon(c, icons.Chevron).FontSize(11).TextColor(p.TextMuted)
				if !open {
					chevron.Rotate(-90)
				}
				ui.Icon(c, icons.Folder).FontSize(13).TextColor(p.TextSecondary)
				ui.Text(c, f.Name).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
				ui.Textf(c, "%d", countRequests(f)).FontSize(design.FontSizeTiny).
					TextColor(p.TextMuted)
			})
			if body.Clicked() {
				if a.foldersClosed == nil {
					a.foldersClosed = map[string]bool{}
				}
				a.foldersClosed[f.Path] = open
			}
			// The buttons at the row's end sit close together: three icons a gap
			// apart read as three separate things.
			ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
				if depth == 0 {
					a.kindFilterButton(c, p)
				}
				a.folderAddButton(c, p, f)
				// The 根目录 is the collection's own directory: nothing renames,
				// moves or deletes it, so it has no menu to open.
				if f.Path == "" {
					return
				}
				more := moreButton(c, p, "文件夹操作 "+f.Name)
				if more.Clicked() {
					*menu = !*menu
				}
				a.popoverMenu(c, p, more, menu, 160, a.folderActions(f))
			})
		})
		if row.Hovered() {
			row.Background(p.SurfaceHover)
		}
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

// folderAddButton opens a folder's add menu: a subfolder of its own, or an
// interface of a kind inside it.
func (a *app) folderAddButton(c *ui.Context, p design.Palette, f folder) {
	open := a.menuFlag("folder/" + f.Path)
	add := ui.ButtonBase(c).Width(18).Height(18).Label("在" + f.Name + "里新建接口").
		Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Plus).FontSize(12).TextColor(p.TextMuted)
	})
	if add.Clicked() {
		*open = !*open
	}
	ui.PopoverBase(c, add, open, func(panel ui.Element) {
		menuPanel(panel, p, 160)
		row := menuRow(ui.ButtonBase(c).Height(28).Label("在"+f.Name+"里新建文件夹").Justify(ui.Start).
			Padding(0, 8).Gap(8).Cursor(ui.CursorPointer).Children(func() {
			ui.Icon(c, icons.Folder).FontSize(13).TextColor(p.TextMuted)
			ui.Text(c, "新建文件夹").FontSize(design.FontSizeBody).
				TextColor(p.TextSecondary).SingleLine()
		}), p)
		if row.Clicked() {
			*open = false
			a.askNewFolder(f.Path)
		}
		ui.Row(c).Height(design.BorderWidth).FillWidth().Background(p.Divider)
		a.kindMenu(c, p, func(k kind) {
			*open = false
			if err := a.addRequest(k, f.Path); err != nil {
				c.Toast(err.Error())
			}
		})
	})
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
// name, indented under the folder it is in, with the menu a right click opens.
// The one a tab shows is tinted with the accent; any row tints under the
// pointer.
func (a *app) requestLineRow(c *ui.Context, p design.Palette, col collection, req savedRequest, depth int) {
	label, color := req.Method, p.Method(req.Method)
	if req.Kind != kindREST {
		label, color = kindName(req.Kind), p.TextMuted
	}
	opened := a.activeShows(req.Path)
	name, bg := p.TextSecondary, ui.Color{}
	if opened {
		name, bg = p.Accent, p.Accent.Alpha(accentTint)
	}
	menu := a.menuFlag("request-menu/" + req.Path)
	row := ui.Row(c).Height(26).FillWidth().Padding(0, 0, 0, depthIndent(depth)).
		Radius(design.Radius).AlignItems(ui.Center).Gap(6).Background(bg).
		Cursor(ui.CursorPointer)
	row.Children(func() {
		// The whole row is the button: its kind, method and name all open it.
		body := ui.ButtonBase(c).Grow(1).Height(26).Label("请求 " + req.Name).
			Justify(ui.Start).Gap(6).Cursor(ui.CursorPointer)
		body.Children(func() {
			ui.Icon(c, kindIcon(req.Kind)).FontSize(11).TextColor(color)
			ui.Text(c, label).Bold().FontSize(design.FontSizeTiny).TextColor(color).SingleLine()
			ui.Text(c, req.Name).FontSize(design.FontSizeBody).TextColor(name).SingleLine()
		})
		if body.Clicked() {
			a.openRequest(col.ID, req)
		}
		more := moreButton(c, p, "接口操作 "+req.Name)
		if more.Clicked() {
			*menu = !*menu
		}
		a.popoverMenu(c, p, more, menu, 170, []menuAction{
			{icon: icons.Pencil, text: "重命名", run: func() { a.askRenameRequest(req.Path) }},
			{icon: icons.Copy, text: "复制接口", run: func() { a.duplicateRequest(c, req.Path) }},
			{icon: icons.Folder, text: "移动接口", run: func() { a.askToMoveRequest(req.Path) }},
			menuRule(),
			{icon: icons.Trash, text: "删除接口", danger: true, run: func() {
				a.askToDeleteRequest(req.Path)
			}},
		})
	})
	if !opened && row.Hovered() {
		row.Background(p.SurfaceHover)
	}
}

// moreButton is the "⋮" a row of the tree opens its menu with, standing on its
// end as the rows stack, and kept after the row's +.
func moreButton(c *ui.Context, p design.Palette, label string) ui.Element {
	return ui.ButtonBase(c).Width(18).Height(18).Label(label).Cursor(ui.CursorPointer).
		Children(func() {
			ui.Icon(c, icons.MoreVertical).FontSize(13).TextColor(p.TextMuted)
		})
}

// folderActions are the lines of a folder's menu: its name, where it goes, and
// deleting it with everything it holds.
func (a *app) folderActions(f folder) []menuAction {
	if f.Path == "" {
		return nil // the 根目录 is the collection's own directory
	}
	return []menuAction{
		{icon: icons.Pencil, text: "重命名", run: func() { a.askRenameFolder(f.Path) }},
		{icon: icons.Folder, text: "移动文件夹", run: func() { a.askToMoveFolder(f.Path) }},
		menuRule(),
		{icon: icons.Trash, text: "删除文件夹", danger: true, run: func() {
			a.askToDeleteFolder(f.Path)
		}},
	}
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
			Justify(ui.Start).Cursor(ui.CursorPointer).Children(func() {
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

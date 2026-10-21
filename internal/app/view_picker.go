package app

import (
	"strings"

	"apigo/internal/design"
	"apigo/internal/icons"
	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// folderModal asks where something of the tree goes, and what it is called when
// it names it: the folders of the collection, 根目录 first, each chosen with a
// click, and the button that goes ahead with what runs on the answers.
func (a *app) folderModal(c *ui.Context, p design.Palette) {
	q := &a.where
	if !q.open {
		return
	}
	ui.DialogBase(c, &q.open, func(backdrop, panel ui.Element) {
		topBackdrop(backdrop)
		panel.Width(420).Padding(20).Gap(12).Radius(8).
			Background(p.Background).Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
		ui.Text(c, q.title).Bold().FontSize(design.FontSizeDisplay).TextColor(p.Text)
		if q.names {
			ui.TextInput(c, &q.name).Label(q.nameTitle).Placeholder(q.nameTitle).AutoFocus()
			if strings.TrimSpace(q.name) == "" {
				ui.Text(c, "名称不能为空").FontSize(design.FontSizeTiny).TextColor(p.Danger)
			}
		}
		if len(q.paths) > 1 {
			ui.Text(c, "选择存放的文件夹").FontSize(design.FontSizeBody).TextColor(p.TextMuted)
		}
		ui.Scroll(c).MaxHeight(280).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				for _, path := range q.paths {
					a.folderChoice(c, p, q, path)
				}
			})
		})
		ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1)
			if ui.Button(c, "取消").Cursor(ui.CursorPointer).Clicked() {
				q.open = false
			}
			if folderButton(c, p, q).Clicked() {
				run := q.run
				if run == nil {
					return
				}
				if err := run(c, q.folder, strings.TrimSpace(q.name)); err != nil {
					// What it asked for was refused: the dialog stays to be set
					// right, and says why.
					c.Toast(err.Error())
					return
				}
				q.open, q.run = false, nil
			}
		})
	})
}

// folderChoice is one folder of the question: indented under the folder it is
// in, and filled with the accent while it is the choice.
func (a *app) folderChoice(c *ui.Context, p design.Palette, q *folderQuestion, path string) {
	selected := q.folder == path
	color, bg := p.TextSecondary, ui.Color{}
	if selected {
		color, bg = p.AccentText, p.Accent
	}
	row := ui.ButtonBase(c).FillWidth().Height(28).
		Padding(0, 8, 0, 8+depthIndent(depthOf(path))).
		Radius(design.Radius).Background(bg).
		Label("存放到 " + store.PathName(path)).Justify(ui.Start).
		Cursor(ui.CursorPointer)
	if !selected && row.Hovered() {
		row.Background(p.SurfaceHover)
	}
	row.Children(func() {
		ui.Icon(c, icons.Folder).FontSize(12).TextColor(color)
		ui.Text(c, store.PathName(path)).FontSize(design.FontSizeBody).
			TextColor(color).SingleLine()
	})
	if row.Clicked() {
		q.folder = path
	}
}

// depthOf is how deep a tree path sits: 根目录 is 0.
func depthOf(treePath string) int {
	if treePath == "" {
		return 0 // the 根目录 sits at the top, not above it
	}
	return len(splitPath(treePath))
}

// splitPath splits a tree path into its names, none for the 根目录.
// splitPath splits a tree path into its names, none for the 根目录.
func splitPath(treePath string) []string {
	var names []string
	for _, name := range strings.Split(treePath, "/") {
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// folderButton is the button that goes ahead with the folder chosen.
func folderButton(c *ui.Context, p design.Palette, q *folderQuestion) ui.Element {
	btn := ui.ButtonBase(c).Height(32).Padding(0, 14).Radius(design.Radius).
		Label(q.action).Background(p.Accent).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(p.AccentHover)
	}
	btn.Children(func() {
		ui.Text(c, q.action).FontSize(design.FontSizeBody).
			TextColor(p.AccentText).SingleLine()
	})
	return btn
}

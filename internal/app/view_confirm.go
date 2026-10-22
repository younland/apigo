package app

import (
	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// confirm is a question asked before an action: what it says, the label of the
// button that runs it, and the action itself. danger paints that button in the
// color of what cannot be undone lightly.
type confirm struct {
	open   bool
	title  string
	body   string
	action string
	danger bool
	run    func(c *ui.Context)
}

// ask asks a question over the window: nothing it guards happens until the
// answer comes.
func (a *app) ask(q confirm) {
	q.open = true
	a.question = q
}

// askToCloseCollection asks before a collection's workspace goes, which writes
// its tabs back and closes their layout.
func (a *app) askToCloseCollection(id string) {
	name := a.nameOfCollection(id)
	if name == "" {
		return
	}
	a.ask(confirm{
		title:  "关闭集合",
		body:   "关闭「" + name + "」的窗口？标签里的改动会先写回接口文件，再打开时是空的标签页。",
		action: "关闭集合",
		run:    func(c *ui.Context) { a.closeCollection(id) },
	})
}

// askToDeleteCollection asks before a collection leaves the list, saying what
// stays: the folder of a collection is never removed.
func (a *app) askToDeleteCollection(id string) {
	name := a.nameOfCollection(id)
	if name == "" {
		return
	}
	a.ask(confirm{
		title:  "删除集合",
		body:   "「" + name + "」将从列表中移除，但它的文件夹、接口文件和环境仍保留在数据目录里，不会被删除。",
		action: "删除集合",
		danger: true,
		run: func(c *ui.Context) {
			if err := a.deleteCollection(id); err != nil {
				c.Toast("删除失败：" + err.Error())
			}
		},
	})
}

// nameOfCollection is the name of a collection, empty when it is gone.
func (a *app) nameOfCollection(id string) string {
	if i := a.collectionIndex(id); i >= 0 {
		return a.store.Collections[i].Name
	}
	return ""
}

// askToForgetMissingCollection asks before a collection whose directory is gone
// leaves the list: there is nothing left to open, and its folder is not there to
// keep either.
func (a *app) askToForgetMissingCollection(id string) {
	name := a.nameOfCollection(id)
	if name == "" {
		return
	}
	a.ask(confirm{
		title:  "集合不存在",
		body:   "「" + name + "」的文件夹已不在数据目录里，无法打开。是否把它从列表中移除？",
		action: "移除集合",
		danger: true,
		run: func(c *ui.Context) {
			if err := a.deleteCollection(id); err != nil {
				c.Toast("移除失败：" + err.Error())
				return
			}
			delete(a.missing, id)
		},
	})
}

// confirmModal draws the question, and runs its action on a yes.
func (a *app) confirmModal(c *ui.Context, p design.Palette) {
	q := &a.question
	if !q.open {
		return
	}
	ui.DialogBase(c, &q.open, func(backdrop, panel ui.Element) {
		topBackdrop(backdrop)
		panel.Width(360).Padding(20).Gap(12).Radius(8).
			Background(p.Background).Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
		ui.Text(c, q.title).Bold().FontSize(design.FontSizeDisplay).TextColor(p.Text)
		ui.Text(c, q.body).FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
		ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1)
			if ui.Button(c, "取消").Cursor(ui.CursorPointer).Clicked() {
				q.open = false
			}
			if confirmButton(c, p, q).Clicked() {
				q.open = false
				if run := q.run; run != nil {
					q.run = nil
					run(c)
				}
			}
		})
	})
}

// confirmButton is the button that answers yes: the accent of the app, or its
// danger color for what asks a warning.
func confirmButton(c *ui.Context, p design.Palette, q *confirm) ui.Element {
	bg, hover := p.Accent, p.AccentHover
	if q.danger {
		bg, hover = p.Danger, p.Danger
	}
	btn := ui.ButtonBase(c).Height(32).Padding(0, 14).Radius(design.Radius).
		Label("确认" + q.action).Background(bg).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(hover)
	}
	btn.Children(func() {
		ui.Text(c, q.action).FontSize(design.FontSizeBody).TextColor(p.AccentText).SingleLine()
	})
	return btn
}

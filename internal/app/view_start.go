package app

import (
	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// startHint says what a collection opened to: a pane of what to do, in the
// manner of the shortcuts Hoppscotch lists, rather than an empty request.
func (a *app) startHint(c *ui.Context, p design.Palette) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(10).Children(func() {
		ui.Text(c, "还没有打开接口").FontSize(design.FontSizeTitle).Bold().TextColor(p.Text)
		ui.Text(c, "在左侧目录里新建一个接口，或点一个已有的接口打开").
			FontSize(design.FontSizeBody).TextColor(p.TextMuted)
		ui.Column(c).Gap(8).Padding(design.RowGap, 0).Children(func() {
			a.hintRow(c, p, "新建接口", "左侧目录行尾的 +")
			a.hintRow(c, p, "打开已有接口", "点击左侧的接口")
			a.hintRow(c, p, "发送请求", "Ctrl", "↵")
			a.hintRow(c, p, "保存接口", "Ctrl", "S")
		})
	})
}

// hintRow is one line of the start pane: what it does, on the right, and how,
// as the keys to press.
func (a *app) hintRow(c *ui.Context, p design.Palette, label string, keys ...string) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(design.FontSizeBody).TextColor(p.TextSecondary).SingleLine()
		for _, key := range keys {
			keycap(c, p, key)
		}
	})
}

// keycap draws one key to press, as a small boxed label.
func keycap(c *ui.Context, p design.Palette, key string) {
	ui.Row(c).Height(22).Padding(0, 7).Radius(design.Radius).
		Background(p.Surface).Border(design.BorderWidth, p.Border).
		Center().Children(func() {
		ui.Text(c, key).FontSize(design.FontSizeTiny).TextColor(p.TextMuted).SingleLine()
	})
}

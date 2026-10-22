package app

import (
	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// startHint says what a collection opened to: a pane of what to do, in the
// manner of the shortcuts Hoppscotch lists, rather than an empty request. The
// title is centered, and the actions under it start on one left edge.
func (a *app) startHint(c *ui.Context, p design.Palette) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(10).Children(func() {
		ui.Text(c, "还没有打开接口").FontSize(design.FontSizeTitle).Bold().TextColor(p.Text)
		ui.Text(c, "在左侧目录里新建一个接口，或点一个已有的接口打开").
			FontSize(design.FontSizeBody).TextColor(p.TextMuted)
		a.hintList(c, p)
	})
}

// hint is one line of the start pane: what it does, and how, as the keys to
// press.
type hint struct {
	label string
	keys  []string
}

// hintList draws the actions of the start pane. Their labels share one column,
// so the keys beside them line up, and the rows start on one left edge under
// the title rather than drifting with the width of each row.
func (a *app) hintList(c *ui.Context, p design.Palette) {
	hints := []hint{
		{"新建接口", []string{"左侧目录行尾的 +"}},
		{"打开已有接口", []string{"点击左侧的接口"}},
		{"发送请求", []string{"Ctrl", "↵"}},
		{"保存接口", []string{"Ctrl", "S"}},
	}
	width := hintLabelWidth(c, hints)
	ui.Column(c).Gap(8).Padding(design.RowGap, 0).Children(func() {
		for _, h := range hints {
			a.hintRow(c, p, width, h)
		}
	})
}

// hintLabelWidth is the room the labels of the start pane take: the widest of
// them, with the hair the toolkit rounds away.
func hintLabelWidth(c *ui.Context, hints []hint) float32 {
	var width float32
	for _, h := range hints {
		width = max(width, measureText(c, h.label, design.FontSizeBody, 400))
	}
	return width + tabWiggle
}

// hintRow is one line of the start pane: what it does on the left, in the
// column its labels share, and how, as the keys to press. The label's room is a
// MinWidth rather than a Width: a fixed width no longer counts in the row's
// own measure, and the keys beside it would be squeezed.
func (a *app) hintRow(c *ui.Context, p design.Palette, labelWidth float32, h hint) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, h.label).MinWidth(labelWidth).FontSize(design.FontSizeBody).
			TextColor(p.TextSecondary).SingleLine()
		for _, key := range h.keys {
			keycap(c, p, key)
		}
	})
}

// keycapPadding is the room a keycap leaves beside its key.
const keycapPadding float32 = 7

// keycap draws one key to press, as a small boxed label. Its width is a
// MinWidth of what the key takes: left to its content the toolkit rounds the
// last fraction away and ellipsizes a key that just fits.
func keycap(c *ui.Context, p design.Palette, key string) {
	width := 2*keycapPadding + measureText(c, key, design.FontSizeTiny, 400) + tabWiggle
	ui.Row(c).Height(22).Padding(0, keycapPadding).MinWidth(width).Radius(design.Radius).
		Background(p.Surface).Border(design.BorderWidth, p.Border).
		Center().Children(func() {
		ui.Text(c, key).FontSize(design.FontSizeTiny).TextColor(p.TextMuted).SingleLine()
	})
}

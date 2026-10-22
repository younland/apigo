package app

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// responseViews are the panes of the response: what the body is in its own
// shape, the body as it came, the response headers and the request headers.
const (
	respPreview = iota
	respRaw
	respHeaders
	respRequestHeaders
)

// responseTabNames are the panes of the response: the preview of the body, the
// body as it came, the response headers and the request headers.
var responseTabNames = []string{"预览", "原始内容", "响应头", "请求头"}

// responseTabStrip draws the panes of the response, with the number of headers
// beside the two that list them.
func (a *app) responseTabStrip(c *ui.Context, p design.Palette, t *tab) {
	badges := []string{"", "", countBadge(len(t.resp.Headers)), countBadge(len(t.sentHeaders))}
	ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
		for i, name := range responseTabNames {
			a.responseTab(c, p, t, i, name, badges[i])
		}
	})
}

// countBadge is the count a header tab shows, empty for none.
func countBadge(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// responseTabLabel names a response tab for assistive technology: the request
// panes carry two of the same names, so the label says which side it is on.
func responseTabLabel(name string) string { return "响应页签 " + name }

// responseTab draws one tab, with the accent line under the selected one.
func (a *app) responseTab(c *ui.Context, p design.Palette, t *tab, i int, name, badge string) {
	const pad, gap float32 = 8, 4
	color, line := p.TextMuted, p.Background
	if t.respTab == i {
		color, line = p.Text, p.Accent
	}
	// The tab keeps the width its name needs: left to itself the toolkit measures
	// the text short and the row squeezes it into 「原始…」.
	width := 2*pad + textWidth(c, name) + tabWiggle
	if badge != "" {
		width += gap + textWidth(c, badge)
	}
	ui.Column(c).Shrink(0).Gap(4).Children(func() {
		if ui.ButtonBase(c).Label(responseTabLabel(name)).Padding(6, pad).Gap(gap).Shrink(0).
			MinWidth(width).Cursor(ui.CursorPointer).Children(func() {
			ui.Text(c, name).FontSize(design.FontSizeBody).TextColor(color).SingleLine().
				MinWidth(textWidth(c, name) + tabWiggle)
			if badge != "" {
				ui.Text(c, badge).FontSize(design.FontSizeTiny).TextColor(p.TextMuted).SingleLine()
			}
		}).Clicked() {
			t.respTab = i
		}
		ui.Row(c).Height(2).FillWidth().Background(line)
	})
}

// responseTabPanel is the pane of the selected response tab.
func (a *app) responseTabPanel(c *ui.Context, p design.Palette, t *tab) {
	switch t.respTab {
	case respRaw:
		editor.View(c, &t.rawEdit).Grow(1).FillWidth()
	case respHeaders:
		a.headerList(c, p, "响应头列表", t.resp.Headers)
	case respRequestHeaders:
		a.headerList(c, p, "请求头列表", t.sentHeaders)
	default:
		a.responseBodyView(c, p, t)
	}
}

// headerList draws a message's headers as the two columns a reader scans: the
// name, then the value, one row each.
func (a *app) headerList(c *ui.Context, p design.Palette, caption string, headers http.Header) {
	names := sortedHeaderNames(headers)
	ui.Column(c).Grow(1).FillWidth().Children(func() {
		ui.Row(c).FillWidth().Padding(design.RowGap, design.PanelPadding).Children(func() {
			ui.Text(c, caption).FontSize(design.FontSizeTiny).TextColor(p.TextMuted).SingleLine()
		})
		if len(names) == 0 {
			ui.Row(c).FillWidth().Padding(design.RowGap, design.PanelPadding).Children(func() {
				ui.Text(c, "没有请求头").FontSize(design.FontSizeBody).
					TextColor(p.TextMuted).SingleLine()
			})
			return
		}
		noScrollbars(ui.Scroll(c)).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				for _, name := range names {
					a.headerRow(c, p, name, headers[name])
				}
			})
		})
	})
}

// headerRow is one header: its name on the left, its value on the right,
// under a divider.
func (a *app) headerRow(c *ui.Context, p design.Palette, name string, values []string) {
	ui.Row(c).FillWidth().Height(32).BorderWidth(0, 0, design.BorderWidth, 0).
		BorderColor(p.Divider).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(2).Padding(0, design.PanelPadding).Children(func() {
			// Header names read in lower case, as the wire writes them.
			ui.Text(c, strings.ToLower(name)).FontSize(design.FontSizeBody).
				TextColor(p.TextSecondary).SingleLine().Selectable()
		})
		ui.Row(c).Grow(3).Padding(0, design.PanelPadding).Children(func() {
			ui.Text(c, strings.Join(values, ", ")).FontSize(design.FontSizeBody).
				TextColor(p.Text).SingleLine().Selectable()
		})
	})
}

// sortedHeaderNames lists a message's header names as a reader expects them.
func sortedHeaderNames(headers http.Header) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

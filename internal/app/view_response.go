package app

import (
	"fmt"
	"time"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// response draws the meta bar — the status of the response and the tabs of its
// panes — over the pane the reader picked, or the state that tells what
// happened instead.
func (a *app) response(c *ui.Context, p design.Palette) {
	t := a.current()
	ui.Column(c).FillWidth().FillHeight().Background(p.Background).Children(func() {
		a.responseMeta(c, p, t)
		a.divider(c, p)
		switch {
		case t.resp == nil:
			a.responseEmpty(c, p)
		case t.resp.Err != nil:
			a.responseError(c, p, t)
		default:
			a.responseTabPanel(c, p, t)
		}
	})
}

// responseMeta draws the status, the time it took and the size of the body.
func (a *app) responseMeta(c *ui.Context, p design.Palette, t *tab) {
	ui.Row(c).FillWidth().Height(44).Padding(0, design.PanelPadding).
		Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "响应").Bold().FontSize(design.FontSizeBody).TextColor(p.Text).SingleLine()
		// The panes of the response sit on the same line as its name: what the
		// body holds, how it came, and the headers of both ways.
		if t.resp != nil && t.resp.Err == nil {
			a.responseTabStrip(c, p, t)
		}
		ui.Row(c).Grow(1)
		switch {
		case t.loading:
			ui.Spinner(c).Size(14, 14)
			ui.Text(c, "请求中…").FontSize(design.FontSizeTiny).TextColor(p.TextMuted).SingleLine()
		case t.resp == nil:
			ui.Text(c, "Ctrl+Enter 发送请求").FontSize(design.FontSizeTiny).
				TextColor(p.TextMuted).SingleLine()
		case t.resp.Err != nil:
			ui.Icon(c, icons.Alert).FontSize(14).TextColor(p.Danger)
			ui.Text(c, t.resp.Err.Error()).FontSize(design.FontSizeBody).
				TextColor(p.Danger).SingleLine()
		default:
			ui.Textf(c, "%d %s", t.resp.Status, t.resp.StatusText).Bold().
				FontSize(design.FontSizeBody).TextColor(p.Status(t.resp.Status)).SingleLine()
			ui.Text(c, formatDuration(t.resp.Duration)).FontSize(design.FontSizeTiny).
				TextColor(p.TextMuted).SingleLine()
			ui.Text(c, formatSize(t.resp.Size)).FontSize(design.FontSizeTiny).
				TextColor(p.TextMuted).SingleLine()
			a.downloadButton(c, p, t)
		}
	})
}

// downloadButton saves the body of the response where the reader says, as the
// download icon at the end of the meta bar.
func (a *app) downloadButton(c *ui.Context, p design.Palette, t *tab) {
	btn := ui.ButtonBase(c).Width(20).Height(20).Label("下载响应").
		Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Download).FontSize(13).TextColor(p.TextMuted)
	})
	if btn.Clicked() {
		a.saveResponse(c, t)
	}
}

// responseEmpty tells the reader how the pane fills.
func (a *app) responseEmpty(c *ui.Context, p design.Palette) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(6).Children(func() {
		ui.Icon(c, icons.Send).FontSize(28).TextColor(p.TextMuted)
		ui.Text(c, "发送请求后，响应会显示在这里").FontSize(design.FontSizeBody).
			TextColor(p.TextSecondary)
		ui.Text(c, "支持 JSON、XML、HTML 与原始文本").FontSize(design.FontSizeTiny).
			TextColor(p.TextMuted)
	})
}

// responseError says what stopped the request, with a way to try again.
func (a *app) responseError(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(6).Children(func() {
		ui.Icon(c, icons.Alert).FontSize(28).TextColor(p.Danger)
		ui.Text(c, t.resp.Err.Error()).FontSize(design.FontSizeBody).
			TextColor(p.TextSecondary)
		if ui.Button(c, "重试").Cursor(ui.CursorPointer).Clicked() {
			a.send(c)
		}
	})
}

// formatDuration writes a request's time as a person reads it.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%d ms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2f s", d.Seconds())
}

// formatSize writes a body's size in B, KB or MB.
func formatSize(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
}

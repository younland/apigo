package main

import (
	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

var requestTabs = []string{"Params", "Headers", "Body", "Authorization", "Pre-request", "Scripts", "Tests"}

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// request draws the request line and the pane of the selected tab.
func (a *app) request(c *ui.Context, p design.Palette) {
	t := a.current()
	c.OnShortcut(ui.Cmd, ui.KeyEnter, func() { a.send(c) })
	ui.Column(c).FillWidth().FillHeight().
		Padding(design.PanelPadding).Gap(design.RowGap).
		Background(p.Background).Children(func() {
		a.requestLine(c, p, t)
		a.requestTabStrip(c, p, t)
		a.requestPanel(c, p, t)
	})
}

// requestLine draws the method menu, the URL field and the send button.
func (a *app) requestLine(c *ui.Context, p design.Palette, t *tab) {
	ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
		a.methodBox(c, p, t)
		ui.TextInput(c, &t.url).Grow(1).Height(design.RequestRowHeight).
			Placeholder("输入请求 URL").Font("monospace")
		if t.loading {
			ui.Button(c, "发送中…").Height(design.RequestRowHeight)
			return
		}
		if ui.PrimaryButton(c, "发送").Height(design.RequestRowHeight).Clicked() {
			a.send(c)
		}
	})
}

// methodBox is the method button and the menu of methods it opens.
func (a *app) methodBox(c *ui.Context, p design.Palette, t *tab) {
	btn := ui.ButtonBase(c).Width(design.MethodWidth).Height(design.RequestRowHeight).
		Label("请求方法").Justify(ui.Start).Padding(0, 12).
		Background(p.Surface).Border(design.BorderWidth, p.Divider).Children(func() {
		ui.Text(c, t.method).Bold().FontSize(design.FontSizeBody).
			TextColor(p.Method(t.method)).SingleLine()
		ui.Row(c).Grow(1)
		ui.Icon(c, icons.Chevron).FontSize(12).TextColor(p.TextMuted)
	})
	if btn.Clicked() {
		a.methodOpen = !a.methodOpen
	}
	ui.PopoverBase(c, btn, &a.methodOpen, func(panel ui.Element) {
		panel.Width(120).Padding(4).Radius(design.Radius).
			Background(p.Surface).Border(design.BorderWidth, p.Divider)
		for _, method := range methods {
			if ui.ButtonBase(c).Height(28).Label(method).Justify(ui.Start).Padding(0, 8).
				Children(func() {
					ui.Text(c, method).FontSize(design.FontSizeBody).
						TextColor(p.Method(method)).SingleLine()
				}).Clicked() {
				t.method, a.methodOpen = method, false
			}
		}
	})
}

// requestTabStrip draws the tabs over the request pane.
func (a *app) requestTabStrip(c *ui.Context, p design.Palette, t *tab) {
	ui.Row(c).FillWidth().Gap(2).Children(func() {
		for i, name := range requestTabs {
			a.requestTab(c, p, t, i, name, t.reqTab == i)
		}
	})
}

// requestTab draws one tab with the accent line under the selected one.
func (a *app) requestTab(c *ui.Context, p design.Palette, t *tab, i int, name string, selected bool) {
	color, line := p.TextMuted, p.Background
	if selected {
		color, line = p.Text, p.Accent
	}
	ui.Column(c).Gap(4).Children(func() {
		if ui.ButtonBase(c).Label(name).Padding(6, 8).Children(func() {
			ui.Text(c, name).FontSize(design.FontSizeBody).TextColor(color).SingleLine()
		}).Clicked() {
			t.reqTab = i
		}
		ui.Row(c).Height(2).FillWidth().Background(line)
	})
}

// requestPanel is the body of the selected request tab.
func (a *app) requestPanel(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).Grow(1).FillWidth().Children(func() {
		switch t.reqTab {
		case 0:
			t.params.View(c, p)
		case 1:
			t.headers.View(c, p)
		case 2:
			a.bodyPanel(c, p, t)
		default:
			ui.Text(c, "该面板尚未实现").FontSize(design.FontSizeBody).TextColor(p.TextMuted)
		}
	})
}

// bodyPanel edits the request body in the format the type selects. JSON
// bodies are edited in the code editor with JSON highlighting, checked
// before they are sent, and re-indented on demand.
func (a *app) bodyPanel(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).FillWidth().FillHeight().Gap(design.RowGap).Children(func() {
		ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
			if ui.Select(c, &t.bodyType, bodyTypes).Width(120).Changed() {
				t.setBodyLanguage()
			}
			if t.bodyType == "json" && ui.Button(c, "格式化").Clicked() {
				a.formatBody(c)
			}
			ui.Text(c, "请求体").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		})
		if t.bodyType == "none" {
			ui.Text(c, "该请求不发送请求体").FontSize(design.FontSizeBody).TextColor(p.TextMuted)
			return
		}
		editor.View(c, &t.bodyEdit).Grow(1).FillWidth()
	})
}

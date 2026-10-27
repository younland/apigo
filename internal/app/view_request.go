package app

import (
	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// requestTabs are the panes of a request, in the language of the interface.
// Authorization, Pre-request, Scripts and Tests are not shown yet: they have
// nothing behind them.
var requestTabs = []string{"参数", "请求头", "请求体"}

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
		a.addressField(c, p, t, "输入请求 URL")
		if t.loading {
			// The request is waiting: the same button stops it.
			a.iconButton(c, p, iconButtonSpec{label: "取消", name: "取消请求", icon: icons.Close,
				run: func() { a.cancelSend(t) }})
			a.saveButton(c, p)
			return
		}
		a.iconButton(c, p, iconButtonSpec{label: "发送", name: "发送请求", icon: icons.Send,
			run: func() { a.send(c) }})
		a.saveButton(c, p)
	})
}

// addressPrefixGap is the room between the prefix URL and the address inside
// the field: enough that the two read as one address, not as one run-on word.
const addressPrefixGap float32 = 4

// addressField is where a request goes: the prefix URL of the environment in
// use stands inside the field, before the address, while the address is a path
// under it. The field reads as one input: the prefix is part of it, not a
// control of its own.
func (a *app) addressField(c *ui.Context, p design.Palette, t *tab, placeholder string) {
	base, under := a.baseURLPrefix(t)
	field := ui.Row(c).Grow(1).Height(design.RequestRowHeight).AlignItems(ui.Center).
		Padding(0, 10).Radius(design.Radius).Background(p.Surface).
		Border(design.BorderWidth, p.Border)
	var input ui.Element
	field.Children(func() {
		if under {
			ui.Text(c, base).FontSize(design.FontSizeBody).Font("monospace").
				TextColor(p.TextMuted).SingleLine()
			ui.Row(c).Width(addressPrefixGap)
		}
		input = ui.TextInputBase(c, &t.url).Grow(1).Font("monospace").Placeholder(placeholder)
	})
	// The whole field takes the focus ring, as its inner input cannot draw one
	// that covers the prefix beside it.
	if input.Focused() {
		field.Border(design.BorderWidth, p.Accent)
	}
}

// saveButton keeps the request in its interface file, asking where it goes when
// it has none: the same as Ctrl+S.
func (a *app) saveButton(c *ui.Context, p design.Palette) {
	a.iconButton(c, p, iconButtonSpec{label: "保存", name: "保存接口", icon: icons.Save, plain: true,
		run: func() { a.saveCurrent(c) }})
}

// methodBox is the method button and the menu of methods it opens.
func (a *app) methodBox(c *ui.Context, p design.Palette, t *tab) {
	btn := ui.ButtonBase(c).Width(design.MethodWidth).Height(design.RequestRowHeight).
		Label("请求方法").Justify(ui.Start).Padding(0, 12).
		Background(p.Surface).Border(design.BorderWidth, p.Border).Cursor(ui.CursorPointer).
		Children(func() {
			ui.Text(c, t.method).Bold().FontSize(design.FontSizeBody).
				TextColor(p.Method(t.method)).SingleLine()
			ui.Row(c).Grow(1)
			ui.Icon(c, icons.Chevron).FontSize(12).TextColor(p.TextMuted)
		})
	if btn.Clicked() {
		a.methodOpen = !a.methodOpen
	}
	ui.PopoverBase(c, btn, &a.methodOpen, func(panel ui.Element) {
		menuPanel(panel, p, 120)
		for _, method := range methods {
			row := menuRow(ui.ButtonBase(c).Height(28).Label(method).Justify(ui.Start).Padding(0, 8).
				Cursor(ui.CursorPointer).Children(func() {
				ui.Text(c, method).FontSize(design.FontSizeBody).
					TextColor(p.Method(method)).SingleLine()
			}), p)
			if row.Clicked() {
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
	paneTab(c, p, name, selected, func() { t.reqTab = i })
}

// paneTab draws a pane's tab with the accent line under the selected one: the
// panes of a request and of a socket share the look.
func paneTab(c *ui.Context, p design.Palette, name string, selected bool, choose func()) {
	color, line := p.TextMuted, p.Background
	if selected {
		color, line = p.Text, p.Accent
	}
	ui.Column(c).Shrink(0).Gap(4).Children(func() {
		if ui.ButtonBase(c).Label(name).Padding(6, 8).Shrink(0).Cursor(ui.CursorPointer).
			Children(func() {
				ui.Text(c, name).FontSize(design.FontSizeBody).TextColor(color).SingleLine()
			}).Clicked() {
			choose()
		}
		ui.Row(c).Height(2).FillWidth().Background(line)
	})
}

// requestPanel is the body of the selected request tab.
func (a *app) requestPanel(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).Grow(1).FillWidth().Children(func() {
		switch t.reqTab {
		case 0:
			a.paramsPanel(c, p, t)
		case 1:
			t.headers.View(c, p)
		case 2:
			a.bodyPanel(c, p, t)
		default:
			ui.Text(c, "该面板尚未实现").FontSize(design.FontSizeBody).TextColor(p.TextMuted)
		}
	})
}

// paramsPanel draws what a request takes: its query parameters, which follow the
// address, and the values its path takes, which the address names. A request
// whose address takes no path parameters has no path table at all.
func (a *app) paramsPanel(c *ui.Context, p design.Palette, t *tab) {
	t.syncPathParams()
	// The two tables scroll together: a path of many parameters pushes the second
	// one down rather than out of the pane.
	noScrollbars(ui.Scroll(c)).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Gap(design.PanelPadding).Children(func() {
			t.params.View(c, p, "Query 参数")
			if len(t.pathParams.Rows) > 0 {
				t.pathParams.View(c, p, "Path 参数")
			}
		})
	})
}

// bodyPanel edits the request body in the format it is sent as: the format
// row picks it, and the pane below edits what that format holds. The
// Content-Type it sends with sits at the right of the row, with the wand that
// re-indents a JSON body beside it.
func (a *app) bodyPanel(c *ui.Context, p design.Palette, t *tab) {
	format := bodyFormatOf(t.bodyType)
	ui.Column(c).FillWidth().FillHeight().Gap(design.RowGap).Children(func() {
		ui.Row(c).FillWidth().Gap(2).AlignItems(ui.Center).Children(func() {
			// The formats scroll before the Content-Type they send with: at a
			// narrow window nine of them do not fit, and squeezing one to a
			// sliver says nothing of its format.
			noScrollbars(ui.ScrollHorizontal(c)).Grow(1).Height(28).Children(func() {
				a.bodyFormatStrip(c, p, t)
			})
			if format.MIME != "" {
				ui.Text(c, format.MIME).FontSize(design.FontSizeTiny).
					TextColor(p.TextMuted).Selectable()
				ui.Text(c, "*").FontSize(design.FontSizeTiny).TextColor(p.Accent)
			}
			a.prettifyButton(c, p, t)
		})
		switch format.Edits {
		case editsNone:
			a.bodyHint(c, p, "该请求不发送请求体")
		case editsRows:
			a.formRows(c, p, t, format)
		case editsText:
			editor.View(c, &t.bodyEdit).Grow(1).FillWidth()
		default:
			a.bodyHint(c, p, format.Label+" 请求体需要选择文件，暂未实现")
		}
	})
}

// formRows draws the rows of an encoded body. form-data rows hold a file each,
// which the app copies into the collection and deletes when the row lets it
// go; the rows of x-www-form-urlencoded hold text.
func (a *app) formRows(c *ui.Context, p design.Palette, t *tab, format bodyFormat) {
	if format.Key == "form-data" {
		t.form.ChooseFile = func() (string, string, error) { return a.pickFormFile(c, t) }
		t.form.FileDropped = func(rel string) { a.dropFormFile(t, rel) }
		t.form.Thumbnail = a.fileThumbnail(a.staticScope(t))
	} else {
		t.form.ChooseFile, t.form.FileDropped, t.form.Thumbnail = nil, nil, nil
	}
	t.form.View(c, p)
}

// bodyHint says what a body format holds, in the room its editor would take.
func (a *app) bodyHint(c *ui.Context, p design.Palette, text string) {
	ui.Column(c).Grow(1).FillWidth().Padding(design.RowGap, 0).Children(func() {
		ui.Text(c, text).FontSize(design.FontSizeBody).TextColor(p.TextMuted)
	})
}

// bodyPillPadding is the room a format pill leaves beside its label.
const bodyPillPadding float32 = 10

// bodyPillWidth is how wide a format pill draws: its label and the room the
// paddings take, with a hair more. The toolkit rounds the last fraction away and
// ellipsizes a label that just fits, which read 「X…」 for XML.
func bodyPillWidth(c *ui.Context, label string) float32 {
	return 2*bodyPillPadding + measureText(c, label, design.FontSizeBody, 400) + tabWiggle
}

// bodyFormatStrip is the row of formats the Body tab offers: the one in use
// filled with the accent, as Hoppscotch draws it.
func (a *app) bodyFormatStrip(c *ui.Context, p design.Palette, t *tab) {
	ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
		for _, format := range bodyFormats {
			selected := t.bodyType == format.Key
			color, bg := p.TextSecondary, ui.Color{}
			if selected {
				color, bg = p.AccentText, p.Accent
			}
			btn := ui.ButtonBase(c).Height(26).Padding(0, bodyPillPadding).Radius(13).
				MinWidth(bodyPillWidth(c, format.Label)).
				Background(bg).Label("请求体格式 " + format.Label).Cursor(ui.CursorPointer)
			if !selected && btn.Hovered() {
				btn.Background(p.SurfaceHover)
			}
			btn.Children(func() {
				ui.Text(c, format.Label).FontSize(design.FontSizeBody).
					TextColor(color).SingleLine()
			})
			if btn.Clicked() {
				t.bodyType = format.Key
				t.setBodyLanguage()
			}
		}
	})
}

// prettifyButton indents a JSON body, as Hoppscotch's wand does.
func (a *app) prettifyButton(c *ui.Context, p design.Palette, t *tab) {
	switch t.bodyType {
	case "json", "graphql":
	default:
		return
	}
	if ui.ButtonBase(c).Width(28).Height(28).Label("美化请求体").Cursor(ui.CursorPointer).
		Children(func() {
			ui.Icon(c, icons.Wand).FontSize(15).TextColor(p.Accent)
		}).Clicked() {
		a.formatBody(c)
	}
}

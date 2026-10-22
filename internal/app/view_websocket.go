package app

import (
	"strings"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// socketTabs are the panes of a socket request: what it sends, and how it
// opens.
var socketTabs = []string{"通讯", "协议"}

// socketPane is a WebSocket request: its address and connection, the message it
// sends, and how it opens.
func (a *app) socketPane(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).FillWidth().FillHeight().
		Padding(design.PanelPadding).Gap(design.RowGap).
		Background(p.Background).Children(func() {
		a.socketLine(c, p, t)
		ui.Row(c).FillWidth().Gap(2).Children(func() {
			for i, name := range socketTabs {
				paneTab(c, p, name, t.wsTab == i, func() { t.wsTab = i })
			}
		})
		if t.wsTab == 1 {
			a.socketProtocolPane(c, p, t)
			return
		}
		a.socketMessagePane(c, p, t)
	})
}

// socketLine draws the address, the connect button and the way to save, with an
// icon beside each label.
func (a *app) socketLine(c *ui.Context, p design.Palette, t *tab) {
	ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
		ui.TextInput(c, &t.url).Grow(1).Height(design.RequestRowHeight).
			Placeholder("ws://localhost:8080/socket").Font("monospace")
		switch {
		case t.connecting:
			ui.Text(c, "连接中…").FontSize(design.FontSizeBody).TextColor(p.TextMuted)
		case t.conn != nil:
			a.iconButton(c, p, iconButtonSpec{label: "断开", name: "断开连接", icon: icons.Plug,
				run: func() { a.disconnectSocket(t) }})
		default:
			a.iconButton(c, p, iconButtonSpec{label: "连接", name: "连接", icon: icons.Plug,
				run: func() { a.connectSocket(c, t) }})
		}
		a.iconButton(c, p, iconButtonSpec{label: "保存", name: "保存接口", icon: icons.Save, plain: true,
			run: func() { a.saveCurrent(c) }})
	})
}

// socketMessagePane draws what the socket sends: the format of a message, the
// message itself, and the button that sends it.
func (a *app) socketMessagePane(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).FillWidth().FillHeight().Gap(design.RowGap).Children(func() {
		a.socketToolbar(c, p, t)
		editor.View(c, &t.bodyEdit).Grow(1).FillWidth()
	})
}

// socketToolbar is the row over the message: its format, whether it clears
// after sending, and the send button.
func (a *app) socketToolbar(c *ui.Context, p design.Palette, t *tab) {
	ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "信息").FontSize(design.FontSizeBody).TextColor(p.TextSecondary).SingleLine()
		a.messageFormatMenu(c, p, t)
		ui.Row(c).Grow(1)
		ui.Checkbox(c, &t.clearOnSend, "").Label("发送后清空输入")
		ui.Text(c, "清空输入").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		if t.bodyType == "binary" {
			ui.Text(c, "二进制消息暂不支持发送").FontSize(design.FontSizeTiny).
				TextColor(p.TextMuted)
		}
		// A message goes out only over an open socket, and as text.
		a.iconButton(c, p, iconButtonSpec{
			label: "发送", name: "发送消息", icon: icons.Send,
			disabled: t.conn == nil || t.bodyType == "binary",
			plain:    true, small: true,
			run: func() { a.sendSocketMessage(c, t) },
		})
	})
}

// messageFormatMenu picks the format the message reads in: Text, JSON, XML,
// HTML or Binary. It draws as the environment picker does, and the format in
// force is the ticked line of its list.
func (a *app) messageFormatMenu(c *ui.Context, p design.Palette, t *tab) {
	const width float32 = 140
	label, _ := socketFormat(t.bodyType)
	open := a.menuFlag("socket-format")
	btn := selectTrigger(c, p, selectSpec{name: "消息格式 " + label, label: label, width: width})
	if btn.Clicked() {
		*open = !*open
	}
	ui.PopoverBase(c, btn, open, func(panel ui.Element) {
		menuPanel(panel, p, width)
		for _, format := range socketFormats {
			selectChoiceRow(c, p, selectChoice{
				name: "消息格式 " + format.label, label: format.label,
				chosen: format.key == t.bodyType,
				run: func() {
					t.bodyType, *open = format.key, false
					t.setBodyLanguage()
				},
			})
		}
	})
}

// socketLanguage is the syntax a message reads in, nil for plain text.
func socketLanguage(format string) *editor.Language {
	_, language := socketFormat(format)
	if language == "" {
		return nil
	}
	return editor.LanguageNamed(language)
}

// socketProtocolPane draws how the socket opens: the subprotocol it asks for
// and the headers of the handshake.
func (a *app) socketProtocolPane(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).FillWidth().FillHeight().Gap(design.RowGap).Children(func() {
		ui.Row(c).Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "子协议").FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
			ui.TextInput(c, &t.protocol).Label("子协议输入").Width(200).
				Height(design.RequestRowHeight).Placeholder("可选，如 chat")
		})
		ui.Text(c, "握手请求头").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		ui.Column(c).Grow(1).FillWidth().Children(func() {
			t.headers.View(c, p)
		})
	})
}

// socketLogPane is the log under a socket: what the connection did, newest
// last.
func (a *app) socketLogPane(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).FillWidth().FillHeight().Background(p.Background).Children(func() {
		ui.Row(c).FillWidth().Height(44).Padding(0, design.PanelPadding).
			Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "日志").Bold().FontSize(design.FontSizeBody).TextColor(p.Text).SingleLine()
			a.logCount(c, p, t)
			ui.Row(c).Grow(1)
			if len(t.socketLog) > 0 {
				ui.TextInput(c, &t.logFilter).Width(180).Height(28).Label("过滤日志").
					Placeholder("过滤日志")
				btn := ui.ButtonBase(c).Height(24).Label("清空日志").Padding(0, 8).
					Cursor(ui.CursorPointer).Children(func() {
					ui.Text(c, "清空").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
				})
				if btn.Clicked() {
					t.socketLog = nil
				}
			}
		})
		a.divider(c, p)
		a.socketLogList(c, p, t)
	})
}

// logCount says how many lines the log shows: the total, or what the filter
// left of it.
func (a *app) logCount(c *ui.Context, p design.Palette, t *tab) {
	shown, total := len(t.logLines()), len(t.socketLog)
	if shown == total {
		ui.Textf(c, "%d 条", total).FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		return
	}
	ui.Textf(c, "%d / %d 条", shown, total).FontSize(design.FontSizeTiny).
		TextColor(p.TextMuted)
}

// logLines are the lines the filter keeps, in the order the log holds them.
func (t *tab) logLines() []int {
	query := strings.ToLower(strings.TrimSpace(t.logFilter))
	if query == "" {
		lines := make([]int, len(t.socketLog))
		for i := range lines {
			lines[i] = i
		}
		return lines
	}
	var lines []int
	for i, line := range t.socketLog {
		if strings.Contains(strings.ToLower(line.text), query) {
			lines = append(lines, i)
		}
	}
	return lines
}

// socketLogList lists the log lines the filter keeps, keeping the newest in
// view. A message line opens into the code view of what it carried.
func (a *app) socketLogList(c *ui.Context, p design.Palette, t *tab) {
	if len(t.socketLog) == 0 {
		a.logEmpty(c, p, "连接后，消息与连接事件会显示在这里")
		return
	}
	lines := t.logLines()
	if len(lines) == 0 {
		a.logEmpty(c, p, "没有匹配「"+strings.TrimSpace(t.logFilter)+"」的日志")
		return
	}
	noScrollbars(ui.Scroll(c)).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			for _, i := range lines {
				a.socketLogRow(c, p, t, i)
			}
		})
	})
}

// logEmpty says why the log shows no line, in the room its lines would take.
func (a *app) logEmpty(c *ui.Context, p design.Palette, text string) {
	ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
		ui.Text(c, text).FontSize(design.FontSizeBody).TextColor(p.TextMuted)
	})
}

// logTimeWidth and logSizeWidth keep the size and the time of a log line in
// their own columns at the end of the row, so the messages all start and end at
// the same place. The time comes last, where the log's eye runs to.
const (
	logTimeWidth float32 = 60
	logSizeWidth float32 = 52
)

// socketLogRow is one line of the log: what happened, when, and — for a message
// the reader opened — the content it carried. The message takes the room the
// row has, with the size and the time in fixed columns after it.
func (a *app) socketLogRow(c *ui.Context, p design.Palette, t *tab, i int) {
	line := t.socketLog[i]
	mark, color := "•", p.TextMuted
	switch line.kind {
	case socketOpen:
		color = p.Accent
	case socketFail:
		mark, color = "!", p.Danger
	case socketSent:
		mark = "→"
	case socketGot:
		mark = "←"
	}
	row := ui.ButtonBase(c).FillWidth().Justify(ui.Start).Gap(design.RowGap).
		Padding(4, design.PanelPadding).Cursor(ui.CursorPointer).Label("日志 " + line.summary())
	row.Children(func() {
		if line.message() {
			caret := ui.Icon(c, icons.Chevron).FontSize(10).TextColor(p.TextMuted)
			if line.view == nil {
				caret.Rotate(-90)
			}
		}
		ui.Text(c, mark).FontSize(design.FontSizeTiny).TextColor(color).SingleLine()
		ui.Text(c, line.summary()).FontSize(design.FontSizeBody).TextColor(p.TextSecondary).
			SingleLine().Grow(1)
		size := ""
		if line.message() {
			size = formatSize(line.size)
		}
		ui.Text(c, size).FontSize(design.FontSizeTiny).TextColor(p.TextMuted).
			SingleLine().Width(logSizeWidth).TextAlign(ui.End)
		ui.Text(c, line.at.Format("15:04:05")).FontSize(design.FontSizeTiny).
			TextColor(p.TextMuted).SingleLine().Width(logTimeWidth).TextAlign(ui.End)
	})
	// The pointer over a line says it can be opened, over a message it does.
	if row.Hovered() {
		row.Background(p.SurfaceHover)
	}
	if row.Clicked() {
		a.toggleLogLine(t, i)
	}
	if line.view == nil {
		return
	}
	ui.Column(c).FillWidth().Padding(0, design.PanelPadding*2).Children(func() {
		editor.View(c, line.view).FillWidth()
	})
	a.divider(c, p)
}

// iconButtonSpec is a button of a request line: the text it shows, the name it
// is found by, its icon, whether it takes clicks, how it draws, and what it
// does.
type iconButtonSpec struct {
	label    string
	name     string
	icon     *ui.SVG
	disabled bool
	// plain draws the button without the accent behind it: a second action
	// beside the one the line is about. It keeps a border, so it reads as a
	// control rather than as a label.
	plain bool
	// small draws the button a size down, for a toolbar.
	small bool
	run   func()
}

// iconButton draws a button of the request lines: an icon beside its label.
func (a *app) iconButton(c *ui.Context, p design.Palette, spec iconButtonSpec) {
	pad, gap, iconSize, height := float32(12), float32(6), float32(14), design.RequestRowHeight
	font := design.FontSizeBody
	if spec.small {
		pad, gap, iconSize, height = 8, 4, 12, design.RequestRowHeight-12
		font = design.FontSizeTiny
	}
	width := 2*pad + iconSize + gap + measureText(c, spec.label, font, 400) + tabWiggle
	btn := ui.ButtonBase(c).Label(spec.name).Height(height).
		MinWidth(width).Gap(gap).Padding(0, pad).Radius(design.Radius).
		Cursor(ui.CursorPointer).Disabled(spec.disabled)
	ink := p.AccentText
	if spec.plain {
		// A plain button keeps the page behind it, with the border the other
		// fields have, and reads in the muted text color.
		ink = p.TextSecondary
		btn.Border(design.BorderWidth, p.Border)
		if btn.Hovered() && !spec.disabled {
			btn.Background(p.SurfaceHover)
		}
	} else {
		btn.Background(p.Accent)
		if btn.Hovered() && !spec.disabled {
			btn.Background(p.AccentHover)
		}
	}
	btn.Children(func() {
		ui.Icon(c, spec.icon).FontSize(iconSize).TextColor(ink)
		ui.Text(c, spec.label).FontSize(font).TextColor(ink).SingleLine().
			MinWidth(measureText(c, spec.label, font, 400) + tabWiggle)
	})
	if btn.Clicked() && !spec.disabled {
		spec.run()
	}
}

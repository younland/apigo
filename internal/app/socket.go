package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"apigo/internal/design"
	"apigo/internal/ws"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// socketLogKind is what a line of a socket's log says: the app's own notes
// beside what the socket reported.
type socketLogKind int

const (
	socketNote socketLogKind = iota // what the app is doing
	socketOpen                      // a socket that opened
	socketSent                      // a message that went out
	socketGot                       // a message that came back
	socketFail                      // something that went wrong
)

// wsEvent is one line of a socket's log. A message keeps the read-only editor
// it opens into, made when the reader expands the line.
type wsEvent struct {
	kind socketLogKind
	at   time.Time
	text string
	size int
	// line is text folded into the one line a collapsed row shows, cut to
	// summaryLine; the row cuts it further to its own width.
	line string
	view *editor.State
}

// message reports whether the line holds a message, which expands into its
// content.
func (e wsEvent) message() bool { return e.kind == socketSent || e.kind == socketGot }

// summary is what a collapsed line says of a message: its content on one line.
// The row ellipsizes what does not fit, so the window's width decides where a
// long message ends.
func (e wsEvent) summary() string {
	if e.line != "" {
		return e.line
	}
	// A line built by hand, as a test builds one, folded as it is drawn.
	return collapseText(e.text)
}

// summaryLine caps the folded line of a message: room enough for the widest
// window, so folding a huge message once does not keep all of it.
const summaryLine = 1024

// collapseText folds text into one line: runs of whitespace become a space, as
// a message written over several lines reads as one in its row.
func collapseText(text string) string {
	folded := strings.Join(strings.Fields(text), " ")
	runes := []rune(folded)
	if len(runes) > summaryLine {
		return string(runes[:summaryLine])
	}
	return folded
}

// socketFormats are the formats a message reads in, as the request pane lets
// one choose.
var socketFormats = []struct{ key, label, language string }{
	{"text", "Text", ""},
	{"json", "JSON", "JSON"},
	{"xml", "XML", "XML"},
	{"html", "HTML", "HTML"},
	{"binary", "Binary", ""},
}

// socketFormat is the format a key names, text for one it does not know.
func socketFormat(key string) (label, language string) {
	for _, format := range socketFormats {
		if format.key == key {
			return format.label, format.language
		}
	}
	return "Text", ""
}

// socketLogLimit caps the log of a socket: a long-lived connection must not
// grow the app's memory with every message.
const socketLogLimit = 500

// appendSocket adds a line to the log of a tab, dropping the oldest when the
// log is full. The line a row shows is folded here, once, rather than every
// frame.
func (t *tab) appendSocket(event wsEvent) {
	event.line = collapseText(event.text)
	t.socketLog = append(t.socketLog, event)
	if len(t.socketLog) > socketLogLimit {
		t.socketLog = t.socketLog[len(t.socketLog)-socketLogLimit:]
	}
}

// socketHandshake is what the dial sends: the headers of the request pane, and
// the subprotocol the protocol pane asks for.
func (t *tab) socketHandshake() (http.Header, []string) {
	headers := http.Header{}
	for _, row := range t.headers.Rows {
		if !row.Enabled || strings.TrimSpace(row.Key) == "" {
			continue
		}
		headers.Set(strings.TrimSpace(row.Key), row.Value)
	}
	protocols := []string{}
	if protocol := strings.TrimSpace(t.protocol); protocol != "" {
		protocols = append(protocols, protocol)
	}
	return headers, protocols
}

// connectSocket opens the tab's socket off the window's thread, keeping the
// connection and what it reports in the log.
func (a *app) connectSocket(c *ui.Context, t *tab) {
	if t.conn != nil || t.connecting {
		return
	}
	address := strings.TrimSpace(t.url)
	if address == "" {
		c.Toast("先填 WebSocket 地址")
		return
	}
	headers, protocols := t.socketHandshake()
	t.connecting = true
	t.appendSocket(wsEvent{kind: socketNote, at: time.Now(), text: "正在连接到 " + address})

	open := func() {
		conn, err := ws.Dial(context.Background(), address, headers, protocols)
		run := func() {
			t.connecting = false
			if err != nil {
				t.appendSocket(wsEvent{kind: socketFail, at: time.Now(), text: err.Error()})
				return
			}
			t.conn = conn
			t.appendSocket(wsEvent{kind: socketOpen, at: time.Now(), text: "已连接 " + address})
			go a.readSocket(t, conn)
		}
		if a.after == nil {
			run()
			return
		}
		a.after(run)
	}

	if a.after == nil {
		open()
		return
	}
	go open()
}

// readSocket turns what a socket reports into log lines, on the window's
// thread.
func (a *app) readSocket(t *tab, conn *ws.Conn) {
	for event := range conn.Events() {
		line, done := socketEventOf(event)
		run := func() {
			if t.conn != conn {
				return // the socket was replaced or closed: its tail is dropped
			}
			t.appendSocket(line)
			if done {
				t.conn = nil
			}
		}
		if a.after == nil {
			run()
			continue
		}
		a.after(run)
	}
}

// socketEventOf turns a socket's event into the line the log shows, and says
// whether the socket is over.
func socketEventOf(event ws.Event) (wsEvent, bool) {
	if event.Kind == ws.Disconnected {
		return wsEvent{kind: socketFail, at: event.At, text: event.Text}, true
	}
	return wsEvent{kind: socketGot, at: event.At, text: event.Text, size: event.Size}, false
}

// disconnectSocket closes a tab's socket and clears its log: a connection that
// is over leaves nothing to read.
func (a *app) disconnectSocket(t *tab) {
	conn := t.conn
	if conn == nil {
		return
	}
	t.conn = nil
	_ = conn.Close()
	t.socketLog = nil
}

// toggleLogLine opens a logged message into its content, or closes it again. A
// JSON message opens indented, as the response preview indents one, so a long
// object reads as a tree rather than one wrapped line.
func (a *app) toggleLogLine(t *tab, i int) {
	if i < 0 || i >= len(t.socketLog) {
		return
	}
	line := &t.socketLog[i]
	if line.view != nil {
		line.view = nil
		return
	}
	buffer := editor.NewBuffer(prettyResponse(line.text))
	buffer.SetLanguage(t.logLanguage(line.text))
	line.view = &editor.State{
		Buffer: buffer,
		// A logged message reads a size down from the editor's own 13, so more
		// of it fits under the line it belongs to.
		Font:       ui.Font{Size: design.FontSizeBody - 1},
		ReadOnly:   true,
		SoftWrap:   true,
		FitContent: true,
	}
}

// logLanguage is the syntax a logged message reads in: JSON when the message is
// JSON, else the format the socket sends in.
func (t *tab) logLanguage(message string) *editor.Language {
	if json.Valid([]byte(message)) {
		return editor.LanguageNamed("JSON")
	}
	return socketLanguage(t.bodyType)
}

// sendSocketMessage sends what the message pane holds, and clears it when the
// tab asks for that.
func (a *app) sendSocketMessage(c *ui.Context, t *tab) {
	if t.conn == nil {
		c.Toast("先连接再发送")
		return
	}
	text := t.body.String()
	if strings.TrimSpace(text) == "" {
		return
	}
	if t.bodyType == "binary" {
		c.Toast("二进制消息暂不支持发送")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := t.conn.Send(ctx, false, []byte(text)); err != nil {
		t.appendSocket(wsEvent{kind: socketFail, at: time.Now(), text: err.Error()})
		return
	}
	t.appendSocket(wsEvent{kind: socketSent, at: time.Now(), text: text, size: len(text)})
	if t.clearOnSend {
		t.body.Set("")
	}
}

// closeSocket lets a tab that is going away drop its socket, without a log
// line: the tab is gone.
func (t *tab) closeSocket() {
	if t.conn == nil {
		return
	}
	_ = t.conn.Close()
	t.conn = nil
}

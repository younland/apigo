package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
	"github.com/gorilla/websocket"
)

// echoSocket is a WebSocket server that sends back what it reads.
func echoSocket(t *testing.T) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer socket.Close()
		for {
			kind, data, err := socket.ReadMessage()
			if err != nil {
				return
			}
			if err := socket.WriteMessage(kind, data); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// socketAddress is the ws address of a test server.
func socketAddress(server *httptest.Server) string {
	return "ws" + strings.TrimPrefix(server.URL, "http")
}

// socketApp is an app with a WebSocket request open on the address, and the
// work the socket asks for queued for the test to run.
func socketApp(t *testing.T, address string) (*app, <-chan func()) {
	t.Helper()
	a := testApp(t)
	pending := make(chan func(), 8)
	a.after = func(run func()) { pending <- run }
	if err := a.addRequest(kindWebSocket, ""); err != nil {
		t.Fatal(err)
	}
	a.current().url = address
	return a, pending
}

// TestSocketConnectsAndEchoes checks a socket connects, says so in its log, and
// shows a message both ways.
func TestSocketConnectsAndEchoes(t *testing.T) {
	server := echoSocket(t)
	a, pending := socketApp(t, socketAddress(server))
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("连接"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending) // the dial comes back as a log line
	tt.Frame()
	if a.current().conn == nil {
		t.Fatal("no socket after connecting")
	}
	if !tt.HasText("已连接 " + socketAddress(server)) {
		t.Errorf("texts %q, want the connection in the log", tt.Texts())
	}

	a.current().body.Set("你好")
	tt.Frame()
	if err := tt.Click("发送消息"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending) // the echo comes back
	tt.Frame()
	if !tt.HasText("你好") {
		t.Errorf("texts %q, want the message echoed back", tt.Texts())
	}
	if err := tt.Click("日志 你好"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tab := a.current()
	if !tt.HasText("消息大小") || tab.logIndex < 0 || formatSize(tab.socketLog[tab.logIndex].size) != "6 B" {
		t.Errorf("texts %q and selected log %d, want the echoed message size in the detail", tt.Texts(), tab.logIndex)
	}
	if got := a.current().body.String(); got != "" {
		t.Errorf("message %q after sending, want it cleared", got)
	}

	if err := tt.Click("断开连接"); err != nil {
		t.Fatal(err)
	}
	if a.current().conn != nil {
		t.Error("the socket is still open after 断开连接")
	}
}

// TestSocketLogClears checks the log of a socket empties on 清空.
func TestSocketLogClears(t *testing.T) {
	// A server that refuses the handshake: the log says so.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	a, pending := socketApp(t, socketAddress(server))
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("连接"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending)
	tt.Frame()
	if !tt.HasText("正在连接到 " + socketAddress(server)) {
		t.Fatalf("texts %q, want the attempt in the log", tt.Texts())
	}
	if err := tt.Click("日志 正在连接到 " + socketAddress(server)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.current().logView == nil {
		t.Fatal("the log note did not open in the detail panel")
	}

	if err := tt.Click("清空日志"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("正在连接到 " + socketAddress(server)) {
		t.Error("the log still holds what 清空日志 cleared")
	}
	if a.current().logView != nil {
		t.Error("clearing the log left its detail panel open")
	}
}

// TestSocketMessageFormatFollowsThePick checks the format of a message is kept
// in the request file, and that a binary message is refused.
func TestSocketMessageFormatFollowsThePick(t *testing.T) {
	server := echoSocket(t)
	a, pending := socketApp(t, socketAddress(server))
	tt := ui.NewTester(a.view, 1280, 800)

	// A socket sends JSON unless the request says otherwise.
	if got := a.current().bodyType; got != "json" {
		t.Fatalf("format %q of a new socket, want json", got)
	}

	if err := tt.Click("连接"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending)
	a.current().body.Set(`{"ping":true}`)
	tt.Frame()
	if err := tt.Click("发送消息"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending)
	tt.Frame()
	if !tt.HasText(`{"ping":true}`) {
		t.Errorf("texts %q, want the JSON message echoed", tt.Texts())
	}
	if err := tt.Click(`日志 {"ping":true}`); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tab := a.current()
	if !tt.HasText("消息大小") || tab.logIndex < 0 || formatSize(tab.socketLog[tab.logIndex].size) != "13 B" {
		t.Errorf("texts %q and selected log %d, want the JSON message size in the detail", tt.Texts(), tab.logIndex)
	}

	// A binary message is refused with a reason rather than sent empty: open the
	// menu the format picks, and pick Binary.
	if err := tt.Click("消息格式 JSON"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("消息格式 Binary"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("二进制消息暂不支持发送") {
		t.Errorf("texts %q, want the note about binary messages", tt.Texts())
	}
}

// TestSocketSubprotocolReachesTheHandshake checks the protocol pane's
// subprotocol is asked for when the socket opens.
func TestSocketSubprotocolReachesTheHandshake(t *testing.T) {
	asked := make(chan string, 1)
	upgrader := websocket.Upgrader{
		Subprotocols: []string{"chat"},
		CheckOrigin:  func(*http.Request) bool { return true },
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		asked <- socket.Subprotocol()
		_ = socket.Close()
	}))
	defer server.Close()

	a, pending := socketApp(t, socketAddress(server))
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("协议"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("子协议输入"); err != nil {
		t.Fatal(err)
	}
	tt.Type("chat")
	if err := tt.Click("通讯"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("连接"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending)

	select {
	case got := <-asked:
		if got != "chat" {
			t.Errorf("subprotocol %q, want chat", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the handshake never reached the server")
	}
}

// TestSocketRequestKeepsItsMessage checks a socket's message and subprotocol
// are written to its interface file and read back.
func TestSocketRequestKeepsItsMessage(t *testing.T) {
	dir := t.TempDir()
	a := newApp()
	a.store = store.New(dir)
	if err := a.store.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.store.NewCollection("Demo"); err != nil {
		t.Fatal(err)
	}
	a.openCollection("Demo")
	if err := a.addRequest(kindWebSocket, ""); err != nil {
		t.Fatal(err)
	}
	t2 := a.current()
	t2.title, t2.url, t2.protocol, t2.bodyType = "聊天", "ws://x.test/chat", "chat", "json"
	t2.body.Set(`{"hi":true}`)
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	// The save button of the request line asks where the interface goes.
	if err := tt.Click("保存接口"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("确认保存"); err != nil {
		t.Fatal(err)
	}

	reopened := newApp()
	reopened.store = store.New(dir)
	if err := reopened.store.Load(); err != nil {
		t.Fatal(err)
	}
	col := reopened.store.Collection("Demo")
	if col == nil || len(col.Root.Requests) != 1 {
		t.Fatalf("collection %+v, want the socket request saved", col)
	}
	saved := col.Root.Requests[0]
	if saved.Kind != kindWebSocket || saved.Protocol != "chat" ||
		saved.BodyType != "json" || saved.Body != `{"hi":true}` {
		t.Errorf("saved %+v, want the socket's protocol, format and message", saved)
	}
}

// TestSocketLogLineOpensRightPanel checks a message opens in the one code
// panel beside the log, another message replaces its content, and closing the
// panel gives the log its room back.
func TestSocketLogLineOpensRightPanel(t *testing.T) {
	server := echoSocket(t)
	a, pending := socketApp(t, socketAddress(server))
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("连接"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending)
	a.current().body.Set(`{"ping":true}`)
	tt.Frame()
	if err := tt.Click("发送消息"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending) // the echo: the sent line is added on this thread
	tt.Frame()

	if err := tt.Click(`日志 {"ping":true}`); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tab := a.current()
	if tab.logView == nil {
		t.Fatal("clicking a message opened no right panel")
	}
	if got, want := tab.logView.Buffer.String(), "{\n  \"ping\": true\n}"; got != want {
		t.Errorf("panel holds %q, want the whole message indented %q", got, want)
	}
	if !tab.logView.ReadOnly {
		t.Error("the panel message is editable, want it read-only")
	}
	if !tt.HasText("消息详情") {
		t.Errorf("texts %q, want the right panel title", tt.Texts())
	}
	if !tt.HasText("消息大小") {
		t.Errorf("texts %q, want the selected message size in the detail", tt.Texts())
	}

	firstView := tab.logView
	tab.appendSocket(wsEvent{kind: socketGot, at: time.Now(), text: `{"other":true}`})
	tt.Frame()
	if err := tt.Click(`日志 {"other":true}`); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tab.logView != firstView {
		t.Error("another message created a second editor, want the one panel updated")
	}
	if got := tab.logView.Buffer.String(); got != "{\n  \"other\": true\n}" {
		t.Errorf("panel holds %q after another message, want the new content", got)
	}
	if err := tt.Click("关闭消息详情"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tab.logView != nil || tt.HasText("消息详情") {
		t.Error("the right panel remained after closing it")
	}
}

func TestSocketLogDetailResizesWithinTwoThirds(t *testing.T) {
	a := testApp(t)
	if err := a.addRequest(kindWebSocket, ""); err != nil {
		t.Fatal(err)
	}
	a.current().socketLog = []wsEvent{{kind: socketGot, at: time.Now(), text: `{"pong":true}`}}
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	if err := tt.Click(`日志 {"pong":true}`); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	panel, ok := tt.Find("日志详情面板")
	if !ok {
		t.Fatal("no right detail panel")
	}
	before := a.current().logListPercent
	tt.Press(panel.X-1, panel.Y+80)
	tt.Move(panel.X-200, panel.Y+80)
	tt.Release(panel.X-200, panel.Y+80)
	tt.Frame()
	if got := a.current().logListPercent; got >= before {
		t.Errorf("list percentage %v after widening the detail, want less than %v", got, before)
	}
	panel, _ = tt.Find("日志详情面板")
	tt.Press(panel.X-1, panel.Y+80)
	tt.Move(0, panel.Y+80)
	tt.Release(0, panel.Y+80)
	tt.Frame()
	if got := a.current().logListPercent; got != minLogListPercent {
		t.Errorf("list percentage %v beyond the detail cap, want %v", got, minLogListPercent)
	}
}

// TestSocketDisconnectClearsTheLog checks closing the socket leaves an empty
// log behind.
func TestSocketDisconnectClearsTheLog(t *testing.T) {
	server := echoSocket(t)
	a, pending := socketApp(t, socketAddress(server))
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("连接"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending)
	tt.Frame()
	if len(a.current().socketLog) == 0 {
		t.Fatal("no log after connecting")
	}

	if err := tt.Click("断开连接"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.current().socketLog) != 0 {
		t.Errorf("log %d lines after disconnecting, want it cleared", len(a.current().socketLog))
	}
	if !tt.HasText("连接后，消息与连接事件会显示在这里") {
		t.Errorf("texts %q, want the empty log", tt.Texts())
	}
}

// TestSocketMessageFollowsTheFormat checks the message editor takes the syntax
// and the hint of the format the pane picked.
func TestSocketMessageFollowsTheFormat(t *testing.T) {
	a := testApp(t)
	if err := a.addRequest(kindWebSocket, ""); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1280, 800)
	if got := a.current().bodyEdit.Placeholder; !strings.Contains(got, `"key"`) {
		t.Errorf("placeholder %q of a new socket, want the JSON one", got)
	}

	if err := tt.Click("消息格式 JSON"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("消息格式 XML"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := a.current().bodyEdit.Placeholder; !strings.Contains(got, "<note>") {
		t.Errorf("placeholder %q after picking XML, want the XML one", got)
	}
}

// TestSocketLogRowsTintUnderThePointer checks the lines of the log answer the
// pointer, as the rows of a list do.
func TestSocketLogRowsTintUnderThePointer(t *testing.T) {
	a := testApp(t)
	if err := a.addRequest(kindWebSocket, ""); err != nil {
		t.Fatal(err)
	}
	a.current().socketLog = []wsEvent{
		{kind: socketGot, at: time.Now(), text: `{"pong":true}`, size: 13},
	}
	tt := ui.NewTester(a.view, 1280, 800)
	page, hover := a.palette().Background, a.palette().SurfaceHover

	tt.Move(700, 400)
	tt.Frame()
	if got := rowBackground(t, tt, `日志 {"pong":true}`); !sameColor(got, page) {
		t.Errorf("line background %v away from the pointer, want %v", got, page)
	}

	row, _ := tt.Find(`日志 {"pong":true}`)
	tt.Move(row.X+row.W/2, row.Y+row.H/2)
	tt.Frame()
	if got := rowBackground(t, tt, `日志 {"pong":true}`); !sameColor(got, hover) {
		t.Errorf("line background %v under the pointer, want %v", got, hover)
	}
}

// TestSocketLogFilters checks the log keeps the lines whose message holds what
// the filter asks for, and says so when none does.
func TestSocketLogFilters(t *testing.T) {
	a := testApp(t)
	if err := a.addRequest(kindWebSocket, ""); err != nil {
		t.Fatal(err)
	}
	a.current().socketLog = []wsEvent{
		{kind: socketOpen, at: time.Now(), text: "已连接 ws://127.0.0.1:6976/ws"},
		{kind: socketSent, at: time.Now(), text: `{"ping":true}`, size: 13},
		{kind: socketGot, at: time.Now(), text: `{"type":7,"body":{"state":"idle"}}`, size: 35},
	}
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	if !tt.HasText("3 条") {
		t.Fatalf("texts %q, want the whole log counted", tt.Texts())
	}

	// The filter is part of the request tab's state: typing into it drops the
	// lines whose message does not hold the query.
	a.current().logFilter = "idle"
	tt.Frame()
	if !tt.HasText("1 / 3 条") {
		t.Errorf("texts %q, want the count to say what the filter kept", tt.Texts())
	}
	if !tt.HasText(`{"type":7,"body":{"state":"idle"}}`) {
		t.Errorf("texts %q, want the line the filter matched", tt.Texts())
	}
	if tt.HasText("已连接 ws://127.0.0.1:6976/ws") || tt.HasText(`{"ping":true}`) {
		t.Errorf("texts %q, want the other lines gone", tt.Texts())
	}
	if err := tt.Click(`日志 {"type":7,"body":{"state":"idle"}}`); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.current().logView == nil {
		t.Fatal("the matching log did not open in the detail panel")
	}

	a.current().logFilter = "NOPE"
	tt.Frame()
	if !tt.HasText("没有匹配「NOPE」的日志") {
		t.Errorf("texts %q, want the empty state of a filter that matches nothing", tt.Texts())
	}
	if a.current().logView != nil {
		t.Error("filtering out the selected log left its detail open")
	}
}

// TestSocketLogSummaryKeepsTheWholeMessage checks a collapsed line carries its
// whole message onto one line, so the row's own width decides where a long one
// ends, and that a huge message is still folded short of the log's memory.
func TestSocketLogSummaryKeepsTheWholeMessage(t *testing.T) {
	a := testApp(t)
	if err := a.addRequest(kindWebSocket, ""); err != nil {
		t.Fatal(err)
	}
	tab := a.current()
	message := strings.Repeat("ab", 150) // 300 runes: past the 72 a row used to keep
	tab.appendSocket(wsEvent{kind: socketGot, text: "{\n  \"k\": \"" + message + "\"\n}", size: 42})
	folded := tab.socketLog[0].summary()
	if !strings.Contains(folded, message) {
		t.Errorf("summary %q, want the whole message", folded)
	}
	if strings.ContainsAny(folded, "\n\t") {
		t.Errorf("summary %q, want its whitespace folded", folded)
	}

	huge := strings.Repeat("x", summaryLine*3)
	tab.appendSocket(wsEvent{kind: socketGot, text: huge})
	if got := len([]rune(tab.socketLog[1].summary())); got != summaryLine {
		t.Errorf("a huge message keeps %d runes in its row, want it cut at %d", got, summaryLine)
	}
}

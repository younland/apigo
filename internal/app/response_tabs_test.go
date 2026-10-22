package app

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"apigo/internal/widgets"

	"github.com/egoist/mygo/ui"
)

// TestSendButtonCancelsTheRequest checks the send button becomes a cancel one
// while a request waits, still in the accent color, and that it stops the
// request.
func TestSendButtonCancelsTheRequest(t *testing.T) {
	hang := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-hang
	}))
	defer server.Close()
	defer close(hang)

	a := testApp(t)
	pending := make(chan func(), 4)
	a.after = func(run func()) { pending <- run }
	a.current().url = server.URL
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if !a.current().loading {
		t.Fatal("the request is not waiting")
	}
	if _, ok := tt.Find("取消"); !ok {
		t.Fatalf("texts %q, want a cancel button while the request waits", tt.Texts())
	}
	if _, ok := tt.Find("发送"); ok {
		t.Fatalf("texts %q, want the send button gone while the request waits", tt.Texts())
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending)

	res := a.current().resp
	if a.current().loading || res == nil {
		t.Fatalf("loading %v response %v, want the cancelled request recorded", a.current().loading, res)
	}
	if res.Err == nil || res.Err.Error() != "请求已取消" {
		t.Errorf("error %v, want 请求已取消", res.Err)
	}
	if a.current().cancel != nil {
		t.Error("the tab still holds the cancel of a finished request")
	}
	tt.Frame()
	if !tt.HasText("重试") {
		t.Errorf("texts %q, want the way to send it again", tt.Texts())
	}
}

// TestResponseTabsShowTheBodyItsHeadersAndTheRequest checks the response panes
// switch: the preview, the body as it came, the response headers, and the
// headers the request went out with.
func TestResponseTabsShowTheBodyItsHeadersAndTheRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Reply", "yes")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	a := testApp(t)
	a.current().url = server.URL
	a.current().headers.Rows = []widgets.Row{{Enabled: true, Key: "X-Token", Value: "t"}}
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}

	if got := a.current().respEdit.Buffer.String(); got != "{\n  \"ok\": true\n}" {
		t.Errorf("preview %q, want the body indented", got)
	}

	if err := tt.Click("响应页签 原始内容"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().respEdit.Buffer.String(); got != "{\n  \"ok\": true\n}" {
		t.Errorf("preview %q after switching, want it kept", got)
	}
	if got := a.current().rawEdit.Buffer.String(); got != `{"ok":true}` {
		t.Errorf("raw body %q, want the body as it came", got)
	}

	if err := tt.Click("响应页签 响应头"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("响应头列表") || !tt.HasText("x-reply") || !tt.HasText("yes") {
		t.Errorf("texts %q, want the response headers listed", tt.Texts())
	}

	if err := tt.Click("响应页签 请求头"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("请求头列表") || !tt.HasText("x-token") || !tt.HasText("t") {
		t.Errorf("texts %q, want the headers the request went out with", tt.Texts())
	}
}

// TestResponseLanguageFollowsTheContentType checks the preview picks the syntax
// of the body by what the server says it is, and falls back to plain text.
func TestResponseLanguageFollowsTheContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{"json", "application/json", `{"ok":true}`, "JSON"},
		{"json by the body", "", `{"ok":true}`, "JSON"},
		{"xml", "application/xml", `<a/>`, "XML"},
		{"svg", "image/svg+xml", `<svg/>`, "XML"},
		{"html", "text/html", `<p>hi</p>`, "HTML"},
		{"css", "text/css", `a{}`, "CSS"},
		{"javascript", "application/javascript", `1`, "JavaScript"},
		{"plain text", "text/plain", "hi", ""},
		{"unknown", "application/octet-stream", "hi", ""},
	}
	for _, test := range tests {
		lang := languageForResponse(test.contentType, test.body)
		got := ""
		if lang != nil {
			got = lang.Name
		}
		if got != test.want {
			t.Errorf("%s: language %q, want %q", test.name, got, test.want)
		}
	}
}

// TestResponseTabsAreFixed checks the panes keep their names whatever the body
// holds, the first one always being the preview.
func TestResponseTabsAreFixed(t *testing.T) {
	if len(responseTabNames) != 4 || responseTabNames[0] != "预览" {
		t.Errorf("tab names %q, want 预览 first", responseTabNames)
	}
	for _, name := range []string{"原始内容", "响应头", "请求头"} {
		found := false
		for _, have := range responseTabNames {
			found = found || have == name
		}
		if !found {
			t.Errorf("tab names %q, want %q among them", responseTabNames, name)
		}
	}
}

// TestHeaderTabCountsWhatItLists checks the header tabs say how many headers
// they hold.
func TestHeaderTabCountsWhatItLists(t *testing.T) {
	t2 := newTab()
	t2.resp = &response{Status: 200, Body: "{}", Raw: []byte("{}"),
		Headers: http.Header{"Content-Type": {"application/json"}, "ETag": {"x"}}}
	t2.sentHeaders = http.Header{"X-Token": {"t"}}
	badges := []string{"", "", countBadge(len(t2.resp.Headers)), countBadge(len(t2.sentHeaders))}
	if badges[2] != "2" || badges[3] != "1" {
		t.Errorf("badges %q, want the counts of both header lists", badges)
	}
	if got := countBadge(0); got != "" {
		t.Errorf("countBadge(0) = %q, want nothing", got)
	}
}

// TestResponseTabsKeepTheirNames pins the width of the tabs across window
// sizes: the names are as wide as the font needs, so the panel squeezes
// something else instead of turning them into 「原始…」. (The truncation itself
// shows only in a real window; the offscreen tester lays the tabs out with room
// to spare, so this pins the widths rather than the pixels.)
func TestResponseTabsKeepTheirNames(t *testing.T) {
	a := testApp(t)
	a.showResponse(a.current(), &response{
		Status: 200, StatusText: "OK", Body: "{}", Raw: []byte("{}"),
		Headers: manyHeaders(20),
	})
	names := []string{"预览", "原始内容", "响应头", "请求头"}

	// The width each tab takes when the window has room to spare.
	wide := ui.NewTester(a.view, 1280, 800)
	wide.Frame()
	roomy := map[string]float32{}
	for _, name := range names {
		rect, ok := wide.Find(responseTabLabel(name))
		if !ok {
			t.Fatalf("no tab %q", name)
		}
		roomy[name] = rect.W
	}

	// A narrow window squeezes everything else, not the names of the tabs.
	narrow := ui.NewTester(a.view, 1024, 700)
	narrow.Frame()
	for _, name := range names {
		rect, ok := narrow.Find(responseTabLabel(name))
		if !ok {
			t.Fatalf("no tab %q in the narrow window", name)
		}
		if rect.W != roomy[name] {
			t.Errorf("tab %q is %v wide in the narrow window, want the %v its name needs",
				name, rect.W, roomy[name])
		}
	}
}

// manyHeaders is a response with n headers, as a busy server sends.
func manyHeaders(n int) http.Header {
	headers := http.Header{}
	for i := 0; i < n; i++ {
		headers["X-Header-"+strconv.Itoa(i)] = []string{"v"}
	}
	return headers
}

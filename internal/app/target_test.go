package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// TestSendReadsPathUnderBaseURL checks an address that starts with a slash goes
// out under the baseurl of the environment in use.
func TestSendReadsPathUnderBaseURL(t *testing.T) {
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := testApp(t)
	a.store.Environments[0].Vars = []savedRow{{Enabled: true, Key: store.BaseURLVariable, Value: server.URL}}
	a.current().url = "/pets?limit=10"
	tt := ui.NewTester(a.view, 1280, 800)

	// The value the path hangs from stands before the field, read-only.
	if !tt.HasText(server.URL) {
		t.Errorf("texts %q, want the baseurl before the address", tt.Texts())
	}
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if path := <-received; path != "/pets" {
		t.Errorf("path %q, want the path under the baseurl", path)
	}
}

// TestSendReadsAddressWithoutScheme checks a baseurl written as a host and port
// is read as http.
func TestSendReadsAddressWithoutScheme(t *testing.T) {
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := testApp(t)
	host := strings.TrimPrefix(server.URL, "http://")
	a.store.Environments[0].Vars = []savedRow{{Enabled: true, Key: store.BaseURLVariable, Value: host}}
	a.current().url = "/orders"
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if path := <-received; path != "/orders" {
		t.Errorf("path %q, want the path under the host of the baseurl", path)
	}
}

// TestSendRefusesAPathWithoutBaseURL checks a path with no prefix URL is refused
// and says what the environment is missing.
func TestSendRefusesAPathWithoutBaseURL(t *testing.T) {
	a := testApp(t)
	a.store.Environments[0].Vars = []savedRow{{Enabled: true, Key: store.BaseURLVariable}}
	a.current().url = "/pets"
	tt := ui.NewTester(a.view, 1280, 800)

	// With nothing to hang from, the field shows the path alone.
	if tt.HasText("前置URL") || tt.HasText("未设置") {
		t.Errorf("texts %q, want no prefix in the address field", tt.Texts())
	}
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if res := a.current().resp; res != nil || a.current().loading {
		t.Errorf("response %+v loading %v, want the path refused", res, a.current().loading)
	}
	if !tt.HasText("没有设置前置URL") {
		t.Errorf("texts %q, want the missing prefix URL reported", tt.Texts())
	}
}

// TestRequestLinePrefixFollowsTheAddress checks the prefix URL shows inside the
// address field only while the address is a path under it.
func TestRequestLinePrefixFollowsTheAddress(t *testing.T) {
	a := testApp(t)
	a.store.Environments[0].Vars = []savedRow{
		{Enabled: true, Key: store.BaseURLVariable, Value: "http://localhost:8080"},
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if tt.HasText("http://localhost:8080") {
		t.Error("prefix shown for an address that is not a path")
	}
	a.current().url = "/pets"
	tt.Frame()
	prefix, ok := tt.Find("http://localhost:8080")
	if !ok {
		t.Fatalf("texts %q, want the prefix inside the field", tt.Texts())
	}
	field, _ := tt.Find("输入请求 URL")
	if prefix.X < field.X {
		t.Errorf("prefix at %v, want it inside the address field at %v", prefix, field)
	}
	a.current().url = "https://api.example.com/pets"
	tt.Frame()
	if tt.HasText("http://localhost:8080") {
		t.Error("prefix shown after the address became absolute")
	}
}

// TestSocketConnectsUnderBaseURL checks a socket address that starts with a
// slash opens on the baseurl of the environment, as a request does.
func TestSocketConnectsUnderBaseURL(t *testing.T) {
	server := echoSocket(t)
	a, pending := socketApp(t, "")
	host := strings.TrimPrefix(socketAddress(server), "ws://")
	a.store.Environments[0].Vars = []savedRow{{Enabled: true, Key: store.BaseURLVariable, Value: host}}
	a.current().url = "/ws"
	tt := ui.NewTester(a.view, 1280, 800)

	if !tt.HasText(host) {
		t.Errorf("texts %q, want the baseurl before the socket address", tt.Texts())
	}
	if err := tt.Click("连接"); err != nil {
		t.Fatal(err)
	}
	waitForApply(t, pending)
	if a.current().conn == nil {
		t.Fatalf("no socket, log %+v", a.current().socketLog)
	}
}

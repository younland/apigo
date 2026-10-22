package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apigo/internal/req"
	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// TestResponseLimitFollowsSettings checks how much of a response the app reads
// comes from the settings, with the default standing in where they say nothing
// usable.
func TestResponseLimitFollowsSettings(t *testing.T) {
	tests := []struct {
		name  string
		saved int
		want  int64
	}{
		{"the default", 0, defaultResponseLimitMB << 20},
		{"saved", 5, 5 << 20},
		{"below the smallest", -3, defaultResponseLimitMB << 20},
		{"above the largest", 99999, maxResponseLimitMB << 20},
	}
	for _, test := range tests {
		a := newApp()
		a.store = store.New(t.TempDir())
		a.store.Settings = settings{MaxResponseMB: test.saved}
		a.applySettings()
		if got := a.responseLimit(); got != test.want {
			t.Errorf("%s: responseLimit = %d, want %d", test.name, got, test.want)
		}
	}
}

// TestResponseLimitIsRemembered checks the limit the settings edited is kept for
// the next run.
func TestResponseLimitIsRemembered(t *testing.T) {
	dir := t.TempDir()
	a := newApp()
	a.store = store.New(dir)
	a.maxResponseMB = 32
	if err := a.rememberSettings(); err != nil {
		t.Fatal(err)
	}

	loaded := store.New(dir)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if got := loaded.Settings.MaxResponseMB; got != 32 {
		t.Errorf("saved limit %d, want 32", got)
	}
}

// TestSendCutsTheBodyAtTheLimit checks a body past the limit reaches the pane
// cut, with the cut said in the response.
func TestSendCutsTheBodyAtTheLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 1<<20+100)))
	}))
	defer server.Close()

	a := testApp(t)
	a.maxResponseMB = minResponseLimitMB // one megabyte, so the test stays small
	a.current().url = server.URL
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}

	res := a.current().resp
	if res == nil {
		t.Fatal("no response recorded after sending")
	}
	if want := int64(minResponseLimitMB << 20); int64(len(res.Raw)) != want {
		t.Errorf("body %d bytes, want the %d of the limit", len(res.Raw), want)
	}
	if res.Err == nil || !strings.Contains(res.Err.Error(), "1 MB") {
		t.Errorf("error %v, want one naming the 1 MB limit", res.Err)
	}
	if !tt.HasText("响应体超过 1 MB，已截断") {
		t.Errorf("texts %q, want the cut in the response pane", tt.Texts())
	}
}

// TestDefaultLimitMatchesTheRequestPackage checks the settings' default is the
// limit the request package reads a body under: 100 MB.
func TestDefaultLimitMatchesTheRequestPackage(t *testing.T) {
	if got := int64(defaultResponseLimitMB) << 20; got != req.DefaultMaxResponse {
		t.Errorf("settings default %d bytes, want the request package's %d", got, req.DefaultMaxResponse)
	}
	if req.DefaultMaxResponse != 100<<20 {
		t.Errorf("default response limit %d, want 100 MB", req.DefaultMaxResponse)
	}
}

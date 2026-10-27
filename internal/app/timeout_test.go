package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestRequestTimeoutDefaultsToHalfAnHour checks the timeout the settings hold
// when they name none, and the bounds they are read within.
func TestRequestTimeoutDefaultsToHalfAnHour(t *testing.T) {
	a := newApp()
	if got := a.requestTimeout(); got != 30*time.Minute {
		t.Errorf("timeout %v of a new app, want 30 分钟", got)
	}
	if got := requestTimeoutSeconds(0); got != defaultTimeoutSeconds {
		t.Errorf("seconds %d of a setting that holds none, want the default", got)
	}
	if got := requestTimeoutSeconds(-3); got != defaultTimeoutSeconds {
		t.Errorf("seconds %d of a negative setting, want the default", got)
	}
	if got := requestTimeoutSeconds(maxTimeoutSeconds + 1); got != maxTimeoutSeconds {
		t.Errorf("seconds %d past the largest, want it capped", got)
	}
}

// TestRequestTimeoutGivesUpOnASlowServer checks a request is given up when the
// timeout passes, saying so rather than waiting forever.
func TestRequestTimeoutGivesUpOnASlowServer(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // answers only once the test lets it, which it never does
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	defer close(release)

	a := testApp(t)
	a.timeoutSeconds = 0.2
	a.current().url = server.URL + "/slow"
	tt := ui.NewTester(a.view, 1280, 800)

	start := time.Now()
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("the request was waited for %v, want it given up at the timeout", elapsed)
	}
	res := a.current().resp
	if res == nil || res.Err == nil {
		t.Fatalf("response %+v, want the timeout reported", res)
	}
	if !strings.Contains(res.Err.Error(), "请求超时") {
		t.Errorf("error %q, want 请求超时", res.Err)
	}
	if !tt.HasText("请求超时") {
		t.Errorf("texts %q, want the timeout in the response pane", tt.Texts())
	}
}

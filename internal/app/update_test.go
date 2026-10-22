package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// TestNewerVersion checks the order of versions is by their numbers, not their
// text, and that a release that cannot be read is never newer.
func TestNewerVersion(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.2.0", "0.1.0", true},
		{"v0.2.0", "0.1.0", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.1", "0.1.0", true},
		{"0.1.0", "0.1.1", false},
		{"1.10.0", "1.9.9", true},
		{"1.9.9", "1.10.0", false},
		{"2", "1.9.9", true},
		{"0.1.0", "", true},
		{"", "0.1.0", false},
		{"nightly", "0.1.0", false},
		{"0.2.0-beta.1", "0.1.0", true},
	}
	for _, tc := range cases {
		if got := newerVersion(tc.latest, tc.current); got != tc.want {
			t.Errorf("newerVersion(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

// TestReleaseAtReadsGitHub checks the newest release is read from GitHub's
// answer, and that what is not published to update to reads as nothing.
func TestReleaseAtReadsGitHub(t *testing.T) {
	body := `{"tag_name":"v0.2.0","html_url":"https://example.com/releases/v0.2.0",
		"body":"- 修复若干问题","draft":false,"prerelease":false}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Error("the request names no user agent, want one GitHub accepts")
		}
		w.Write([]byte(body))
	}))
	defer server.Close()

	rel, err := releaseAt(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version != "0.2.0" || rel.URL != "https://example.com/releases/v0.2.0" {
		t.Errorf("release %+v, want 0.2.0 of the page GitHub names", rel)
	}

	// A draft, a prerelease and a namespace with no release have nothing to
	// update to.
	for _, answer := range []string{
		`{"tag_name":"v0.3.0","html_url":"https://example.com","draft":true}`,
		`{"tag_name":"v0.3.0","html_url":"https://example.com","prerelease":true}`,
		`{"html_url":"https://example.com"}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(answer))
		}))
		rel, err := releaseAt(context.Background(), server.URL)
		server.Close()
		if err != nil {
			t.Fatalf("%s: %v", answer, err)
		}
		if rel.Version != "" {
			t.Errorf("%s reads as release %+v, want nothing to update to", answer, rel)
		}
	}

	// A refusal is reported rather than taken for an up-to-date app.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer bad.Close()
	if _, err := releaseAt(context.Background(), bad.URL); err == nil {
		t.Error("a refused request reads as no error, want it reported")
	}
}

// updateApp is the app with the settings open, and the release its check finds.
func updateApp(t *testing.T, rel release, err error) (*app, *ui.Tester) {
	t.Helper()
	a := testApp(t)
	a.settingsOpen = true
	a.checkRelease = func(context.Context) (release, error) { return rel, err }
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	return a, tt
}

// TestAutoUpdateIsOffUntilTurnedOn checks the check for a newer version is off
// by default, that turning it on survives a restart, and that the button says
// what it found.
func TestAutoUpdateIsOffUntilTurnedOn(t *testing.T) {
	a, tt := updateApp(t, release{Version: "0.2.0", URL: "https://example.com/v0.2.0"}, nil)
	if a.autoUpdate {
		t.Fatal("auto update is on in a new app, want it off unless asked for")
	}
	if err := tt.Click("自动更新"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.autoUpdate {
		t.Error("the switch did not turn auto update on")
	}
	loaded := store.New(a.store.Dir)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if !loaded.Settings.AutoUpdate {
		t.Error("auto update was not written to the settings file")
	}

	// The button checks now, and says what it found: a newer version offers the
	// page it is published on.
	if err := tt.Click("检查更新"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := a.updateNote; got != "有新版本 0.2.0" {
		t.Errorf("note %q after the check, want the newer version", got)
	}
	if !tt.HasText("发现新版本 0.2.0") {
		t.Errorf("texts %q, want the question about the newer version", tt.Texts())
	}

	// An up-to-date app says so, and a failed check says why rather than
	// claiming the app is current.
	a.checkRelease = func(context.Context) (release, error) { return release{}, nil }
	a.checkUpdates(true)
	tt.Frame()
	if got := a.updateNote; !strings.Contains(got, "已是最新版本") {
		t.Errorf("note %q after an up-to-date check, want it said", got)
	}
	a.checkRelease = func(context.Context) (release, error) {
		return release{}, errors.New("网络不可达")
	}
	a.checkUpdates(true)
	tt.Frame()
	if got := a.updateNote; !strings.Contains(got, "检查更新失败：网络不可达") {
		t.Errorf("note %q after a failed check, want the reason", got)
	}
}

// TestStartupCheckRunsOnlyWhenEnabled checks the app reaches the network at
// startup only when the reader asked it to.
func TestStartupCheckRunsOnlyWhenEnabled(t *testing.T) {
	a, _ := updateApp(t, release{Version: "0.2.0", URL: "https://example.com/v0.2.0"}, nil)
	var checks int
	a.checkRelease = func(context.Context) (release, error) {
		checks++
		return release{Version: "0.2.0", URL: "https://example.com/v0.2.0"}, nil
	}

	a.checkUpdatesAtStart()
	if checks != 0 || a.updateNote != "" {
		t.Fatalf("%d checks and note %q before auto update is on, want none", checks, a.updateNote)
	}

	a.autoUpdate = true
	a.checkUpdatesAtStart()
	if checks != 1 {
		t.Errorf("%d checks at startup with auto update on, want one", checks)
	}
	if got := a.updateNote; got != "有新版本 0.2.0" {
		t.Errorf("note %q after the startup check, want the newer version", got)
	}
}

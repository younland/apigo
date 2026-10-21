package app

import (
	"testing"

	"apigo/internal/design"
	"apigo/internal/store"
)

// testApp is an app with a store in a temporary directory, one collection
// with a saved request, and its default environment, the collection open.
func testApp(t *testing.T) *app {
	t.Helper()
	a := newApp()
	a.store = store.New(t.TempDir())
	a.store.Collections = []collection{{
		ID:   "Demo",
		Name: "Demo",
		Root: folder{Name: store.RootFolderName, Requests: []savedRequest{{
			Path: "Pets", Name: "Pets", Kind: kindREST,
			Method: "POST", URL: "https://api.example.com/pets",
		}}},
	}}
	a.store.Environments = []environment{{
		ID: "env-1", Name: "本地", Scope: "Demo", Active: true,
		Vars: []savedRow{{Enabled: true, Key: "BaseURL", Value: "https://api.example.com"}},
	}}
	a.openCollection("Demo")
	// The window opens on the interface the collection holds, as a click on its
	// tree row does: the tests below work on the request panel.
	a.openRequest("Demo", a.store.Collection("Demo").Root.Requests[0])
	return a
}

// TestSettingsRoundTrip checks the appearance, accent and language survive a
// restart.
func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir)
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if s.Settings != (settings{}) {
		t.Fatalf("settings %+v of a new store, want the defaults", s.Settings)
	}
	a := newApp()
	a.store = s
	a.mode, a.accent, a.language = design.Dark, "Blue", "简体中文"
	if err := a.rememberSettings(); err != nil {
		t.Fatal(err)
	}

	loaded := store.New(dir)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	restarted := newApp()
	restarted.store = loaded
	restarted.applySettings()
	if restarted.mode != design.Dark {
		t.Errorf("appearance %v after a restart, want 深色", restarted.mode)
	}
	if restarted.accent != "Blue" {
		t.Errorf("accent %q after a restart, want Blue", restarted.accent)
	}
	if restarted.language != "简体中文" {
		t.Errorf("language %q after a restart, want 简体中文", restarted.language)
	}
	if got := restarted.palette().Accent; got != design.AccentByName("Blue").Base {
		t.Errorf("palette accent %v after a restart, want Blue", got)
	}
}

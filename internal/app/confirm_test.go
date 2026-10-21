package app

import (
	"os"
	"path/filepath"
	"testing"

	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// TestCloseCollectionAsksFirst checks closing a collection asks before the
// workspace goes, and that a cancel leaves it where it is.
func TestCloseCollectionAsksFirst(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("关闭集合 Demo"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("关闭「Demo」的窗口？") {
		t.Fatalf("texts %q, want the question closing asks", tt.Texts())
	}
	if a.active != "Demo" {
		t.Errorf("active %q while the question is asked, want Demo still open", a.active)
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	if a.active != "Demo" {
		t.Errorf("active %q after cancelling, want Demo still open", a.active)
	}

	if err := tt.Click("关闭集合 Demo"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("确认关闭集合"); err != nil {
		t.Fatal(err)
	}
	if a.active != "" {
		t.Errorf("active %q after confirming, want the home window", a.active)
	}
}

// TestDeleteCollectionKeepsItsFolder checks a deleted collection leaves the
// list and keeps its directory, its interfaces and its environments: the next
// run leaves it out by hidden.yaml, not by having lost it.
func TestDeleteCollectionKeepsItsFolder(t *testing.T) {
	s := store.New(t.TempDir())
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddRequest(col.ID, "", "Pets", savedRequest{Kind: kindREST, Method: "GET"}); err != nil {
		t.Fatal(err)
	}

	if err := s.HideCollection(col.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.Collections) != 0 {
		t.Errorf("collections %+v after the deletion, want none listed", s.Collections)
	}
	for _, path := range []string{
		s.CollectionDir(col.ID),
		filepath.Join(s.CollectionDir(col.ID), store.MetaFile),
		s.EnvDir(col.ID),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%q: %v, want what a deletion keeps", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(s.Dir, store.HiddenFile)); err != nil {
		t.Errorf("%q: %v, want the collection remembered as hidden", store.HiddenFile, err)
	}

	reopened := store.New(s.Dir)
	if err := reopened.Load(); err != nil {
		t.Fatal(err)
	}
	if len(reopened.Collections) != 0 {
		t.Errorf("collections %+v after a reload, want the hidden one left out", reopened.Collections)
	}
	if !reopened.IsHidden(col.ID) {
		t.Errorf("hidden %v, want %q remembered", reopened.Collections, col.ID)
	}
}

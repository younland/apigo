package main

import (
	"os"
	"path/filepath"
	"testing"

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
	s := newStore(t.TempDir())
	if err := s.load(); err != nil {
		t.Fatal(err)
	}
	col, err := s.newCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.addRequest(col.ID, "", "Pets", savedRequest{Kind: kindREST, Method: "GET"}); err != nil {
		t.Fatal(err)
	}

	if err := s.hideCollection(col.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.collections) != 0 {
		t.Errorf("collections %+v after the deletion, want none listed", s.collections)
	}
	for _, path := range []string{
		s.collectionDir(col.ID),
		filepath.Join(s.collectionDir(col.ID), metaFile),
		s.envDir(col.ID),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%q: %v, want what a deletion keeps", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(s.dir, hiddenFile)); err != nil {
		t.Errorf("%q: %v, want the collection remembered as hidden", hiddenFile, err)
	}

	reopened := newStore(s.dir)
	if err := reopened.load(); err != nil {
		t.Fatal(err)
	}
	if len(reopened.collections) != 0 {
		t.Errorf("collections %+v after a reload, want the hidden one left out", reopened.collections)
	}
	if !reopened.isHidden(col.ID) {
		t.Errorf("hidden %v, want %q remembered", reopened.hidden, col.ID)
	}
}

package app

import (
	"context"
	"io"
	"testing"
	"time"

	"apigo/internal/store"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// fakeWatch is a directory watch a test drives: it reports the changes the
// test writes down, and ends when its context does or the app closes it.
type fakeWatch struct {
	events chan mygo.FileEvent
	done   chan struct{}

	path string
	opts mygo.FileWatchOptions
}

func newFakeWatch(path string, opts mygo.FileWatchOptions) *fakeWatch {
	return &fakeWatch{
		events: make(chan mygo.FileEvent, 4),
		done:   make(chan struct{}),
		path:   path,
		opts:   opts,
	}
}

// Next waits for the next change of the directory, until the watch ends.
func (f *fakeWatch) Next(ctx context.Context) ([]mygo.FileEvent, error) {
	select {
	case <-ctx.Done():
		return nil, io.EOF
	case <-f.done:
		return nil, io.EOF
	case ev := <-f.events:
		return []mygo.FileEvent{ev}, nil
	}
}

// Close ends the watch.
func (f *fakeWatch) Close() error {
	select {
	case <-f.done:
	default:
		close(f.done)
	}
	return nil
}

// report tells the app what another program did to the directory.
func (f *fakeWatch) report(t *testing.T, op mygo.FileOp, path string) {
	t.Helper()
	select {
	case f.events <- mygo.FileEvent{Op: op, Path: path}:
	case <-time.After(2 * time.Second):
		t.Fatal("the watch took no change")
	}
}

// closed reports whether the app closed the watch.
func (f *fakeWatch) closed() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// watchLog is the watches an app started, oldest first.
type watchLog struct{ all []*fakeWatch }

// start is what the app calls to watch a directory.
func (l *watchLog) start(_ context.Context, path string, opts mygo.FileWatchOptions) (fileEvents, error) {
	w := newFakeWatch(path, opts)
	l.all = append(l.all, w)
	return w, nil
}

// last is the watch the app started most recently.
func (l *watchLog) last(t *testing.T) *fakeWatch {
	t.Helper()
	if len(l.all) == 0 {
		t.Fatal("the app started no watch")
	}
	return l.all[len(l.all)-1]
}

// watchApp is the demo collection open with a watch a test drives, and the
// work the watch asks for queued for the test to run, as the window runs it.
func watchApp(t *testing.T) (*app, *watchLog, <-chan func()) {
	t.Helper()
	a := treeApp(t)
	log := &watchLog{}
	pending := make(chan func(), 4)
	a.watch = log.start
	a.after = func(run func()) { pending <- run }
	return a, log, pending
}

// applyWatch runs the work the watch asked for, which the window would run.
func applyWatch(t *testing.T, pending <-chan func()) {
	t.Helper()
	select {
	case run := <-pending:
		run()
	case <-time.After(2 * time.Second):
		t.Fatal("the watch's change never reached the window")
	}
}

// TestWatchFindsFilesChangedOutside checks the tree follows an interface
// another program writes into the open collection, as the watch reports it,
// rather than a frame that goes looking.
func TestWatchFindsFilesChangedOutside(t *testing.T) {
	a, log, pending := watchApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame() // the first frame starts the watch

	fake := log.last(t)
	if want := a.store.CollectionDir(a.active); fake.path != want {
		t.Errorf("watched %q, want the open collection's directory %q", fake.path, want)
	}
	if !fake.opts.Recursive {
		t.Error("the watch is not recursive, want the folders in the collection watched too")
	}
	if tt.HasText("请求 新接口") {
		t.Fatal("the tree lists an interface that is not there yet")
	}

	// Another program writes an interface beside the ones the app knows, and
	// the watch says so.
	if err := store.WriteYAML(a.store.FolderDir(a.active, "")+"/新接口.yaml", savedRequest{
		Kind: kindREST, Method: "GET",
	}); err != nil {
		t.Fatal(err)
	}
	fake.report(t, mygo.FileCreate, "新接口.yaml")
	applyWatch(t, pending)

	tt.Frame()
	if !tt.HasText("请求 新接口") {
		t.Errorf("texts %q, want the interface another program wrote", tt.Texts())
	}
}

// TestWatchFollowsTheOpenCollection checks the watch moves with the window:
// opening another collection watches that one and stops the first, and going
// home stops watching at all.
func TestWatchFollowsTheOpenCollection(t *testing.T) {
	a, log, _ := watchApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()

	first := log.last(t)
	other, err := a.store.NewCollection("另一个")
	if err != nil {
		t.Fatal(err)
	}
	a.openCollection(other.ID)
	tt.Frame()

	second := log.last(t)
	if second == first {
		t.Fatal("opening another collection kept watching the first one")
	}
	if want := a.store.CollectionDir(other.ID); second.path != want {
		t.Errorf("watched %q, want %q", second.path, want)
	}
	if !first.closed() {
		t.Error("the watch of the collection left behind is still running")
	}

	a.openHome()
	tt.Frame()
	if !second.closed() {
		t.Error("the watch of the collection is still running in the home window")
	}
}

// TestWatchSkipsCollectionDirs checks the watch of a collection is not woken by
// its environments, its uploads, or the files a write waits in: none of them
// are folders of its tree.
func TestWatchSkipsCollectionDirs(t *testing.T) {
	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{".env", true, true},
		{".tmp", true, true},
		{"static", true, true},
		{"分组 1/static", true, true},
		{"分组 1", true, false},
		{"分组 1/子分组", true, false},
		{"接口.yaml", false, false},
		{"分组 1/接口.yaml", false, false},
	}
	for _, tc := range cases {
		if got := reservedInCollection(tc.path, tc.isDir); got != tc.want {
			t.Errorf("reservedInCollection(%q, %v) = %v, want %v", tc.path, tc.isDir, got, tc.want)
		}
	}
}

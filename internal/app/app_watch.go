package app

import (
	"context"
	"os"
	"time"

	"apigo/internal/store"

	"github.com/egoist/mygo"
)

// fileEvents is a watch of a directory: the changes another program makes in
// it, which the tree follows.
type fileEvents interface {
	// Next waits for the next batch of changes, and ends with an error once
	// the watch stops.
	Next(ctx context.Context) ([]mygo.FileEvent, error)
	// Close stops the watch.
	Close() error
}

// watchStarter starts a watch of a directory. It is mygo's watch in the app,
// and one a test drives in the tests.
type watchStarter func(ctx context.Context, path string, opts mygo.FileWatchOptions) (fileEvents, error)

// watchFiles is mygo's watch of a directory, as the app asks for one.
func watchFiles(ctx context.Context, path string, opts mygo.FileWatchOptions) (fileEvents, error) {
	return mygo.WatchFiles(ctx, path, opts)
}

// watchDebounce is how long a watch waits for changes to stop before it
// reports them: a save writes a file and then moves it into place, which is
// one change of the collection, not two.
const watchDebounce = 150 * time.Millisecond

// followFiles starts the watch of the open collection's directory, and stops
// the watch of the one the window left: the tree follows what another program
// does to the files, without a frame that goes looking. It also watches the
// directory the collections live in, so the list knows which of them another
// program deleted.
func (a *app) followFiles() {
	a.followList()
	a.followWindow()
}

// followWindow keeps the watch on the collection the window shows, and drops
// the one of a collection it left.
func (a *app) followWindow() {
	if a.active == a.watched {
		return
	}
	a.stopWatch()
	if a.active == "" || a.watch == nil {
		return
	}
	a.startWatch(a.active)
}

// followList watches the directory the collections are kept in, so one another
// program deletes is marked on the list without a frame that goes looking. It
// is tried once: a directory the platform cannot watch is not tried again every
// frame.
func (a *app) followList() {
	if a.listWatched || a.watch == nil {
		return
	}
	a.listWatched = true
	dir := a.store.CollectionsDir()
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return // nothing on disk to watch: the list keeps what it holds
	}
	ctx, cancel := context.WithCancel(context.Background())
	events, err := a.watch(ctx, dir, mygo.FileWatchOptions{Debounce: watchDebounce})
	if err != nil {
		cancel()
		return
	}
	a.listWatchStop = func() {
		cancel()
		_ = events.Close()
	}
	go a.drainList(ctx, events)
}

// drainList marks the collections that lost their directory, on the window's
// thread: the list is built there.
func (a *app) drainList(ctx context.Context, events fileEvents) {
	for {
		if _, err := events.Next(ctx); err != nil {
			return // the watch ended: the app quits
		}
		run := func() { a.refreshMissing() }
		if a.after == nil {
			run()
			continue
		}
		a.after(run)
	}
}

// refreshMissing records which collections have lost their directory. The list
// greys them out, and asks to take them off rather than opening what is not
// there.
func (a *app) refreshMissing() {
	missing := map[string]bool{}
	for _, col := range a.store.Collections {
		if info, err := os.Stat(a.store.CollectionDir(col.ID)); err != nil || !info.IsDir() {
			missing[col.ID] = true
		}
	}
	a.missing = missing
}

// collectionMissing reports whether a collection's directory is gone.
func (a *app) collectionMissing(id string) bool { return a.missing[id] }

// startWatch watches a collection's directory.
func (a *app) startWatch(id string) {
	// Tried once per collection: what the platform cannot watch, and a
	// collection that is held without a directory, are not tried again every
	// frame, only when the collection is opened again.
	a.watched = id
	dir := a.store.CollectionDir(id)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return // nothing on disk to watch: what is held stays
	}
	ctx, cancel := context.WithCancel(context.Background())
	events, err := a.watch(ctx, dir, mygo.FileWatchOptions{
		Recursive: true,
		Debounce:  watchDebounce,
		Ignore:    reservedInCollection,
	})
	if err != nil {
		cancel()
		a.saveErr = "监听集合目录失败：" + err.Error()
		return
	}
	a.watchStop = func() {
		cancel()
		_ = events.Close()
	}
	go a.drainWatch(ctx, id, events)
}

// drainWatch reads the changes a watch reports, and reads the collection again
// on the window's thread: the tree is built there, and a change of another
// program's must not build it from this goroutine.
func (a *app) drainWatch(ctx context.Context, id string, events fileEvents) {
	for {
		if _, err := events.Next(ctx); err != nil {
			return // the watch ended: the collection was closed, or the app quits
		}
		run := func() {
			if a.active == id {
				a.reloadActive()
			}
		}
		if a.after == nil {
			run()
			continue
		}
		a.after(run)
	}
}

// stopWatch ends the watch of the collection the window left, if any.
func (a *app) stopWatch() {
	if a.watchStop != nil {
		a.watchStop()
	}
	a.watchStop, a.watched = nil, ""
}

// reservedInCollection reports whether a path of a collection belongs to the
// collection itself rather than to its tree: its environments, its uploads,
// and the files a write waits in.
func reservedInCollection(path string, isDir bool) bool {
	if !isDir {
		return false
	}
	for _, name := range splitPath(path) {
		if store.IsReservedDir(name) {
			return true
		}
	}
	return false
}

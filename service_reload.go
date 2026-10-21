package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// reloadCollection reads a collection again from its directory: its tree of
// interfaces and the environments of its .env, replacing what was held for it.
// Opening a collection does this, and so does a change another program made.
func (s *store) reloadCollection(collectionID string) error {
	if s.collectionIndex(collectionID) < 0 {
		return fmt.Errorf("没有集合 %q", collectionID)
	}
	if info, err := os.Stat(s.collectionDir(collectionID)); err != nil || !info.IsDir() {
		return nil // nothing on disk to read: what is held stays
	}
	kept := s.environments[:0]
	for _, env := range s.environments {
		if env.Scope != collectionID {
			kept = append(kept, env)
		}
	}
	s.environments = kept
	col, err := s.readCollection(collectionID)
	if err != nil {
		return err
	}
	s.collections[s.collectionIndex(collectionID)] = col
	s.sortCollections()
	return nil
}

// treeSignature is what a collection's files say: their names, sizes and
// modification times. Comparing it finds a change another program made, which
// a folder watcher would report where the platform has one.
func (s *store) treeSignature(collectionID string) string {
	var b strings.Builder
	root := s.collectionDir(collectionID)
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && isReservedDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		fmt.Fprintf(&b, "%s|%d|%d\n", filepath.ToSlash(rel), info.Size(), info.ModTime().UnixNano())
		return nil
	})
	return b.String()
}

// treeHasRequest reports whether an interface is still in the tree.
func treeHasRequest(f folder, treePath string) bool {
	for _, req := range f.Requests {
		if req.Path == treePath {
			return true
		}
	}
	for _, sub := range f.Folders {
		if treeHasRequest(sub, treePath) {
			return true
		}
	}
	return false
}

// removeTempFiles deletes what a crashed write left in the collection's .tmp
// directory: a write moves its file out of there, so whatever stays is stale.
func (s *store) removeTempFiles(collectionID string) {
	dir := filepath.Join(s.collectionDir(collectionID), tmpDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}

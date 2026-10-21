package main

import (
	"fmt"
	"os"
	"path/filepath"
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

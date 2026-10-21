package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// reloadCollection reads a collection again from its directory: its tree of
// interfaces and the environments of its .env, replacing what was held for it.
// Opening a collection does this, and so does a change another program made.
func (s *Store) ReloadCollection(collectionID string) error {
	if s.CollectionIndex(collectionID) < 0 {
		return fmt.Errorf("没有集合 %q", collectionID)
	}
	if info, err := os.Stat(s.CollectionDir(collectionID)); err != nil || !info.IsDir() {
		return nil // nothing on disk to read: what is held stays
	}
	kept := s.Environments[:0]
	for _, env := range s.Environments {
		if env.Scope != collectionID {
			kept = append(kept, env)
		}
	}
	s.Environments = kept
	col, err := s.readCollection(collectionID)
	if err != nil {
		return err
	}
	s.Collections[s.CollectionIndex(collectionID)] = col
	s.sortCollections()
	return nil
}

// treeHasRequest reports whether an interface is still in the tree.
func TreeHasRequest(f Folder, treePath string) bool {
	for _, req := range f.Requests {
		if req.Path == treePath {
			return true
		}
	}
	for _, sub := range f.Folders {
		if TreeHasRequest(sub, treePath) {
			return true
		}
	}
	return false
}

// removeTempFiles deletes what a crashed write left in the collection's .tmp
// directory: a write moves its file out of there, so whatever stays is stale.
func (s *Store) RemoveTempFiles(collectionID string) {
	dir := filepath.Join(s.CollectionDir(collectionID), TmpDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}

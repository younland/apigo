package store

import (
	"fmt"
	"os"
	"strings"
)

// defaultCollectionName names a collection that was saved without one.
const defaultCollectionName = "集合"

// newCollection creates a collection directory with its default
// environments, each carrying an empty BaseURL variable.
func (s *Store) NewCollection(name string) (Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultCollectionName
	}
	col := Collection{ID: s.uniqueCollectionID(name, ""), Name: name}
	col.Root = Folder{Name: RootFolderName}
	if err := s.saveCollection(col); err != nil {
		return Collection{}, err
	}
	return col, s.addDefaultEnvironments(col.ID)
}

// uniqueCollectionID turns a collection name into the directory its files
// live in, numbered when the name is taken. The directory of except is left
// out of the check, so renaming a collection to its own name keeps it.
func (s *Store) uniqueCollectionID(name, except string) string {
	base := safeFileName(name)
	if base == "" {
		base = "collection"
	}
	id := base
	for n := 2; s.collectionIDTaken(id, except); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

// collectionIDTaken reports whether a directory name is already in use,
// leaving except out: a collection keeps its own directory.
func (s *Store) collectionIDTaken(id, except string) bool {
	if id == except {
		return false
	}
	if s.CollectionIndex(id) >= 0 {
		return true
	}
	_, err := os.Stat(s.CollectionDir(id))
	return err == nil
}

// renameCollection gives a collection a new name and the directory that goes
// with it, keeping its environments and interfaces with it.
func (s *Store) RenameCollection(id, name string) (Collection, error) {
	i := s.CollectionIndex(id)
	if i < 0 {
		return Collection{}, fmt.Errorf("没有集合 %q", id)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultCollectionName
	}
	col := s.Collections[i]
	newID := s.uniqueCollectionID(name, id)
	if newID != id {
		if err := renameDir(s.CollectionDir(id), s.CollectionDir(newID)); err != nil {
			return Collection{}, err
		}
	}
	col.ID, col.Name = newID, name
	// The collection takes its new id here, before saveCollection looks it up.
	s.Collections[i] = col
	if err := s.saveCollection(col); err != nil {
		return Collection{}, err
	}
	s.sortCollections()
	for j := range s.Environments {
		if s.Environments[j].Scope == id {
			s.Environments[j].Scope = newID
		}
	}
	return col, s.SaveEnvironments()
}

// renameDir moves a collection's directory, treating one that is not there
// as nothing to move.
func renameDir(oldDir, newDir string) error {
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return nil
	}
	return os.Rename(oldDir, newDir)
}

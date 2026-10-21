package main

import (
	"fmt"
	"os"
	"strings"
)

// defaultCollectionName names a collection that was saved without one.
const defaultCollectionName = "集合"

// newCollection creates a collection directory with its default
// environments, each carrying an empty BaseURL variable.
func (s *store) newCollection(name string) (collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultCollectionName
	}
	col := collection{ID: s.uniqueCollectionID(name, ""), Name: name}
	col.Root = folder{Name: rootFolderName}
	if err := s.saveCollection(col); err != nil {
		return collection{}, err
	}
	return col, s.addDefaultEnvironments(col.ID)
}

// uniqueCollectionID turns a collection name into the directory its files
// live in, numbered when the name is taken. The directory of except is left
// out of the check, so renaming a collection to its own name keeps it.
func (s *store) uniqueCollectionID(name, except string) string {
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
func (s *store) collectionIDTaken(id, except string) bool {
	if id == except {
		return false
	}
	if s.collectionIndex(id) >= 0 {
		return true
	}
	_, err := os.Stat(s.collectionDir(id))
	return err == nil
}

// renameCollection gives a collection a new name and the directory that goes
// with it, keeping its environments and interfaces with it.
func (s *store) renameCollection(id, name string) (collection, error) {
	i := s.collectionIndex(id)
	if i < 0 {
		return collection{}, fmt.Errorf("没有集合 %q", id)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultCollectionName
	}
	col := s.collections[i]
	newID := s.uniqueCollectionID(name, id)
	if newID != id {
		if err := renameDir(s.collectionDir(id), s.collectionDir(newID)); err != nil {
			return collection{}, err
		}
	}
	col.ID, col.Name = newID, name
	// The collection takes its new id here, before saveCollection looks it up.
	s.collections[i] = col
	if err := s.saveCollection(col); err != nil {
		return collection{}, err
	}
	s.sortCollections()
	for j := range s.environments {
		if s.environments[j].Scope == id {
			s.environments[j].Scope = newID
		}
	}
	return col, s.saveEnvironments()
}

// renameDir moves a collection's directory, treating one that is not there
// as nothing to move.
func renameDir(oldDir, newDir string) error {
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return nil
	}
	return os.Rename(oldDir, newDir)
}

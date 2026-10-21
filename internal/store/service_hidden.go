package store

import "path/filepath"

// hiddenFile lists the collections the app leaves out of the window: deleting
// one takes it off the list and leaves its directory where it is, so nothing
// it holds is lost.
const HiddenFile = "hidden.yaml"

// hiddenList is hidden.yaml: the ids of the collections taken off the list.
type hiddenList struct {
	Collections []string `yaml:"collections,omitempty"`
}

// loadHidden reads the collections taken off the list.
func (s *Store) loadHidden() error {
	var h hiddenList
	if err := readYAMLIfExists(filepath.Join(s.Dir, HiddenFile), &h); err != nil {
		return err
	}
	s.hidden = h.Collections
	return nil
}

// isHidden reports whether a collection is off the list.
func (s *Store) IsHidden(id string) bool {
	for _, hidden := range s.hidden {
		if hidden == id {
			return true
		}
	}
	return false
}

// hideCollection takes a collection off the list, keeping its directory, its
// interfaces and its environments where they are: hidden.yaml says which ones
// the next run leaves out again.
func (s *Store) HideCollection(id string) error {
	if i := s.CollectionIndex(id); i >= 0 {
		s.Collections = append(s.Collections[:i], s.Collections[i+1:]...)
	}
	kept := s.Environments[:0]
	for _, env := range s.Environments {
		if env.Scope != id {
			kept = append(kept, env)
		}
	}
	s.Environments = kept
	if !s.IsHidden(id) {
		s.hidden = append(s.hidden, id)
	}
	if err := WriteYAML(filepath.Join(s.Dir, HiddenFile), hiddenList{Collections: s.hidden}); err != nil {
		return err
	}
	return s.SaveEnvironments()
}

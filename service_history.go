package main

import (
	"os"
	"path/filepath"
	"time"
)

// historyEntry is one request that was sent.
type historyEntry struct {
	Method  string    `yaml:"method"`
	URL     string    `yaml:"url"`
	Status  int       `yaml:"status"`
	At      time.Time `yaml:"at"`
	Title   string    `yaml:"title,omitempty"`
	Kind    kind      `yaml:"kind,omitempty"`
	SavedID string    `yaml:"request,omitempty"`
}

// historyLimit caps the entries kept on disk.
const historyLimit = 200

// saveHistory writes the history file, keeping its newest entries.
func (s *store) saveHistory() error {
	if len(s.history) > historyLimit {
		s.history = s.history[:historyLimit]
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return writeYAML(filepath.Join(s.dir, "history.yaml"), s.history)
}

// addHistory puts a sent request first in the history and writes it.
func (s *store) addHistory(entry historyEntry) error {
	s.history = append([]historyEntry{entry}, s.history...)
	return s.saveHistory()
}

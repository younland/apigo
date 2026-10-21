package store

import (
	"os"
	"path/filepath"
	"time"
)

// historyEntry is one request that was sent.
type HistoryEntry struct {
	Method  string    `yaml:"method"`
	URL     string    `yaml:"url"`
	Status  int       `yaml:"status"`
	At      time.Time `yaml:"at"`
	Title   string    `yaml:"title,omitempty"`
	Kind    Kind      `yaml:"kind,omitempty"`
	SavedID string    `yaml:"request,omitempty"`
}

// historyLimit caps the entries kept on disk.
const historyLimit = 200

// saveHistory writes the history file, keeping its newest entries.
func (s *Store) saveHistory() error {
	if len(s.History) > historyLimit {
		s.History = s.History[:historyLimit]
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	return WriteYAML(filepath.Join(s.Dir, "history.yaml"), s.History)
}

// addHistory puts a sent request first in the history and writes it.
func (s *Store) AddHistory(entry HistoryEntry) error {
	s.History = append([]HistoryEntry{entry}, s.History...)
	return s.saveHistory()
}

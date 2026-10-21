package store

import (
	"os"
	"path/filepath"
)

// settings is what the app remembers between runs: the appearance, the
// accent and the language, kept in one file of the data directory.
type Settings struct {
	Mode     string `yaml:"appearance,omitempty"`
	Accent   string `yaml:"accent,omitempty"`
	Language string `yaml:"language,omitempty"`
}

// settingsPath is the file the settings are kept in.
func (s *Store) settingsPath() string { return filepath.Join(s.Dir, "settings.yaml") }

// loadSettings reads what the last run kept, treating a missing file as the
// defaults.
func (s *Store) loadSettings() error {
	return readYAMLIfExists(s.settingsPath(), &s.Settings)
}

// saveSettings writes what the app should remember.
func (s *Store) SaveSettings() error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	return WriteYAML(s.settingsPath(), s.Settings)
}

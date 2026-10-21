package main

import (
	"os"
	"path/filepath"
)

// settings is what the app remembers between runs: the appearance, the
// accent and the language, kept in one file of the data directory.
type settings struct {
	Mode     string `yaml:"appearance,omitempty"`
	Accent   string `yaml:"accent,omitempty"`
	Language string `yaml:"language,omitempty"`
}

// settingsPath is the file the settings are kept in.
func (s *store) settingsPath() string { return filepath.Join(s.dir, "settings.yaml") }

// loadSettings reads what the last run kept, treating a missing file as the
// defaults.
func (s *store) loadSettings() error {
	return readYAMLIfExists(s.settingsPath(), &s.settings)
}

// saveSettings writes what the app should remember.
func (s *store) saveSettings() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return writeYAML(s.settingsPath(), s.settings)
}

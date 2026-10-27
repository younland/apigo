package store

import (
	"os"
	"path/filepath"
)

// Settings is what the app remembers between runs: the appearance, the accent,
// the language, how much of a response it reads, how long it waits for one, and
// whether it looks for a newer version at startup, kept in one file of the data
// directory.
type Settings struct {
	Mode     string `yaml:"appearance,omitempty"`
	Accent   string `yaml:"accent,omitempty"`
	Language string `yaml:"language,omitempty"`
	// MaxResponseMB caps a response body, in megabytes: zero keeps the
	// default.
	MaxResponseMB int `yaml:"maxResponseMB,omitempty"`
	// TimeoutSeconds is how long a request is waited for, in seconds: zero
	// keeps the default.
	TimeoutSeconds int `yaml:"timeoutSeconds,omitempty"`
	// AutoUpdate makes the app look for a newer version when it starts. It is
	// off unless the reader turns it on: a check reaches the network.
	AutoUpdate bool `yaml:"autoUpdate,omitempty"`
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

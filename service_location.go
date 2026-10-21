package main

import (
	"errors"
	"os"
	"path/filepath"
)

// location is the pointer to the data directory the user picked, kept in the
// default data directory so the choice outlives it.
type location struct {
	DataDir string `yaml:"dataDir,omitempty"`
}

// locationPath is the file the chosen data directory is kept in.
func locationPath(baseDir string) string { return filepath.Join(baseDir, "location.yaml") }

// storeDirAt returns where the collections live: the directory the user
// picked, or baseDir when they never did.
func storeDirAt(baseDir string) string {
	var l location
	if err := readYAMLIfExists(locationPath(baseDir), &l); err != nil || l.DataDir == "" {
		return baseDir
	}
	return l.DataDir
}

// errNoBaseDir is what saving the choice reports where the app has no default
// data directory to keep it in.
var errNoBaseDir = errors.New("没有默认的数据目录")

// rememberStoreDir keeps the chosen data directory for the next run.
func rememberStoreDir(baseDir, dataDir string) error {
	if baseDir == "" {
		return errNoBaseDir
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}
	return writeYAML(locationPath(baseDir), location{DataDir: dataDir})
}

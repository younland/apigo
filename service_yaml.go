package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func readYAML(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}

func readYAMLIfExists(path string, v any) error {
	err := readYAML(path, v)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// writeYAML writes a file whole, replacing it in one rename so a crash
// cannot leave half a file behind.
func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// isYAMLFile reports whether a name is one of the files we read.
func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

// safeFileName keeps a name usable as a file name on every platform, so an
// interface, folder or environment name can name its file.
func safeFileName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		if r < 0x20 {
			continue
		}
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Trim(strings.TrimSpace(b.String()), ".")
}

// nextID returns an id for a new item: the prefix and the first number not
// taken by the other ids.
func nextID(prefix string, taken []string) string {
	used := make(map[string]bool, len(taken))
	for _, id := range taken {
		used[id] = true
	}
	for n := 1; ; n++ {
		id := fmt.Sprintf("%s-%d", prefix, n)
		if !used[id] {
			return id
		}
	}
}

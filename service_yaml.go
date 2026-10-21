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

// tmpDirName is the directory that holds files being written: a write lands
// there, then moves over the file it replaces, so a crash cannot leave half a
// file behind.
const tmpDirName = ".tmp"

// writeYAML writes a file whole, replacing it in one rename so a crash
// cannot leave half a file behind. The file being written waits in the .tmp
// directory of the collection it belongs to, or of its own directory when it
// belongs to none.
func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmpDir := filepath.Join(tmpRoot(dir), tmpDirName)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(tmpDir, filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// tmpRoot is where the temporary files of a directory's writes wait: the
// collection the directory belongs to, the one holding apigo.yaml, or the
// directory itself when it belongs to none.
func tmpRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, metaFile)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
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

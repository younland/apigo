package main

// Where the data lives, with one directory per collection:
//
//	<数据目录>/collections/<集合>/apigo.yaml          集合自己的说明与设置
//	<数据目录>/collections/<集合>/.env/<环境>.yaml     集合的环境与其变量
//	<数据目录>/collections/<集合>/static/              上传等接口要用的临时文件
//	<数据目录>/collections/<集合>/<接口>.yaml          根目录下的接口
//	<数据目录>/collections/<集合>/<文件夹>/<接口>.yaml  子文件夹里的接口
//
// 界面里的「根目录」就是集合自己的目录，它没有对应的文件；子文件夹与接口
// 一个对一个文件和目录，所以文件改名即接口改名。

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/egoist/mygo"
)

// savedRow is a key/value row as YAML stores it.
type savedRow struct {
	Enabled bool   `yaml:"enabled"`
	Key     string `yaml:"key,omitempty"`
	Value   string `yaml:"value,omitempty"`
}

// savedRequest is one interface file, named after the interface itself.
type savedRequest struct {
	Path     string     `yaml:"-"` // the interface's place in the tree
	Name     string     `yaml:"-"` // the file name without its extension
	Kind     kind       `yaml:"kind"`
	Method   string     `yaml:"method,omitempty"`
	URL      string     `yaml:"url,omitempty"`
	Params   []savedRow `yaml:"params,omitempty"`
	Headers  []savedRow `yaml:"headers,omitempty"`
	BodyType string     `yaml:"bodyType,omitempty"`
	Body     string     `yaml:"body,omitempty"`
}

// folder is one directory of a collection's tree: a folder of the tree, or
// its 根目录, whose Path is empty.
type folder struct {
	Path     string
	Name     string
	Folders  []folder
	Requests []savedRequest
}

// collection is one directory of the data directory: the interfaces of its
// tree, its own environments, and what apigo.yaml says about it.
type collection struct {
	ID   string // the directory under collections/
	Name string
	Desc string
	Root folder
}

// collectionMeta is apigo.yaml: what the collection says about itself,
// rather than about one of its interfaces.
type collectionMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// metaFile is the reserved file of a collection's own settings.
const metaFile = "apigo.yaml"

// rootFolderName is how the collection's own directory shows in the tree.
const rootFolderName = "根目录"

// store keeps one directory per collection, plus the environments that
// apply to every collection and the history, under dir.
type store struct {
	dir          string
	collections  []collection
	environments []environment
	history      []historyEntry
}

func newStore(dir string) *store { return &store{dir: dir} }

// defaultStoreDir is where the app keeps its data: MyGo's user data
// directory, or the user's configuration directory when it has none.
func defaultStoreDir() string {
	if dir, err := mygo.App.Path(mygo.PathUserData); err == nil && dir != "" {
		return dir
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ".apigo"
	}
	return filepath.Join(dir, "apigo")
}

func (s *store) collectionsDir() string { return filepath.Join(s.dir, "collections") }

// collectionDir is the directory a collection is kept in.
func (s *store) collectionDir(id string) string {
	return filepath.Join(s.collectionsDir(), id)
}

// staticDir is where a collection keeps the files its interfaces upload.
func (s *store) staticDir(id string) string {
	return filepath.Join(s.collectionDir(id), "static")
}

// load reads every collection directory, the global environments and the
// history, treating missing ones as empty.
func (s *store) load() error {
	entries, err := os.ReadDir(s.collectionsDir())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	s.collections, s.environments = nil, nil
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		col, err := s.readCollection(entry.Name())
		if err != nil {
			return err
		}
		s.collections = append(s.collections, col)
	}
	s.sortCollections()
	if err := s.loadGlobalEnvironments(); err != nil {
		return err
	}
	return readYAMLIfExists(filepath.Join(s.dir, "history.yaml"), &s.history)
}

// readCollection reads one collection directory: apigo.yaml, the tree of
// interface files and the environments of its .env directory.
func (s *store) readCollection(id string) (collection, error) {
	var meta collectionMeta
	dir := s.collectionDir(id)
	if err := readYAMLIfExists(filepath.Join(dir, metaFile), &meta); err != nil {
		return collection{}, err
	}
	col := collection{ID: id, Name: meta.Name, Desc: meta.Description}
	if col.Name == "" {
		col.Name = id
	}
	root, err := s.readFolder(id, "")
	if err != nil {
		return collection{}, err
	}
	col.Root = root
	envs, err := s.readEnvironments(id)
	if err != nil {
		return collection{}, err
	}
	s.environments = append(s.environments, envs...)
	return col, nil
}

// saveCollection writes a collection's apigo.yaml and its tree, and keeps
// the result in memory.
func (s *store) saveCollection(c collection) error {
	if c.ID == "" {
		return fmt.Errorf("collection without an id")
	}
	dir := s.collectionDir(c.ID)
	for _, sub := range []string{dir, filepath.Join(dir, envDirName), s.staticDir(c.ID)} {
		if err := os.MkdirAll(sub, 0o755); err != nil {
			return err
		}
	}
	meta := collectionMeta{Name: c.Name, Description: c.Desc}
	if err := writeYAML(filepath.Join(dir, metaFile), meta); err != nil {
		return err
	}
	if err := writeFolder(dir, c.Root); err != nil {
		return err
	}
	if i := s.collectionIndex(c.ID); i >= 0 {
		s.collections[i] = c
		return nil
	}
	s.collections = append(s.collections, c)
	s.sortCollections()
	return nil
}

func (s *store) sortCollections() {
	sort.Slice(s.collections, func(i, j int) bool { return s.collections[i].Name < s.collections[j].Name })
}

// collectionIndex returns where a collection is kept, or -1.
func (s *store) collectionIndex(id string) int {
	for i, c := range s.collections {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// collection returns the collection with the given id, nil when it is gone.
func (s *store) collection(id string) *collection {
	if i := s.collectionIndex(id); i >= 0 {
		return &s.collections[i]
	}
	return nil
}

// defaultCollectionName names a collection that was saved without one.
const defaultCollectionName = "集合"

// newCollection creates a collection directory with its default
// environments, each carrying an empty BaseURL variable.
func (s *store) newCollection(name string) (collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultCollectionName
	}
	col := collection{ID: s.uniqueCollectionID(name), Name: name}
	col.Root = folder{Name: rootFolderName}
	if err := s.saveCollection(col); err != nil {
		return collection{}, err
	}
	return col, s.addDefaultEnvironments(col.ID)
}

// uniqueCollectionID turns a collection name into the directory its files
// live in, numbered when the name is taken.
func (s *store) uniqueCollectionID(name string) string {
	base := safeFileName(name)
	if base == "" {
		base = "collection"
	}
	id := base
	for n := 2; s.collectionIDTaken(id); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

// collectionIDTaken reports whether a directory name is already in use.
func (s *store) collectionIDTaken(id string) bool {
	if s.collectionIndex(id) >= 0 {
		return true
	}
	_, err := os.Stat(s.collectionDir(id))
	return err == nil
}

// deleteCollection removes a collection, its environments and its
// directory.
func (s *store) deleteCollection(id string) error {
	if i := s.collectionIndex(id); i >= 0 {
		s.collections = append(s.collections[:i], s.collections[i+1:]...)
	}
	kept := s.environments[:0]
	for _, env := range s.environments {
		if env.Scope != id {
			kept = append(kept, env)
		}
	}
	s.environments = kept
	if err := s.saveEnvironments(); err != nil {
		return err
	}
	return os.RemoveAll(s.collectionDir(id))
}

// dirText is the directory the collections and environments are kept in.
func (s *store) dirText() string {
	if s.dir != "" {
		return s.dir
	}
	return defaultStoreDir()
}

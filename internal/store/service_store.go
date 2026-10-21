package store

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

	"github.com/egoist/mygo"
)

// savedRow is a key/value row as YAML stores it.
type Row struct {
	Enabled bool   `yaml:"enabled"`
	Key     string `yaml:"key,omitempty"`
	Value   string `yaml:"value,omitempty"`
	// File is the path of an uploaded file relative to the collection, and
	// FileName the name it had when it was chosen.
	File     string `yaml:"file,omitempty"`
	FileName string `yaml:"fileName,omitempty"`
}

// savedRequest is one interface file, named after the interface itself.
type Request struct {
	Path     string `yaml:"-"` // the interface's place in the tree
	Name     string `yaml:"-"` // the file name without its extension
	Kind     Kind   `yaml:"kind"`
	Method   string `yaml:"method,omitempty"`
	URL      string `yaml:"url,omitempty"`
	Params   []Row  `yaml:"params,omitempty"`
	Headers  []Row  `yaml:"headers,omitempty"`
	BodyType string `yaml:"bodyType,omitempty"`
	Body     string `yaml:"body,omitempty"`
	// Form holds the rows of form-data and x-www-form-urlencoded bodies.
	Form []Row `yaml:"form,omitempty"`
}

// folder is one directory of a collection's tree: a folder of the tree, or
// its 根目录, whose Path is empty.
type Folder struct {
	Path     string
	Name     string
	Folders  []Folder
	Requests []Request
}

// collection is one directory of the data directory: the interfaces of its
// tree, its own environments, and what apigo.yaml says about it.
type Collection struct {
	ID   string // the directory under collections/
	Name string
	Desc string
	Root Folder
}

// collectionMeta is apigo.yaml: what the collection says about itself,
// rather than about one of its interfaces.
type collectionMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// metaFile is the reserved file of a collection's own settings.
const MetaFile = "apigo.yaml"

// rootFolderName is how the collection's own directory shows in the tree.
const RootFolderName = "根目录"

// Store keeps one directory per collection, plus the environments that apply
// to every collection and the history, under Dir.
type Store struct {
	Dir          string
	Collections  []Collection
	Environments []Environment
	History      []HistoryEntry
	Settings     Settings
	// hidden are the collections taken off the list, whose directories stay.
	hidden []string
}

func New(dir string) *Store { return &Store{Dir: dir} }

// defaultStoreDir is where the app keeps its data: MyGo's user data
// directory, or the user's configuration directory when it has none.
func DefaultDir() string {
	if dir, err := mygo.App.Path(mygo.PathUserData); err == nil && dir != "" {
		return dir
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ".apigo"
	}
	return filepath.Join(dir, "apigo")
}

func (s *Store) collectionsDir() string { return filepath.Join(s.Dir, "collections") }

// collectionDir is the directory a collection is kept in.
func (s *Store) CollectionDir(id string) string {
	return filepath.Join(s.collectionsDir(), id)
}

// staticDir is where a collection keeps the files its interfaces upload.
func (s *Store) staticDir(id string) string {
	return filepath.Join(s.CollectionDir(id), "static")
}

// load reads every collection directory, the global environments and the
// history, treating missing ones as empty. The collections taken off the list
// stay on disk and are left out here.
func (s *Store) Load() error {
	entries, err := os.ReadDir(s.collectionsDir())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := s.loadHidden(); err != nil {
		return err
	}
	s.Collections, s.Environments = nil, nil
	for _, entry := range entries {
		if !entry.IsDir() || s.IsHidden(entry.Name()) {
			continue
		}
		col, err := s.readCollection(entry.Name())
		if err != nil {
			return err
		}
		s.Collections = append(s.Collections, col)
	}
	s.sortCollections()
	if err := s.loadGlobalEnvironments(); err != nil {
		return err
	}
	if err := s.loadSettings(); err != nil {
		return err
	}
	return readYAMLIfExists(filepath.Join(s.Dir, "history.yaml"), &s.History)
}

// readCollection reads one collection directory: apigo.yaml, the tree of
// interface files and the environments of its .env directory.
func (s *Store) readCollection(id string) (Collection, error) {
	var meta collectionMeta
	dir := s.CollectionDir(id)
	if err := readYAMLIfExists(filepath.Join(dir, MetaFile), &meta); err != nil {
		return Collection{}, err
	}
	col := Collection{ID: id, Name: meta.Name, Desc: meta.Description}
	if col.Name == "" {
		col.Name = id
	}
	root, err := s.readFolder(id, "")
	if err != nil {
		return Collection{}, err
	}
	col.Root = root
	envs, err := s.readEnvironments(id)
	if err != nil {
		return Collection{}, err
	}
	s.Environments = append(s.Environments, envs...)
	return col, nil
}

// saveCollection writes a collection's apigo.yaml and its tree, and keeps
// the result in memory.
func (s *Store) saveCollection(c Collection) error {
	if c.ID == "" {
		return fmt.Errorf("collection without an id")
	}
	dir := s.CollectionDir(c.ID)
	for _, sub := range []string{dir, filepath.Join(dir, envDirName), s.staticDir(c.ID)} {
		if err := os.MkdirAll(sub, 0o755); err != nil {
			return err
		}
	}
	meta := collectionMeta{Name: c.Name, Description: c.Desc}
	if err := WriteYAML(filepath.Join(dir, MetaFile), meta); err != nil {
		return err
	}
	if err := writeFolder(dir, c.Root); err != nil {
		return err
	}
	if i := s.CollectionIndex(c.ID); i >= 0 {
		s.Collections[i] = c
		return nil
	}
	s.Collections = append(s.Collections, c)
	s.sortCollections()
	return nil
}

func (s *Store) sortCollections() {
	sort.Slice(s.Collections, func(i, j int) bool { return s.Collections[i].Name < s.Collections[j].Name })
}

// collectionIndex returns where a collection is kept, or -1.
func (s *Store) CollectionIndex(id string) int {
	for i, c := range s.Collections {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// collection returns the collection with the given id, nil when it is gone.
func (s *Store) Collection(id string) *Collection {
	if i := s.CollectionIndex(id); i >= 0 {
		return &s.Collections[i]
	}
	return nil
}

// dirText is the directory the collections and environments are kept in.
func (s *Store) DirText() string {
	if s.Dir != "" {
		return s.Dir
	}
	return DefaultDir()
}

package store

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// envDirName is the directory of a collection's environments, and
// reservedDirs are the directories of a collection that are not folders of
// its tree.
const envDirName = ".env"

var reservedDirs = []string{envDirName, "static"}

// folderDir is the directory a tree path stands for.
func (s *Store) FolderDir(collectionID, folderPath string) string {
	return filepath.Join(s.CollectionDir(collectionID), filepath.FromSlash(folderPath))
}

// joinPath puts a name under a tree path, at the top of the tree when the
// path is empty.
func joinPath(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "/" + name
}

// pathName is the name a tree path shows: the collection's own directory
// shows as 根目录.
func PathName(treePath string) string {
	if treePath == "" {
		return RootFolderName
	}
	return path.Base(treePath)
}

// readFolder reads one directory of a collection into the tree: folders of
// their own become folders, *.yaml files become interfaces, and apigo.yaml
// of the collection's own directory stays its settings.
func (s *Store) readFolder(collectionID, folderPath string) (Folder, error) {
	dir := s.FolderDir(collectionID, folderPath)
	f := Folder{Path: folderPath, Name: PathName(folderPath)}
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return f, err
	}
	for _, entry := range entries {
		name := entry.Name()
		switch {
		case entry.IsDir():
			if IsReservedDir(name) {
				continue
			}
			sub, err := s.readFolder(collectionID, joinPath(folderPath, name))
			if err != nil {
				return f, err
			}
			f.Folders = append(f.Folders, sub)
		case isInterfaceFile(name, folderPath):
			req, err := readRequest(filepath.Join(dir, name), joinPath(folderPath, interfaceName(name)))
			if err != nil {
				return f, err
			}
			f.Requests = append(f.Requests, req)
		}
	}
	return f, nil
}

// isReservedDir reports whether a directory belongs to the collection
// itself rather than to its tree.
func IsReservedDir(name string) bool {
	for _, reserved := range reservedDirs {
		if name == reserved {
			return true
		}
	}
	return strings.HasPrefix(name, ".")
}

// isInterfaceFile reports whether a file of a directory is an interface:
// any *.yaml file, but the collection's own apigo.yaml.
func isInterfaceFile(name, folderPath string) bool {
	if !isYAMLFile(name) {
		return false
	}
	return folderPath != "" || name != MetaFile
}

// interfaceName is the name an interface file gives its interface.
func interfaceName(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file))
}

// readRequest reads one interface file, naming the interface after it.
func readRequest(file, treePath string) (Request, error) {
	var req Request
	if err := readYAML(file, &req); err != nil {
		return Request{}, err
	}
	req.Path, req.Name = treePath, PathName(treePath)
	return req, nil
}

// writeFolder writes a folder of the tree and everything under it.
func writeFolder(dir string, f Folder) error {
	for _, sub := range f.Folders {
		subDir := filepath.Join(dir, sub.Name)
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			return err
		}
		if err := writeFolder(subDir, sub); err != nil {
			return err
		}
	}
	for _, req := range f.Requests {
		if err := WriteYAML(filepath.Join(dir, req.Name+".yaml"), req); err != nil {
			return err
		}
	}
	return nil
}

// findFolder returns the folder at a path of the tree, nil when it is gone.
func FindFolder(f *Folder, treePath string) *Folder {
	if f.Path == treePath {
		return f
	}
	for i := range f.Folders {
		if found := FindFolder(&f.Folders[i], treePath); found != nil {
			return found
		}
	}
	return nil
}

// countRequests counts the interfaces of a folder and of the folders
// inside it.
func CountRequests(f Folder) int {
	total := len(f.Requests)
	for _, sub := range f.Folders {
		total += CountRequests(sub)
	}
	return total
}

// addRequest adds an interface file to a folder of a collection, naming it
// after the interface, and writes the collection.
func (s *Store) AddRequest(collectionID, folderPath, name string, req Request) (Request, error) {
	col := s.Collection(collectionID)
	if col == nil {
		return Request{}, fmt.Errorf("没有集合 %q", collectionID)
	}
	f := FindFolder(&col.Root, folderPath)
	if f == nil {
		return Request{}, fmt.Errorf("没有文件夹 %q", folderPath)
	}
	req.Name = uniqueName(requestNames(*f), name)
	req.Path = joinPath(f.Path, req.Name)
	f.Requests = append(f.Requests, req)
	return req, s.saveCollection(*col)
}

// updateRequest writes an edited interface back to the file it came from,
// leaving every other interface alone.
func (s *Store) UpdateRequest(collectionID, requestPath string, req Request) error {
	col := s.Collection(collectionID)
	if col == nil {
		return fmt.Errorf("没有集合 %q", collectionID)
	}
	f := FindFolder(&col.Root, ParentPath(requestPath))
	if f == nil {
		return fmt.Errorf("没有文件夹 %q", ParentPath(requestPath))
	}
	for i := range f.Requests {
		if f.Requests[i].Path != requestPath {
			continue
		}
		req.Path, req.Name = requestPath, f.Requests[i].Name
		f.Requests[i] = req
		return WriteYAML(filepath.Join(s.FolderDir(collectionID, f.Path), req.Name+".yaml"), req)
	}
	return fmt.Errorf("没有接口 %q", requestPath)
}

// parentPath is the folder a tree path is in, "" at the 根目录.
func ParentPath(treePath string) string {
	if dir := path.Dir(treePath); dir != "." {
		return dir
	}
	return ""
}

// addFolder creates a folder in a collection's tree and its directory, naming
// it after the folder. A level of the tree holds one folder of a name, so a
// name already there is refused rather than numbered.
func (s *Store) AddFolder(collectionID, folderPath, name string) (Folder, error) {
	col := s.Collection(collectionID)
	if col == nil {
		return Folder{}, fmt.Errorf("没有集合 %q", collectionID)
	}
	parent := FindFolder(&col.Root, folderPath)
	if parent == nil {
		return Folder{}, fmt.Errorf("没有文件夹 %q", folderPath)
	}
	want := safeFileName(name)
	if want == "" {
		return Folder{}, errors.New("文件夹名称不能为空")
	}
	for _, have := range folderNames(*parent) {
		if have == want {
			return Folder{}, fmt.Errorf("同级目录里已有文件夹「%s」", want)
		}
	}
	created := Folder{Name: want, Path: joinPath(parent.Path, want)}
	parent.Folders = append(parent.Folders, created)
	return created, s.saveCollection(*col)
}

// requestNames lists the interface names a folder holds.
func requestNames(f Folder) []string {
	names := make([]string, 0, len(f.Requests))
	for _, req := range f.Requests {
		names = append(names, req.Name)
	}
	return names
}

// folderNames lists the folder names a folder holds.
func folderNames(f Folder) []string {
	names := make([]string, 0, len(f.Folders))
	for _, sub := range f.Folders {
		names = append(names, sub.Name)
	}
	return names
}

// uniqueName makes a name usable as a file name and free of the names
// already taken, numbering it when it is not.
func uniqueName(taken []string, name string) string {
	want := safeFileName(name)
	if want == "" {
		want = "Untitled"
	}
	used := make(map[string]bool, len(taken))
	for _, have := range taken {
		used[have] = true
	}
	unique := want
	for n := 2; used[unique]; n++ {
		unique = fmt.Sprintf("%s %d", want, n)
	}
	return unique
}

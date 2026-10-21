package store

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// errRootFolder is what deleting or moving the 根目录 reports: it is the
// collection's own directory.
var errRootFolder = errors.New("根目录不能移动或删除")

// foldersOf lists a folder and every folder inside it, 根目录 first, in the
// order the tree shows them.
func FoldersOf(f Folder) []Folder {
	out := []Folder{f}
	for _, sub := range f.Folders {
		out = append(out, FoldersOf(sub)...)
	}
	return out
}

// deleteRequest removes an interface from the tree and deletes its file.
func (s *Store) DeleteRequest(collectionID, requestPath string) error {
	col := s.Collection(collectionID)
	if col == nil {
		return fmt.Errorf("没有集合 %q", collectionID)
	}
	parent := FindFolder(&col.Root, ParentPath(requestPath))
	if parent == nil {
		return fmt.Errorf("没有文件夹 %q", ParentPath(requestPath))
	}
	for i := range parent.Requests {
		if parent.Requests[i].Path != requestPath {
			continue
		}
		parent.Requests = append(parent.Requests[:i], parent.Requests[i+1:]...)
		file := filepath.Join(s.FolderDir(collectionID, parent.Path), path.Base(requestPath)+".yaml")
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return fmt.Errorf("没有接口 %q", requestPath)
}

// deleteFolder removes a folder from the tree and deletes its directory, with
// the interfaces and folders inside it.
func (s *Store) DeleteFolder(collectionID, folderPath string) error {
	if folderPath == "" {
		return errRootFolder
	}
	col := s.Collection(collectionID)
	if col == nil {
		return fmt.Errorf("没有集合 %q", collectionID)
	}
	parent := FindFolder(&col.Root, ParentPath(folderPath))
	if parent == nil {
		return fmt.Errorf("没有文件夹 %q", ParentPath(folderPath))
	}
	for i := range parent.Folders {
		if parent.Folders[i].Path != folderPath {
			continue
		}
		parent.Folders = append(parent.Folders[:i], parent.Folders[i+1:]...)
		if err := os.RemoveAll(s.FolderDir(collectionID, folderPath)); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("没有文件夹 %q", folderPath)
}

// moveRequest moves an interface into another folder of its collection, taking
// its file with it, and returns where it is kept now.
func (s *Store) MoveRequest(collectionID, requestPath, target string) (Request, error) {
	col := s.Collection(collectionID)
	if col == nil {
		return Request{}, fmt.Errorf("没有集合 %q", collectionID)
	}
	src := FindFolder(&col.Root, ParentPath(requestPath))
	dst := FindFolder(&col.Root, target)
	if src == nil || dst == nil {
		return Request{}, fmt.Errorf("没有接口 %q 或文件夹 %q", requestPath, target)
	}
	for i := range src.Requests {
		req := src.Requests[i]
		if req.Path != requestPath {
			continue
		}
		if src.Path == dst.Path {
			return req, nil // already there
		}
		req.Name = uniqueName(requestNames(*dst), req.Name)
		req.Path = joinPath(dst.Path, req.Name)
		src.Requests = append(src.Requests[:i], src.Requests[i+1:]...)
		dst.Requests = append(dst.Requests, req)
		return req, renameFile(
			filepath.Join(s.FolderDir(collectionID, src.Path), path.Base(requestPath)+".yaml"),
			filepath.Join(s.FolderDir(collectionID, dst.Path), req.Name+".yaml"),
		)
	}
	return Request{}, fmt.Errorf("没有接口 %q", requestPath)
}

// moveFolder moves a folder of the tree into another folder of its collection,
// with everything inside it, and returns its place there.
func (s *Store) MoveFolder(collectionID, folderPath, target string) (Folder, error) {
	if folderPath == "" {
		return Folder{}, errRootFolder
	}
	if target == folderPath || strings.HasPrefix(target, folderPath+"/") {
		return Folder{}, fmt.Errorf("不能把文件夹 %q 移进它自己", PathName(folderPath))
	}
	col := s.Collection(collectionID)
	if col == nil {
		return Folder{}, fmt.Errorf("没有集合 %q", collectionID)
	}
	src := FindFolder(&col.Root, ParentPath(folderPath))
	dst := FindFolder(&col.Root, target)
	if src == nil || dst == nil {
		return Folder{}, fmt.Errorf("没有文件夹 %q 或 %q", folderPath, target)
	}
	if src.Path == dst.Path {
		for _, sub := range src.Folders {
			if sub.Path == folderPath {
				return sub, nil // already there
			}
		}
	}
	for i := range src.Folders {
		sub := src.Folders[i]
		if sub.Path != folderPath {
			continue
		}
		before := sub.Path
		src.Folders = append(src.Folders[:i], src.Folders[i+1:]...)
		// Taking the folder out may have moved the folders the target sits
		// among, so it is found again rather than kept as a pointer.
		if dst = FindFolder(&col.Root, target); dst == nil {
			return Folder{}, fmt.Errorf("没有文件夹 %q", target)
		}
		sub.Name = uniqueName(folderNames(*dst), sub.Name)
		renamePaths(&sub, joinPath(dst.Path, sub.Name))
		dst.Folders = append(dst.Folders, sub)
		return sub, renameFile(
			s.FolderDir(collectionID, before),
			s.FolderDir(collectionID, sub.Path),
		)
	}
	return Folder{}, fmt.Errorf("没有文件夹 %q", folderPath)
}

// renamePaths moves the paths of a folder's tree to where the folder went.
func renamePaths(f *Folder, to string) {
	f.Path = to
	for i := range f.Folders {
		renamePaths(&f.Folders[i], joinPath(to, f.Folders[i].Name))
	}
	for i := range f.Requests {
		f.Requests[i].Path = joinPath(to, f.Requests[i].Name)
	}
}

// renameFile moves a file or directory, treating one that is not there as
// nothing to move.
func renameFile(from, to string) error {
	if _, err := os.Stat(from); os.IsNotExist(err) {
		return nil
	}
	return os.Rename(from, to)
}

// renameRequest renames an interface, taking its file with it, and returns
// where it is kept now: a name another interface of the folder holds is
// numbered rather than taken.
func (s *Store) RenameRequest(collectionID, requestPath, name string) (Request, error) {
	col := s.Collection(collectionID)
	if col == nil {
		return Request{}, fmt.Errorf("没有集合 %q", collectionID)
	}
	parent := FindFolder(&col.Root, ParentPath(requestPath))
	if parent == nil {
		return Request{}, fmt.Errorf("没有文件夹 %q", ParentPath(requestPath))
	}
	for i := range parent.Requests {
		req := &parent.Requests[i]
		if req.Path != requestPath {
			continue
		}
		taken := requestNames(*parent)
		want := uniqueName(withoutName(taken, req.Name), name)
		if want == req.Name {
			return *req, nil
		}
		from := filepath.Join(s.FolderDir(collectionID, parent.Path), req.Name+".yaml")
		to := filepath.Join(s.FolderDir(collectionID, parent.Path), want+".yaml")
		req.Name, req.Path = want, joinPath(parent.Path, want)
		return *req, renameFile(from, to)
	}
	return Request{}, fmt.Errorf("没有接口 %q", requestPath)
}

// withoutName lists names but one, as renaming an interface to its own name
// keeps it.
func withoutName(names []string, name string) []string {
	kept := make([]string, 0, len(names))
	for _, have := range names {
		if have != name {
			kept = append(kept, have)
		}
	}
	return kept
}

// renameFolder renames a folder of the tree, taking its directory with it and
// every path inside it, and returns where it is kept now.
func (s *Store) RenameFolder(collectionID, folderPath, name string) (Folder, error) {
	if folderPath == "" {
		return Folder{}, errRootFolder
	}
	col := s.Collection(collectionID)
	if col == nil {
		return Folder{}, fmt.Errorf("没有集合 %q", collectionID)
	}
	parent := FindFolder(&col.Root, ParentPath(folderPath))
	if parent == nil {
		return Folder{}, fmt.Errorf("没有文件夹 %q", ParentPath(folderPath))
	}
	for i := range parent.Folders {
		sub := &parent.Folders[i]
		if sub.Path != folderPath {
			continue
		}
		want := uniqueName(withoutName(folderNames(*parent), sub.Name), name)
		if want == sub.Name {
			return *sub, nil
		}
		from := s.FolderDir(collectionID, folderPath)
		renamePaths(sub, joinPath(parent.Path, want))
		return *sub, renameFile(from, s.FolderDir(collectionID, sub.Path))
	}
	return Folder{}, fmt.Errorf("没有文件夹 %q", folderPath)
}

// duplicateRequest copies an interface beside itself, under a name of its own,
// and returns the copy.
func (s *Store) DuplicateRequest(collectionID, requestPath string) (Request, error) {
	col := s.Collection(collectionID)
	if col == nil {
		return Request{}, fmt.Errorf("没有集合 %q", collectionID)
	}
	parent := FindFolder(&col.Root, ParentPath(requestPath))
	if parent == nil {
		return Request{}, fmt.Errorf("没有文件夹 %q", ParentPath(requestPath))
	}
	for _, req := range parent.Requests {
		if req.Path != requestPath {
			continue
		}
		return s.AddRequest(collectionID, parent.Path, req.Name+" 副本", req)
	}
	return Request{}, fmt.Errorf("没有接口 %q", requestPath)
}

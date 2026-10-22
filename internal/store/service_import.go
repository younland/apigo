package store

import "strings"

// ImportOptions says what a parsed document becomes: a collection of its own,
// or the update of the collection of the same name.
type ImportOptions struct {
	// Name names the collection, as the document titles it.
	Name string
	// BaseURL is the server the document was written against: the collection's
	// first environment carries it as its BaseURL variable.
	BaseURL string
	// Root is the tree of interfaces the document describes.
	Root Folder
}

// ImportResult says what an import did: the collection it wrote, whether it
// made it, and how many interfaces it added and wrote over.
type ImportResult struct {
	Collection Collection
	Created    bool
	Added      int
	Updated    int
}

// Import writes the interfaces of a document as a collection. A collection of
// the same name is updated rather than made again: an interface whose name a
// folder already holds is written over, the ones it does not hold are added
// beside it, and the interfaces the document says nothing of stay as they are.
func (s *Store) Import(opts ImportOptions) (ImportResult, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = defaultCollectionName
	}
	col, created := s.collectionNamed(name)
	if created {
		col.Root = Folder{Name: RootFolderName}
	}
	added, updated := mergeFolder(&col.Root, opts.Root)
	normalizeTree(&col.Root, "")
	if err := s.saveCollection(col); err != nil {
		return ImportResult{}, err
	}
	if created {
		if err := s.addDefaultEnvironments(col.ID); err != nil {
			return ImportResult{}, err
		}
	}
	if err := s.setBaseURL(col.ID, opts.BaseURL); err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Collection: *s.Collection(col.ID), Created: created, Added: added, Updated: updated}, nil
}

// collectionNamed is the collection a name is taken by: an existing one, which
// the import updates, or a new one with a directory of its own.
func (s *Store) collectionNamed(name string) (Collection, bool) {
	for _, col := range s.Collections {
		if col.Name == name {
			return col, false
		}
	}
	return Collection{ID: s.uniqueCollectionID(name, ""), Name: name}, true
}

// mergeFolder writes the interfaces of a document into a folder of a
// collection, matching the folders and the interfaces of both by name.
func mergeFolder(dst *Folder, src Folder) (added, updated int) {
	for _, req := range src.Requests {
		if i := requestIndex(*dst, req.Name); i >= 0 {
			dst.Requests[i] = overwritten(req, dst.Requests[i])
			updated++
			continue
		}
		dst.Requests = append(dst.Requests, req)
		added++
	}
	for _, folder := range src.Folders {
		if i := folderIndex(*dst, folder.Name); i >= 0 {
			a, u := mergeFolder(&dst.Folders[i], folder)
			added, updated = added+a, updated+u
			continue
		}
		dst.Folders = append(dst.Folders, folder)
		added += CountRequests(folder)
	}
	return added, updated
}

// overwritten is what an imported interface leaves in the place of the one it
// writes over: its own request, under the name and the place of the file the
// collection already keeps.
func overwritten(req, existing Request) Request {
	req.Path, req.Name = existing.Path, existing.Name
	if req.Kind == "" {
		req.Kind = KindREST
	}
	return req
}

// requestIndex is where a folder keeps the interface of a name, -1 when it
// keeps none.
func requestIndex(f Folder, name string) int {
	want := safeFileName(name)
	for i, req := range f.Requests {
		if safeFileName(req.Name) == want {
			return i
		}
	}
	return -1
}

// folderIndex is where a folder keeps the folder of a name, -1 when it keeps
// none.
func folderIndex(f Folder, name string) int {
	want := safeFileName(name)
	for i, sub := range f.Folders {
		if safeFileName(sub.Name) == want {
			return i
		}
	}
	return -1
}

// normalizeTree makes the names of a tree usable as file names and unique in
// their folder, and gives every folder and interface the place it is kept at.
func normalizeTree(f *Folder, path string) {
	f.Path, f.Name = path, PathName(path)
	names := make([]string, 0, len(f.Requests))
	for i := range f.Requests {
		name := uniqueName(names, f.Requests[i].Name)
		names = append(names, name)
		f.Requests[i].Name = name
		f.Requests[i].Path = joinPath(path, name)
		if f.Requests[i].Kind == "" {
			f.Requests[i].Kind = KindREST
		}
	}
	folders := make([]string, 0, len(f.Folders))
	for i := range f.Folders {
		name := uniqueName(folders, f.Folders[i].Name)
		folders = append(folders, name)
		f.Folders[i].Name = name
		normalizeTree(&f.Folders[i], joinPath(path, name))
	}
}

// setBaseURL writes the server of an imported document into the baseurl of a
// collection's first environment, which is the one it opens on.
func (s *Store) setBaseURL(collectionID, baseURL string) error {
	if baseURL == "" {
		return nil
	}
	for i := range s.Environments {
		env := &s.Environments[i]
		if env.Scope != collectionID {
			continue
		}
		for j := range env.Vars {
			if env.Vars[j].Key == BaseURLVariable {
				env.Vars[j].Value, env.Vars[j].Enabled = baseURL, true
			}
		}
		return s.SaveEnvironments()
	}
	return nil
}

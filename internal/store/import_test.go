package store

import (
	"os"
	"path/filepath"
	"testing"
)

// importTree is what a parsed document holds: a folder of two interfaces and
// one interface at the top of the tree.
func importTree() Folder {
	return Folder{Requests: []Request{
		{Kind: KindREST, Method: "GET", URL: "https://api.example.com/pets"},
	}, Folders: []Folder{{Name: "订单", Requests: []Request{
		{Kind: KindREST, Method: "POST", URL: "https://api.example.com/orders"},
	}}}}
}

func TestImportMakesACollection(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	tree := importTree()
	tree.Requests[0].Name = "列出宠物"
	tree.Folders[0].Requests[0].Name = "新建订单"
	res, err := s.Import(ImportOptions{Name: "宠物商店", BaseURL: "https://api.example.com", Root: tree})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created || res.Added != 2 || res.Updated != 0 {
		t.Fatalf("result %+v, want a new collection with two interfaces", res)
	}
	if res.Collection.ID != "宠物商店" || res.Collection.Name != "宠物商店" {
		t.Errorf("collection %+v, want it named after the document", res.Collection)
	}
	col := s.Collection(res.Collection.ID)
	if col == nil || CountRequests(col.Root) != 2 {
		t.Fatalf("collection %+v, want the two interfaces of the document", col)
	}
	folder := FindFolder(&col.Root, "订单")
	if folder == nil || len(folder.Requests) != 1 || folder.Requests[0].Path != "订单/新建订单" {
		t.Fatalf("folder %+v, want the interface at 订单/新建订单", folder)
	}
	// Every interface is a file of its own, written where the tree says.
	file := filepath.Join(dir, "collections", "宠物商店", "订单", "新建订单.yaml")
	if _, err := os.Stat(file); err != nil {
		t.Errorf("%s missing: %v", file, err)
	}
	// The collection comes with its environments, the first one holding the
	// server of the document.
	envs := s.EnvironmentsFor(col.ID)
	if len(envs) != len(defaultEnvironmentNames) {
		t.Fatalf("environments %+v, want the defaults", envs)
	}
	if got := envs[0].Vars[0]; got.Key != BaseURLVariable || got.Value != "https://api.example.com" {
		t.Errorf("variable %+v, want the server of the document", got)
	}
}

func TestImportUpdatesTheCollectionOfTheName(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.NewCollection("宠物商店"); err != nil {
		t.Fatal(err)
	}
	// An interface the collection already holds, under the name the document
	// will use, and one the document says nothing about.
	if _, err := s.AddRequest("宠物商店", "", "列出宠物", Request{
		Kind: KindREST, Method: "GET", URL: "http://localhost/pets",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddRequest("宠物商店", "", "本地接口", Request{Kind: KindREST, URL: "http://localhost/local"}); err != nil {
		t.Fatal(err)
	}
	tree := Folder{Requests: []Request{{Kind: KindREST, Method: "GET", URL: "https://api.example.com/pets"}}}
	tree.Requests[0].Name = "列出宠物"
	res, err := s.Import(ImportOptions{Name: "宠物商店", Root: tree})
	if err != nil {
		t.Fatal(err)
	}
	if res.Created || res.Updated != 1 || res.Added != 0 {
		t.Fatalf("result %+v, want the interface written over", res)
	}
	col := s.Collection("宠物商店")
	if col == nil {
		t.Fatal("the collection went away")
	}
	if CountRequests(col.Root) != 2 {
		t.Fatalf("collection holds %d interfaces, want the two it had", CountRequests(col.Root))
	}
	updated := col.Root.Requests[0]
	if updated.URL != "https://api.example.com/pets" {
		t.Errorf("interface %+v, want the url of the document", updated)
	}
	// The interface the document does not name stays where it was.
	if col.Root.Requests[1].Name != "本地接口" {
		t.Errorf("interface %+v, want the local one kept", col.Root.Requests[1])
	}
}

func TestImportMergesTheFoldersOfOneName(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	// A document that names the same folder twice: its interfaces land in one
	// folder, and the names it takes twice are numbered.
	tree := Folder{Folders: []Folder{{Name: "分组"}, {Name: "分组"}}}
	tree.Folders[0].Requests = []Request{{Name: "同一个"}, {Name: "同一个"}}
	tree.Folders[1].Requests = []Request{{Name: "别的"}}
	res, err := s.Import(ImportOptions{Name: "重复", Root: tree})
	if err != nil {
		t.Fatal(err)
	}
	col := s.Collection(res.Collection.ID)
	if col == nil {
		t.Fatal("no collection imported")
	}
	if len(col.Root.Folders) != 1 {
		t.Fatalf("folders %+v, want the one folder the document names", col.Root.Folders)
	}
	folder := col.Root.Folders[0]
	if CountRequests(folder) != 3 {
		t.Fatalf("folder %+v, want the three interfaces", folder)
	}
	if name := folder.Requests[1].Name; name != "同一个 2" {
		t.Errorf("second interface %q, want it numbered", name)
	}
	if path := folder.Requests[1].Path; path != "分组/同一个 2" {
		t.Errorf("path %q, want the place the interface is kept at", path)
	}
}

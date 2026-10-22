package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCollectionLayoutOnDisk pins the layout every collection has on disk:
// one directory per collection, apigo.yaml for the collection itself, .env
// for its environments, static for uploads, and one file per interface.
func TestCollectionLayoutOnDisk(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	if col.ID != "Demo" {
		t.Fatalf("collection id %q, want the directory named after it", col.ID)
	}
	Folder, err := s.AddFolder(col.ID, "", "设备")
	if err != nil {
		t.Fatal(err)
	}
	if Folder.Path != "设备" {
		t.Fatalf("folder path %q, want 设备", Folder.Path)
	}
	req, err := s.AddRequest(col.ID, Folder.Path, "列表", Request{
		Kind: KindREST, Method: "GET", URL: "https://api.example.com/pets",
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.Path != "设备/列表" || req.Name != "列表" {
		t.Fatalf("request %+v, want 设备/列表", req)
	}

	base := filepath.Join(dir, "collections", "Demo")
	for _, want := range []string{"apigo.yaml", ".env", "static", "设备"} {
		if _, err := os.Stat(filepath.Join(base, want)); err != nil {
			t.Errorf("%s missing from the collection directory: %v", want, err)
		}
	}
	for _, env := range defaultEnvironmentNames {
		if _, err := os.Stat(filepath.Join(base, ".env", env+".yaml")); err != nil {
			t.Errorf("environment file %s.yaml missing: %v", env, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(base, "设备", "列表.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"kind: rest", "method: GET", "url: https://api.example.com/pets"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("interface file %q, want it to contain %q", data, want)
		}
	}
	if strings.Contains(string(data), "path:") || strings.Contains(string(data), "name:") {
		t.Errorf("interface file %q, want no copied name or path", data)
	}
}

// TestLoadReadsTreeFromDisk reads a tree that was never written by the app:
// folders of their own, however deep, become folders of the tree.
func TestLoadReadsTreeFromDisk(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "collections", "demo")
	files := map[string]string{
		"apigo.yaml":     "name: Demo\ndescription: 示例集合\n",
		".env/本地.yaml":   "id: env-1\nname: 本地\nactive: true\nvariables:\n  - enabled: true\n    key: BaseURL\n",
		"顶层.yaml":        "kind: rest\nmethod: GET\nurl: https://api.example.com/top\n",
		"设备/列表.yaml":     "kind: rest\nmethod: GET\n",
		"设备/传感器/温度.yaml": "kind: rest\nmethod: POST\n",
		"static/上传.txt":  "payload",
		"设备/note.txt":    "not an interface",
	}
	for name, body := range files {
		path := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := New(dir)
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s.Collections) != 1 {
		t.Fatalf("collections %+v, want the one directory", s.Collections)
	}
	col := s.Collections[0]
	if col.ID != "demo" || col.Name != "Demo" || col.Desc != "示例集合" {
		t.Fatalf("collection %+v, want Demo read from apigo.yaml", col)
	}
	if len(col.Root.Requests) != 1 || col.Root.Requests[0].Name != "顶层" {
		t.Errorf("root requests %+v, want only 顶层", col.Root.Requests)
	}
	if len(col.Root.Folders) != 1 || col.Root.Folders[0].Name != "设备" {
		t.Fatalf("folders %+v, want only 设备", col.Root.Folders)
	}
	device := col.Root.Folders[0]
	if len(device.Requests) != 1 || device.Requests[0].Name != "列表" {
		t.Errorf("设备 requests %+v, want only 列表", device.Requests)
	}
	if len(device.Folders) != 1 || device.Folders[0].Path != "设备/传感器" {
		t.Fatalf("nested folders %+v, want 设备/传感器", device.Folders)
	}
	if got := CountRequests(col.Root); got != 3 {
		t.Errorf("countRequests = %d, want 3", got)
	}
	if envs := s.EnvironmentsFor("demo"); len(envs) != 1 || envs[0].Name != "本地" {
		t.Errorf("environments %+v, want the one in .env", envs)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddRequest(col.ID, "", "Pets", Request{
		Kind: KindREST, Method: "GET", URL: "https://api.example.com/pets",
		Params:   []Row{{Enabled: true, Key: "limit", Value: "10"}},
		BodyType: "json", Body: `{"a":1}`,
	}); err != nil {
		t.Fatal(err)
	}
	env := s.ActiveEnvironment(col.ID)
	if env == nil {
		t.Fatal("a new collection has no active environment")
	}
	env.Vars = []Row{{Enabled: true, Key: "BaseURL", Value: "http://localhost:8080"}}
	if err := s.SaveEnvironments(); err != nil {
		t.Fatal(err)
	}

	loaded := New(dir)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Collections) != 1 {
		t.Fatalf("collections %+v, want the saved one", loaded.Collections)
	}
	saved := loaded.Collections[0]
	if saved.Name != "Demo" || len(saved.Root.Requests) != 1 {
		t.Fatalf("collection %+v, want Demo with one interface", saved)
	}
	req := saved.Root.Requests[0]
	if req.Method != "GET" || req.URL != "https://api.example.com/pets" || req.Kind != KindREST {
		t.Errorf("request %+v, want the saved REST request", req)
	}
	if len(req.Params) != 1 || req.Params[0].Key != "limit" {
		t.Errorf("params %+v, want the saved row", req.Params)
	}
	envs := loaded.EnvironmentsFor(saved.ID)
	if len(envs) != 3 {
		t.Fatalf("environments %+v, want the three defaults", envs)
	}
	active := loaded.ActiveEnvironment(saved.ID)
	if active == nil || active.Vars[0].Value != "http://localhost:8080" {
		t.Errorf("active environment %+v, want the saved BaseURL", active)
	}
}

func TestEnvironmentRenameMovesFile(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	env := s.ActiveEnvironment(col.ID)
	if env == nil {
		t.Fatal("a new collection has no active environment")
	}
	name, err := s.renameEnvironment(env.ID, "预发")
	if err != nil {
		t.Fatal(err)
	}
	if name != "预发" {
		t.Errorf("renamed to %q, want 预发", name)
	}
	EnvDir := filepath.Join(dir, "collections", "Demo", ".env")
	if _, err := os.Stat(filepath.Join(EnvDir, "预发.yaml")); err != nil {
		t.Errorf("预发.yaml missing after the rename: %v", err)
	}
	if _, err := os.Stat(filepath.Join(EnvDir, "本地.yaml")); !os.IsNotExist(err) {
		t.Errorf("本地.yaml still there after the rename: %v", err)
	}
	if got := s.ActiveEnvironment(col.ID); got == nil || got.ID != env.ID || !got.Active {
		t.Errorf("active environment %+v, want the renamed one still active", got)
	}
}

// TestRenameCollectionMovesDirectory checks a renamed collection keeps its
// interfaces and environments: only the directory's name changes.
func TestRenameCollectionMovesDirectory(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddRequest(col.ID, "", "Pets", Request{
		Kind: KindREST, Method: "GET", URL: "https://api.example.com/pets",
	}); err != nil {
		t.Fatal(err)
	}

	renamed, err := s.RenameCollection(col.ID, "订单服务")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.ID != "订单服务" || renamed.Name != "订单服务" {
		t.Fatalf("collection %+v, want it named by its new directory", renamed)
	}
	if _, err := os.Stat(filepath.Join(dir, "collections", "Demo")); !os.IsNotExist(err) {
		t.Errorf("old directory still there after the rename: %v", err)
	}
	base := filepath.Join(dir, "collections", "订单服务")
	if _, err := os.Stat(filepath.Join(base, "Pets.yaml")); err != nil {
		t.Errorf("interface did not move with the collection: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".env", "本地.yaml")); err != nil {
		t.Errorf("environment did not move with the collection: %v", err)
	}
	if envs := s.EnvironmentsFor(renamed.ID); len(envs) != 3 {
		t.Errorf("environments %+v, want them following the new id", envs)
	}
	meta, err := os.ReadFile(filepath.Join(base, "apigo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), "name: 订单服务") {
		t.Errorf("apigo.yaml %q, want the new name", meta)
	}

	loaded := New(dir)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Collections) != 1 {
		t.Fatalf("collections %+v, want the renamed one only", loaded.Collections)
	}
	saved := loaded.Collections[0]
	if saved.Name != "订单服务" || len(saved.Root.Requests) != 1 {
		t.Errorf("collection %+v, want the renamed one with its interface", saved)
	}
	if len(loaded.EnvironmentsFor(saved.ID)) != 3 {
		t.Errorf("environments %+v after reload, want three", loaded.Environments)
	}
}

func TestNewCollectionHasDefaultEnvironments(t *testing.T) {
	s := New(t.TempDir())
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	envs := s.EnvironmentsFor(col.ID)
	if len(envs) != 3 {
		t.Fatalf("environments %+v, want 本地 / 测试 / 生产", envs)
	}
	names := []string{envs[0].Name, envs[1].Name, envs[2].Name}
	if names[0] != "本地" || names[1] != "测试" || names[2] != "生产" {
		t.Errorf("names %v, want 本地 / 测试 / 生产", names)
	}
	for _, env := range envs {
		if len(env.Vars) != 1 || env.Vars[0].Key != "BaseURL" {
			t.Errorf("environment %+v, want an initialized BaseURL", env)
		}
	}
	if !envs[0].Active {
		t.Error("本地 is not the active environment of a new collection")
	}
	if active := s.ActiveEnvironment(col.ID); active == nil || active.Name != "本地" {
		t.Errorf("active %+v, want 本地", active)
	}
}

func TestNewCollectionDirectoriesDiffer(t *testing.T) {
	s := New(t.TempDir())
	first, err := s.NewCollection("订单服务")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.NewCollection("订单服务")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID {
		t.Fatalf("both collections live in %q", first.ID)
	}
	if second.Name != "订单服务" {
		t.Errorf("second name %q, want the name the user typed", second.Name)
	}
}

func TestSafeFileName(t *testing.T) {
	for _, test := range []struct{ in, want string }{
		{"列表", "列表"},
		{"a/b:c", "a-b-c"},
		{" 空格 ", "空格"},
		{"点.", "点"},
		{"", ""},
	} {
		if got := safeFileName(test.in); got != test.want {
			t.Errorf("safeFileName(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

// TestFirstWriteStaysInTheCollection checks the first write of a collection,
// its own apigo.yaml, waits in the collection: nothing is written above it,
// where a walk up looking for apigo.yaml would end at the volume root.
func TestFirstWriteStaysInTheCollection(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.CollectionDir(col.ID), TmpDirName)); err != nil {
		t.Errorf("temp directory: %v, want the writes kept in the collection", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), TmpDirName)); !os.IsNotExist(err) {
		t.Errorf("a temp directory was made above the store, want none")
	}
}

func TestEnvironmentLifecycle(t *testing.T) {
	s := New(t.TempDir())
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	env, err := s.addEnvironment(col.ID, "预发")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.EnvironmentsFor(col.ID)) != 4 {
		t.Fatalf("environments %+v, want the new one added", s.Environments)
	}
	if err := s.ActivateEnvironment(env.ID); err != nil {
		t.Fatal(err)
	}
	if got := s.ActiveEnvironment(col.ID); got == nil || got.ID != env.ID {
		t.Fatalf("active %+v, want the new environment", got)
	}
	if err := s.deleteEnvironment(env.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.EnvironmentsFor(col.ID)) != 3 {
		t.Errorf("environments %+v, want it deleted", s.Environments)
	}
	if got := s.ActiveEnvironment(col.ID); got == nil || got.Name != "本地" {
		t.Errorf("active %+v after deleting the active one, want another to take over", got)
	}
	if _, err := os.Stat(filepath.Join(s.CollectionDir(col.ID), ".env", "预发.yaml")); !os.IsNotExist(err) {
		t.Errorf("file of the deleted environment is still there: %v", err)
	}
}

func TestVarsForMergesGlobalThenCollection(t *testing.T) {
	s := New("")
	s.Environments = []Environment{
		{ID: "global", Scope: scopeGlobal, Active: true, Vars: []Row{
			{Enabled: true, Key: "host", Value: "global.example.com"},
			{Enabled: true, Key: "token", Value: "t"},
			{Enabled: false, Key: "off", Value: "x"},
		}},
		{ID: "collection", Scope: "collection-1", Active: true, Vars: []Row{
			{Enabled: true, Key: "host", Value: "collection.example.com"},
		}},
	}
	vars := s.VarsFor("collection-1")
	if vars["host"] != "collection.example.com" {
		t.Errorf("host %q, want the collection's value", vars["host"])
	}
	if vars["token"] != "t" {
		t.Errorf("token %q, want the global value", vars["token"])
	}
	if _, ok := vars["off"]; ok {
		t.Error("disabled variable included")
	}
	if vars := s.VarsFor("collection-2"); vars["host"] != "global.example.com" {
		t.Errorf("host %q in another collection, want the global value", vars["host"])
	}
}

// TestStoreDirAt checks the chosen data directory is read back from the
// pointer file the default one keeps.
func TestStoreDirAt(t *testing.T) {
	base := t.TempDir()
	if got := DirAt(base); got != base {
		t.Errorf("storeDirAt of an untouched base = %q, want %q", got, base)
	}
	other := t.TempDir()
	if err := RememberDir(base, other); err != nil {
		t.Fatal(err)
	}
	if got := DirAt(base); got != other {
		t.Errorf("storeDirAt = %q, want the chosen %q", got, other)
	}
	var l location
	if err := readYAML(locationPath(base), &l); err != nil {
		t.Fatal(err)
	}
	if l.DataDir != other {
		t.Errorf("location %+v, want the chosen directory", l)
	}
}

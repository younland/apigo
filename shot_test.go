package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"apigo/internal/design"
	"apigo/widgets"

	"github.com/egoist/mygo/ui"
)

// shotStore is the directory the demo app reads and writes: TestShots points
// it at the test's own directory, so a shot never leaves a collection beside
// the repository. Empty means the app's default data directory.
var shotStore string

// demoApp is an app with one collection, its default-equivalent
// environment, and the collection open on one of its requests.
func demoApp() *app {
	a := newApp()
	a.store = newStore(shotStore)
	a.store.collections = []collection{
		{
			ID: "collection-1", Name: "示例接口集",
			Root: folder{Name: rootFolderName, Requests: []savedRequest{
				{Path: "列出宠物", Name: "列出宠物", Kind: kindREST, Method: "GET", URL: "https://api.example.com/v1/pets"},
				{Path: "新建宠物", Name: "新建宠物", Kind: kindREST, Method: "POST", URL: "https://api.example.com/v1/pets"},
			}, Folders: []folder{
				{Path: "设备", Name: "设备", Requests: []savedRequest{
					{Path: "设备/订阅设备事件", Name: "订阅设备事件", Kind: kindWebSocket},
					{Path: "设备/设备列表", Name: "设备列表", Kind: kindREST, Method: "GET"},
				}, Folders: []folder{
					{Path: "设备/传感器", Name: "传感器", Requests: []savedRequest{
						{Path: "设备/传感器/温度", Name: "温度", Kind: kindREST, Method: "GET"},
					}},
				}},
				{Path: "用户", Name: "用户", Requests: []savedRequest{
					{Path: "用户/查询 Schema", Name: "查询 Schema", Kind: kindGraphQL},
				}},
			}},
		},
		{ID: "collection-2", Name: "空白示例集", Root: folder{Name: rootFolderName}},
	}
	a.store.environments = []environment{
		{ID: "env-1", Name: "本地", Scope: "collection-1", Active: true, Vars: []savedRow{
			{Enabled: true, Key: "BaseURL", Value: "http://localhost:8080"},
			{Enabled: true, Key: "token", Value: "dev-token"},
		}},
		{ID: "env-2", Name: "测试", Scope: "collection-1", Vars: []savedRow{
			{Enabled: true, Key: "BaseURL", Value: "https://test.example.com"},
		}},
		{ID: "env-3", Name: "生产", Scope: "collection-1", Vars: []savedRow{
			{Enabled: true, Key: "BaseURL", Value: "https://api.example.com"},
		}},
	}
	a.openCollection("collection-1")
	// A collection opens with no tab at all, so the shots start from one of its
	// requests being open. The tab is seeded in memory rather than opened from
	// the tree: the demo collection has no files, and opening a request would
	// write the tab back beside the repository.
	t := newTab()
	t.kind, t.collectionID = kindREST, "collection-1"
	t.title, t.method, t.url = "列出宠物", "GET", "https://api.example.com/v1/pets"
	t.path, t.folder = "列出宠物", ""
	t.markSaved()
	a.openTab(a.space(), t, 0)
	return a
}

// settingsDemo is the demo app with the settings dialog over it. The data
// directory reads as an example path rather than the test's own: a shot
// should not carry the name of the machine that made it.
func settingsDemo() *app {
	const exampleDir = `C:\Users\you\AppData\Roaming\apigo`
	a := demoApp()
	a.store.dir = exampleDir
	a.settingsOpen = true
	return a
}

// TestShots writes the window's screens as PNGs for a layout review, and
// for the screenshots CI keeps as artifacts. It runs only when APIGO_SHOTS
// names the directory to write to:
//
//	APIGO_SHOTS=docs/shots/m5 go test -run TestShots .
func TestShots(t *testing.T) {
	dir := os.Getenv("APIGO_SHOTS")
	if dir == "" {
		t.Skip("set APIGO_SHOTS to the directory to write screenshots to")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The demo collection has no files, so whatever a shot writes goes to the
	// test's own directory rather than beside the repository.
	stores := t.TempDir()
	shotStore = stores
	shots := []struct {
		name  string
		mode  design.Mode
		setup func(*app)
	}{
		{"home-empty", design.Light, nil},
		{"home-new-collection", design.Light, func(a *app) { a.openCollectionModal() }},
		{"home-list", design.Light, func(a *app) {
			a.store = demoApp().store
		}},
		{"home-row-menu", design.Light, func(a *app) {
			a.store = demoApp().store
			open := a.menuFlag("collection/collection-1")
			*open = true
		}},
		{"home-rename-collection", design.Light, func(a *app) {
			a.store = demoApp().store
			a.openRenameModal("collection-1")
		}},
		{"tab-menu", design.Light, func(a *app) {
			*a = *demoApp()
			open := a.menuFlag("tab/collection-1")
			*open = true
		}},
		{"request-tab-menu", design.Light, func(a *app) {
			*a = *demoApp()
			open := a.menuFlag(requestTabMenuKey(a.current()))
			*open = true
		}},
		{"collection-request", design.Light, func(a *app) { *a = *demoApp() }},
		{"start-hint", design.Light, func(a *app) {
			// A collection opened, nothing chosen yet: the pane says what to do.
			*a = *demoApp()
			a.spaces["collection-1"] = newWorkspace()
		}},
		{"root-add-menu", design.Light, func(a *app) {
			*a = *demoApp()
			open := a.menuFlag("folder/")
			*open = true
		}},
		{"root-add-menu-dark", design.Dark, func(a *app) {
			*a = *demoApp()
			open := a.menuFlag("folder/")
			*open = true
		}},
		{"collection-dark", design.Dark, func(a *app) { *a = *demoApp() }},
		{"collection-black", design.Black, func(a *app) { *a = *demoApp() }},
		{"request-menu", design.Light, func(a *app) {
			*a = *demoApp()
			open := a.menuFlag("request-menu/设备/设备列表")
			*open = true
		}},
		{"save-dialog", design.Light, func(a *app) {
			*a = *demoApp()
			a.current().title = "新的接口"
			a.askWhereToSave(a.current())
		}},
		{"tree-open-request", design.Light, func(a *app) {
			*a = *demoApp()
			col := a.store.collection("collection-1")
			if col == nil {
				return
			}
			if req, ok := findRequest(col.Root, "设备列表"); ok {
				a.openRequest("collection-1", req)
			}
		}},
		{"request-filled", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			t.url = "https://api.example.com/v1/pets"
			t.params.Rows = []widgets.Row{
				{Enabled: true, Key: "limit", Value: "10"},
				{Enabled: true, Key: "cursor", Value: "eyJpZCI6MTB9"},
				{Enabled: false, Key: "debug", Value: "1"},
			}
			t.headers.Rows = []widgets.Row{{Enabled: true, Key: "Accept", Value: "application/json"}}
			a.addTab(a.space())
			a.current().method = "POST"
		}},
		{"response-json", design.Dark, func(a *app) {
			*a = *demoApp()
			t := a.current()
			t.url = "https://api.example.com/v1/pets"
			a.apply(t, &response{
				status: 200, statusText: "OK", duration: 184 * time.Millisecond, size: 156,
				body: "{\n  \"data\": [\n    {\"id\": 1, \"name\": \"Ada\"},\n    {\"id\": 2, \"name\": \"Grace\"}\n  ],\n  \"next\": null\n}",
			})
		}},
		{"body-json", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			t.url = "https://api.example.com/v1/pets"
			t.reqTab = 2
			t.bodyType = "json"
			t.setBodyLanguage()
			t.body.Set("{\n  \"name\": \"Ada\",\n  \"tags\": [\"cat\"]\n}")
		}},
		{"body-form", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			t.url = "https://api.example.com/v1/pets"
			t.reqTab = 2
			t.bodyType = "urlencoded"
			t.form.Rows = []widgets.Row{
				{Enabled: true, Key: "name", Value: "Ada"},
				{Enabled: true, Key: "tags", Value: "cat"},
				{Enabled: false, Key: "debug", Value: "1"},
			}
		}},
		{"body-binary", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			t.url = "https://api.example.com/v1/pets"
			t.reqTab = 2
			t.bodyType = "binary"
		}},
		{"body-form-file", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			t.url = "https://api.example.com/v1/pets"
			t.reqTab = 2
			t.bodyType = "form-data"
			t.form.Rows = []widgets.Row{
				{Enabled: true, Key: "参数 1", File: "static/2025-10-20/9f2c", FileName: "Vue-Go冲锋Logo.png"},
				{Enabled: true, Key: "备注", Value: "说明文本"},
				{Enabled: true},
			}
		}},
		{"tab-renaming", design.Light, func(a *app) {
			*a = *demoApp()
			a.addTab(a.space())
			a.space().active = 0
			a.current().startRename()
			a.current().title = "获取宠物列表"
		}},
		{"more-menu", design.Light, func(a *app) {
			*a = *demoApp()
			a.closed = []string{"collection-2"}
			a.moreOpen = true
		}},
		{"env-modal", design.Light, func(a *app) {
			*a = *demoApp()
			a.envOpen = true
			a.openEnvManager()
		}},
		{"settings-modal", design.Light, func(a *app) {
			*a = *settingsDemo()
		}},
		{"delete-collection-confirm", design.Light, func(a *app) {
			*a = *demoApp()
			a.askToDeleteCollection("collection-1")
		}},
		{"close-collection-confirm", design.Light, func(a *app) {
			*a = *demoApp()
			a.askToCloseCollection("collection-1")
		}},
		{"settings-dark", design.Dark, func(a *app) {
			*a = *settingsDemo()
		}},
		{"settings-accent", design.Light, func(a *app) {
			*a = *settingsDemo()
			a.accent = "Blue"
		}},
	}
	for _, shot := range shots {
		// Each shot gets a store of its own: the demo collection lives in
		// memory, and what a shot writes stays in its directory instead of
		// being read back by the next one.
		shotStore = filepath.Join(stores, shot.name)
		if err := os.MkdirAll(shotStore, 0o755); err != nil {
			t.Fatal(err)
		}
		a := newApp()
		a.mode = shot.mode
		if shot.setup != nil {
			shot.setup(a)
			a.mode = shot.mode
		}
		tt := ui.NewTester(a.view, 1280, 800)
		tt.Frame()
		writeShot(t, filepath.Join(dir, shot.name+".png"), tt)
	}
}

func writeShot(t *testing.T, path string, tt *ui.Tester) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

package app

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"apigo/internal/design"
	"apigo/internal/store"
	"apigo/internal/widgets"

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
	a.store = store.New(shotStore)
	a.store.Collections = []collection{
		{
			ID: "collection-1", Name: "示例接口集",
			Root: folder{Name: store.RootFolderName, Requests: []savedRequest{
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
		{ID: "collection-2", Name: "空白示例集", Root: folder{Name: store.RootFolderName}},
	}
	a.store.Environments = []environment{
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
	a.store.Dir = exampleDir
	a.settingsOpen = true
	return a
}

// TestShots writes the window's screens as PNGs for a layout review, and
// for the screenshots CI keeps as artifacts. It runs only when APIGO_SHOTS
// names the directory to write to:
//
//	APIGO_SHOTS=docs/shots go test -run TestShots ./internal/app/
func TestShots(t *testing.T) {
	dir := os.Getenv("APIGO_SHOTS")
	if dir == "" {
		t.Skip("set APIGO_SHOTS to the directory to write screenshots to")
	}
	// A relative directory is taken from the module root: a test runs in its
	// package's directory, and "docs/shots" would land inside internal/app.
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(moduleRoot(t), dir)
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
		{"home-missing-collection", design.Light, func(a *app) {
			// A collection another program deleted: marked, not openable.
			a.store = demoApp().store
			a.missing = map[string]bool{"collection-2": true}
		}},
		{"missing-collection-confirm", design.Light, func(a *app) {
			a.store = demoApp().store
			a.missing = map[string]bool{"collection-2": true}
			a.askToForgetMissingCollection("collection-2")
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
		{"new-tab-menu", design.Light, func(a *app) {
			// The strip's + asks which kind of request the new tab holds.
			*a = *demoApp()
			open := a.menuFlag("new-tab")
			*open = true
		}},
		{"collection-request", design.Light, func(a *app) { *a = *demoApp() }},
		{"sidebar-collapsed", design.Light, func(a *app) {
			// The tree folded away, for the room the request needs.
			*a = *demoApp()
			a.sidebarCollapsed = true
		}},
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
			col := a.store.Collection("collection-1")
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
			a.showResponse(t, &response{
				Status: 200, StatusText: "OK", Duration: 184 * time.Millisecond, Size: 156,
				Body: "{\n  \"data\": [\n    {\"id\": 1, \"name\": \"Ada\"},\n    {\"id\": 2, \"name\": \"Grace\"}\n  ],\n  \"next\": null\n}",
			})
		}},
		{"response-image", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			body := shotPicture()
			t.url = "https://api.example.com/v1/pets/cover.png"
			a.showResponse(t, &response{
				Status: 200, StatusText: "OK", Duration: 96 * time.Millisecond, Size: len(body),
				Headers: http.Header{"Content-Type": {"image/png"}},
				Body:    string(body), Raw: body,
			})
		}},
		{"response-audio", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			body := shotSound()
			t.url = "https://api.example.com/v1/pets/bark.wav"
			a.showResponse(t, &response{
				Status: 200, StatusText: "OK", Duration: 120 * time.Millisecond, Size: len(body),
				Headers: http.Header{"Content-Type": {"audio/wav"}},
				Body:    string(body), Raw: body,
			})
		}},
		{"response-file", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			body := []byte("%PDF-1.7 a report of the pets")
			t.url = "https://api.example.com/v1/pets/report"
			a.showResponse(t, &response{
				Status: 200, StatusText: "OK", Duration: 240 * time.Millisecond, Size: len(body),
				Headers: http.Header{
					"Content-Type":        {"application/pdf"},
					"Content-Disposition": {`attachment; filename="宠物报告.pdf"`},
				},
				Body: string(body), Raw: body,
			})
		}},
		{"response-headers", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			t.url = "https://api.example.com/v1/pets"
			a.showResponse(t, &response{
				Status: 200, StatusText: "OK", Duration: 184 * time.Millisecond, Size: 156,
				Headers: http.Header{
					"Content-Type":  {"application/json"},
					"Cache-Control": {"max-age=2592000"},
					"Etag":          {"a2276f588ab3291eaf895fba5ad61f2c"},
					"Date":          {"Wed, 21 Oct 2026 08:32:19 GMT"},
				},
				Body: `{"ok":true}`, Raw: []byte(`{"ok":true}`),
			})
			t.respTab = respHeaders
		}},
		{"response-request-headers", design.Light, func(a *app) {
			*a = *demoApp()
			t := a.current()
			t.url = "https://api.example.com/v1/pets"
			a.showResponse(t, &response{
				Status: 200, StatusText: "OK", Duration: 184 * time.Millisecond, Size: 156,
				Headers: http.Header{"Content-Type": {"application/json"}},
				Body:    `{"ok":true}`, Raw: []byte(`{"ok":true}`),
			})
			t.sentHeaders = http.Header{
				"Accept":        {"application/json"},
				"Authorization": {"Bearer dev-token"},
				"User-Agent":    {"apigo/0.1.0"},
			}
			t.respTab = respRequestHeaders
		}},
		{"websocket-connected", design.Light, func(a *app) {
			*a = *demoApp()
			socketShot(a)
			t := a.current()
			t.url = "wss://echo-websocket.hoppscotch.io"
			t.bodyType = "json"
			t.body.Set("{\n  \"ping\": true\n}")
			t.socketLog = []wsEvent{
				{kind: socketNote, at: shotTime(0), text: "正在连接到 wss://echo-websocket.hoppscotch.io"},
				{kind: socketOpen, at: shotTime(1), text: "已连接 wss://echo-websocket.hoppscotch.io"},
				{kind: socketSent, at: shotTime(2), text: `{"ping": true}`, size: 15},
				{kind: socketGot, at: shotTime(3), text: `{"pong": true}`, size: 15},
			}
		}},
		{"websocket-message-open", design.Light, func(a *app) {
			*a = *demoApp()
			socketShot(a)
			t := a.current()
			t.url = "wss://echo-websocket.hoppscotch.io"
			t.bodyType = "json"
			t.socketLog = []wsEvent{
				{kind: socketOpen, at: shotTime(1), text: "已连接 wss://echo-websocket.hoppscotch.io"},
				{kind: socketGot, at: shotTime(3), size: 33,
					text: "{\n  \"type\": 9,\n  \"body\": {\"multi\": false}\n}"},
			}
			a.toggleLogLine(t, 1)
		}},
		{"websocket-format-menu", design.Light, func(a *app) {
			*a = *demoApp()
			socketShot(a)
			t := a.current()
			t.url = "wss://echo-websocket.hoppscotch.io"
			t.bodyType = "xml"
			*a.menuFlag("socket-format") = true
		}},
		{"websocket-protocol", design.Light, func(a *app) {
			*a = *demoApp()
			socketShot(a)
			t := a.current()
			t.url = "wss://echo-websocket.hoppscotch.io"
			t.protocol = "chat"
			t.headers.Rows = []widgets.Row{
				{Enabled: true, Key: "Authorization", Value: "Bearer dev-token"},
				{Enabled: true},
			}
			t.wsTab = 1
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

// shotPicture is a picture a shot shows as a response: a 320×180 gradient, so
// the pane's fitting and the size line have something to show.
func shotPicture() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 320, 180))
	for y := 0; y < 180; y++ {
		for x := 0; x < 320; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(46 + x/4), G: uint8(160 - y/4), B: uint8(106 + y/6), A: 0xff,
			})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// shotSound is a sound a shot shows as a response: a short tone, so the player
// has a length to draw.
func shotSound() []byte {
	const rate, frames = 48000, 48000 * 3 / 2 // a second and a half
	data := make([]byte, frames*2)
	for i := 0; i < frames; i++ {
		sample := int16(6000 * math.Sin(2*math.Pi*440*float64(i)/rate))
		binary.LittleEndian.PutUint16(data[i*2:], uint16(sample))
	}
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+len(data)))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], rate)
	binary.LittleEndian.PutUint32(header[28:], rate*2)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(len(data)))
	return append(header, data...)
}

// moduleRoot is the directory holding go.mod, which a relative APIGO_SHOTS is
// taken from.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %q", dir)
		}
		dir = parent
	}
}

// socketShot replaces the open request with a WebSocket one, so a shot shows the
// socket pane rather than the request panes.
func socketShot(a *app) {
	if err := a.addRequest(kindWebSocket, ""); err != nil {
		return
	}
	a.current().title = "实时对话"
}

// shotTime is a moment of a shot's log, at fixed offsets from a fixed time.
func shotTime(seconds int) time.Time {
	return time.Date(2026, 10, 21, 8, 32, seconds, 0, time.UTC)
}

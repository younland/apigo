package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apigo/internal/design"
	"apigo/widgets"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// The view runs without a window in tests, which click and type as a user
// would.
func TestHomeListsCollections(t *testing.T) {
	a := newApp()
	a.store = newStore(t.TempDir())
	a.store.collections = []collection{{ID: "Demo", Name: "Demo"}}
	tt := ui.NewTester(a.view, 1280, 800)

	if !tt.HasText("Demo") || !tt.HasText("HTTP") || !tt.HasText("0 个接口") {
		t.Errorf("texts %q, want the collection row of the home window", tt.Texts())
	}
	if !tt.HasText("导入集合") || !tt.HasText("新建集合") || tt.HasText("新建项目") {
		t.Errorf("texts %q, want 导入集合 and 新建集合 as the header actions", tt.Texts())
	}
	if err := tt.Click("打开 Demo"); err != nil {
		t.Fatal(err)
	}
	if a.active != "Demo" {
		t.Errorf("active %q after opening, want Demo", a.active)
	}
}

func TestHomeEmptyCreatesCollection(t *testing.T) {
	a := newApp()
	a.store = newStore(t.TempDir())
	tt := ui.NewTester(a.view, 1280, 800)

	if !tt.HasText("还没有集合") {
		t.Fatalf("texts %q, want the empty home window", tt.Texts())
	}
	if err := tt.Click("新建集合"); err != nil {
		t.Fatal(err)
	}
	if !a.collectionOpen || !tt.HasText("标签") {
		t.Fatalf("dialog open %v texts %q, want the new-collection dialog", a.collectionOpen, tt.Texts())
	}
	if len(a.store.collections) != 0 {
		t.Fatal("collection created before the dialog was saved")
	}
	if err := tt.Click("集合名称"); err != nil {
		t.Fatal(err)
	}
	tt.Type("订单服务")
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if len(a.store.collections) != 1 || a.store.collections[0].Name != "订单服务" {
		t.Fatalf("collections %+v, want 订单服务", a.store.collections)
	}
	if a.active == "" {
		t.Fatal("no collection open after saving")
	}
	if envs := a.store.environmentsFor(a.active); len(envs) != 3 {
		t.Errorf("environments %+v, want the three defaults", envs)
	}
}

func TestNewCollectionDialogCancels(t *testing.T) {
	a := newApp()
	a.store = newStore(t.TempDir())
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("新建集合"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	if a.collectionOpen {
		t.Error("dialog open after 取消")
	}
	if len(a.store.collections) != 0 {
		t.Errorf("collections %+v, want none created", a.store.collections)
	}
}

func TestLayoutRegions(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	top, ok := tt.Find("主窗口")
	if !ok || top.Y >= design.TabStripHeight {
		t.Errorf("home button at %+v, want it inside the %v DIP collection bar", top, design.TabStripHeight)
	}
	tabs, ok := tt.Find("标签页 Untitled")
	if !ok || tabs.Y >= 2*design.TabStripHeight {
		t.Errorf("request tab at %+v, want it inside the second strip", tabs)
	}
	panel, ok := tt.Find("接口管理")
	if !ok || panel.X >= design.SidebarWidth {
		t.Errorf("requests panel at %+v, want it inside the %v DIP panel", panel, design.SidebarWidth)
	}
	empty, ok := tt.Find("发送请求后，响应会显示在这里")
	if !ok || empty.Y <= 2*design.TabStripHeight {
		t.Errorf("response empty state at %+v, want it under the strips", empty)
	}
}

func TestRequestTabs(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if !tt.HasText("添加行") {
		t.Fatalf("texts %q, want the Params key/value list first", tt.Texts())
	}
	if err := tt.Click("Body"); err != nil {
		t.Fatal(err)
	}
	if a.current().bodyType != "json" || !tt.HasText("格式化") {
		t.Errorf("body type %q, texts %q; want the JSON editor with its format action",
			a.current().bodyType, tt.Texts())
	}
	if err := tt.Click("Authorization"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("该面板尚未实现") {
		t.Errorf("texts %q after selecting Authorization, want its placeholder", tt.Texts())
	}
}

func TestMethodMenuSelectsMethod(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("请求方法"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("PATCH"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().method; got != "PATCH" {
		t.Errorf("method %q after choosing PATCH, want PATCH", got)
	}
}

func TestTabsKeepTheirOwnRequests(t *testing.T) {
	a := testApp(t)
	ws := a.space()
	tt := ui.NewTester(a.view, 1280, 800)

	a.current().url = "https://one.example"
	if err := tt.Click("新建标签页"); err != nil {
		t.Fatal(err)
	}
	if len(ws.tabs) != 2 || ws.active != 1 {
		t.Fatalf("tabs %d active %d after the new tab, want 2 tabs with the second active",
			len(ws.tabs), ws.active)
	}
	if got := a.current().url; got != "" {
		t.Errorf("new tab URL %q, want an empty request", got)
	}
	a.current().url = "https://two.example"

	if err := tt.Click("标签页 Untitled"); err != nil {
		t.Fatal(err)
	}
	if ws.active != 0 {
		t.Fatalf("active %d after clicking the first tab, want 0", ws.active)
	}
	if got := ws.tabs[0].url; got != "https://one.example" {
		t.Errorf("first tab URL %q, want its own request", got)
	}
	if got := ws.tabs[1].url; got != "https://two.example" {
		t.Errorf("second tab URL %q, want its own request", got)
	}
}

func TestCloseTabKeepsOneTab(t *testing.T) {
	a := testApp(t)
	ws := a.space()
	a.addTab(ws)
	a.closeTab(ws, 1)
	if len(ws.tabs) != 1 || ws.active != 0 {
		t.Fatalf("tabs %d active %d after closing one of two, want one tab", len(ws.tabs), ws.active)
	}
	a.closeTab(ws, 0)
	if len(ws.tabs) != 1 {
		t.Fatalf("tabs %d after closing the last, want a fresh empty tab", len(ws.tabs))
	}
}

func TestTabContextMenuDuplicateAndRename(t *testing.T) {
	a := testApp(t)
	ws := a.space()
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.RightClick("标签页 Untitled"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("复制当前标签页"); err != nil {
		t.Fatal(err)
	}
	if len(ws.tabs) != 2 || ws.active != 1 {
		t.Fatalf("tabs %d active %d after duplicating, want the copy active",
			len(ws.tabs), ws.active)
	}

	if err := tt.RightClick("标签页 Untitled"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("重命名请求"); err != nil {
		t.Fatal(err)
	}
	if !ws.tabs[0].renaming {
		t.Fatal("first tab not renaming after choosing 重命名请求")
	}
}

func TestTabRenameThroughContextMenu(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.RightClick("标签页 Untitled"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("重命名请求"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Pets")
	if err := tt.Click("完成重命名"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().title; got != "Pets" {
		t.Errorf("title %q after renaming, want Pets", got)
	}
	if a.current().renaming {
		t.Error("tab still renaming after the check button")
	}

	if err := tt.RightClick("标签页 Pets"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("重命名请求"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if got := a.current().title; got != "Pets" || a.current().renaming {
		t.Errorf("title %q renaming %v after Escape, want the old title", got, a.current().renaming)
	}
}

func TestNewRequestAddsToCollection(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("WebSocket"); err != nil {
		t.Fatal(err)
	}
	col := a.store.collections[0]
	if len(col.Root.Requests) != 2 || col.Root.Requests[1].Kind != kindWebSocket {
		t.Fatalf("requests %+v, want a WebSocket request saved", col.Root.Requests)
	}
	if got := a.current().kind; got != kindWebSocket {
		t.Errorf("tab kind %q, want websocket", got)
	}
	if !tt.HasText("WebSocket 请求尚未实现") {
		t.Errorf("texts %q, want the placeholder of the unimplemented kind", tt.Texts())
	}
}

func TestCollectionRequestOpensTab(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("请求 Pets"); err != nil {
		t.Fatal(err)
	}
	cur := a.current()
	if cur.url != "https://api.example.com/pets" || cur.method != "POST" {
		t.Errorf("tab %q %q, want the saved request", cur.method, cur.url)
	}
	if cur.collectionID != "Demo" {
		t.Errorf("collection %q, want Demo", cur.collectionID)
	}
}

func TestSaveCurrentRequestToCollection(t *testing.T) {
	a := testApp(t)
	a.current().title, a.current().url = "新建宠物", "https://api.example.com/pets/new"

	if err := a.saveCurrentToCollection("Demo"); err != nil {
		t.Fatal(err)
	}
	loaded := newStore(a.store.dir)
	if err := loaded.load(); err != nil {
		t.Fatal(err)
	}
	if len(loaded.collections) != 1 || len(loaded.collections[0].Root.Requests) != 2 {
		t.Fatalf("collections %+v, want the new request saved", loaded.collections)
	}
	saved := loaded.collections[0].Root.Requests[1]
	if saved.Name != "新建宠物" || saved.URL != "https://api.example.com/pets/new" {
		t.Errorf("saved %+v, want the active request", saved)
	}
	if got := a.current().title; got != "新建宠物" {
		t.Errorf("tab title %q, want the name the interface file kept", got)
	}
}

func TestHistoryRecordedAfterSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	a := testApp(t)
	a.current().url = server.URL
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if len(a.store.history) != 1 {
		t.Fatalf("history %+v, want one entry", a.store.history)
	}
	if got := a.store.history[0].Status; got != http.StatusOK {
		t.Errorf("status %d, want 200", got)
	}
	loaded := newStore(a.store.dir)
	if err := loaded.load(); err != nil {
		t.Fatal(err)
	}
	if len(loaded.history) != 1 || loaded.history[0].URL != server.URL {
		t.Errorf("history %+v, want the entry on disk", loaded.history)
	}
}

func TestSendRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want the enabled query parameter", got)
		}
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	a := testApp(t)
	a.current().url = server.URL
	a.current().params.Rows = []widgets.Row{{Enabled: true, Key: "limit", Value: "10"}}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	res := a.current().resp
	if res == nil {
		t.Fatal("no response recorded after sending")
	}
	if res.status != http.StatusCreated || res.body != `{"ok":true}` {
		t.Errorf("response %d %q, want 201 and the JSON body", res.status, res.body)
	}
	if !tt.HasText("201 Created") {
		t.Errorf("texts %q, want the status line in the meta bar", tt.Texts())
	}
}

func TestSendJSONBody(t *testing.T) {
	type got struct {
		method, contentType, body string
	}
	received := make(chan got, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- got{r.Method, r.Header.Get("Content-Type"), string(body)}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	a := testApp(t)
	a.current().url = server.URL
	a.current().method = "POST"
	a.current().body.Set(`{"name":"Ada"}`)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	req := <-received
	if req.method != "POST" || req.contentType != "application/json" || req.body != `{"name":"Ada"}` {
		t.Errorf("received %+v, want the JSON body as written", req)
	}
}

func TestSendRejectsInvalidJSONBody(t *testing.T) {
	a := testApp(t)
	a.current().url = "http://example.com"
	a.current().method = "POST"
	a.current().body.Set(`{"name": }`)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if res := a.current().resp; res != nil || a.current().loading {
		t.Errorf("response %+v loading %v, want the invalid body refused", res, a.current().loading)
	}
}

func TestSendResolvesEnvironmentVariables(t *testing.T) {
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := testApp(t)
	a.store.environments[0].Vars = []savedRow{{Enabled: true, Key: "BaseURL", Value: server.URL}}
	a.current().url = "{{BaseURL}}/pets"
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if path := <-received; path != "/pets" {
		t.Errorf("path %q, want the resolved URL", path)
	}
	if res := a.current().resp; res == nil || res.status != http.StatusOK {
		t.Errorf("response %+v, want 200", res)
	}
}

func TestSendRefusesUnresolvedVariables(t *testing.T) {
	a := testApp(t)
	a.current().url = "https://api.example.com/{{missing}}"
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if res := a.current().resp; res != nil || a.current().loading {
		t.Errorf("response %+v loading %v, want the send refused", res, a.current().loading)
	}
}

func TestEnvironmentModal(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("环境：本地"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("管理环境"); err != nil {
		t.Fatal(err)
	}
	if !a.envOpen || !tt.HasText("环境变量") {
		t.Fatalf("env open %v texts %q, want the environment manager", a.envOpen, tt.Texts())
	}
	if err := tt.Click("新建环境"); err != nil {
		t.Fatal(err)
	}
	if envs := a.collectionEnvironments(); len(envs) != 2 {
		t.Fatalf("environments %+v, want the new one", envs)
	}
	if err := tt.Click("删除环境 新环境"); err != nil {
		t.Fatal(err)
	}
	if envs := a.collectionEnvironments(); len(envs) != 1 {
		t.Errorf("environments %+v, want the new one deleted", envs)
	}
}

func TestSettingsModal(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if !a.settingsOpen || !tt.HasText("数据目录") || !tt.HasText("外观") {
		t.Fatalf("settings open %v texts %q, want the settings dialog", a.settingsOpen, tt.Texts())
	}
	for _, mode := range []design.Mode{design.Light, design.Dark, design.Black} {
		if err := tt.Click(a.mode.String()); err != nil {
			t.Fatal(err)
		}
		if a.mode != mode {
			t.Fatalf("appearance %v after a click, want %v", a.mode, mode)
		}
	}
}

func TestResponseRenderedAsReadOnlyJSON(t *testing.T) {
	a := testApp(t)
	a.apply(a.current(), &response{
		status:     http.StatusOK,
		headers:    http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		body:       `{"data":[]}`,
		statusText: "OK",
	})
	edit := a.current().respEdit
	if !edit.ReadOnly {
		t.Error("response editor is editable")
	}
	if got := edit.Buffer.Language(); got != editor.LanguageFor("response.json") {
		t.Errorf("language %v, want JSON highlighting", got)
	}
	if got := edit.Buffer.String(); got != `{"data":[]}` {
		t.Errorf("buffer %q, want the response body as sent", got)
	}
}

func TestIndentJSON(t *testing.T) {
	got, err := indentJSON(`{"a":[1,2],"b":null}`)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"a\": [\n    1,\n    2\n  ],\n  \"b\": null\n}"
	if got != want {
		t.Errorf("indentJSON = %q, want %q", got, want)
	}
	if got, err := indentJSON("   "); got != "" || err != nil {
		t.Errorf("indentJSON of blank text = %q, %v; want empty and no error", got, err)
	}
	if _, err := indentJSON("{oops}"); err == nil {
		t.Error("no error for invalid JSON")
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		contentType string
		body        string
		want        bool
	}{
		{"application/json; charset=utf-8", `{"a":1}`, true},
		{"application/json", "not json", false},
		{"text/plain", `{"a":1}`, false},
		{"", `{"a":1}`, true},
		{"", "hello", false},
	}
	for _, test := range tests {
		if got := isJSON(test.contentType, test.body); got != test.want {
			t.Errorf("isJSON(%q, %q) = %v, want %v", test.contentType, test.body, got, test.want)
		}
	}
}

func TestHomeHeaderNewCollectionButton(t *testing.T) {
	a := newApp()
	a.store = newStore(t.TempDir())
	a.store.collections = []collection{{ID: "Demo", Name: "Demo"}}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("新建集合"); err != nil {
		t.Fatal(err)
	}
	if !a.collectionOpen {
		t.Fatal("dialog closed after the header's 新建集合")
	}
	if len(a.store.collections) != 1 {
		t.Errorf("collections %+v, want none created before saving", a.store.collections)
	}
}

func TestMoreButtonListsClosedCollections(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if _, ok := tt.Find("更多"); !ok {
		t.Fatal("更多 missing from the collection bar")
	}
	if err := tt.Click("更多"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("暂无最近关闭的集合") {
		t.Errorf("texts %q, want the empty hint of the closed list", tt.Texts())
	}
	if err := tt.Click("更多"); err != nil {
		t.Fatal(err)
	}

	if err := tt.Click("关闭集合 Demo"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("更多"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("重新打开 Demo") {
		t.Errorf("texts %q, want the closed collection listed", tt.Texts())
	}
	if err := tt.Click("重新打开 Demo"); err != nil {
		t.Fatal(err)
	}
	if a.active != "Demo" {
		t.Errorf("active %q after reopening, want collection-1", a.active)
	}
}

func TestCollectionTabContextMenu(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.RightClick("集合标签 Demo"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("关闭集合"); err != nil {
		t.Fatal(err)
	}
	if a.active != "" {
		t.Errorf("active %q after closing the only collection, want the home window", a.active)
	}
}

func TestHomeEmptyGuidesWithOwnActions(t *testing.T) {
	a := newApp()
	a.store = newStore(t.TempDir())
	tt := ui.NewTester(a.view, 1280, 800)

	for _, want := range []string{"还没有集合", "新建集合", "从集合文件导入", "BaseURL"} {
		if !tt.HasText(want) {
			t.Errorf("texts %q, want %q in the empty home window", tt.Texts(), want)
		}
	}
	if _, ok := tt.Find("导入集合"); ok {
		t.Error("header 导入集合 shown in the empty window, want its own actions only")
	}
}

func TestFolderTreeAddsAndNavigates(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if _, ok := tt.Find("文件夹 根目录"); !ok {
		t.Fatal("根目录 missing from the interfaces tree")
	}
	if err := tt.Click("新建文件夹"); err != nil {
		t.Fatal(err)
	}
	col := a.store.collections[0]
	if len(col.Root.Folders) != 1 || col.Root.Folders[0].Name != "分组 1" {
		t.Fatalf("folders %+v, want the new one in the tree", col.Root.Folders)
	}

	if err := tt.Click("在分组 1里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("WebSocket"); err != nil {
		t.Fatal(err)
	}
	col = a.store.collections[0]
	folder := col.Root.Folders[0]
	if len(folder.Requests) != 1 || folder.Requests[0].Kind != kindWebSocket {
		t.Fatalf("folder requests %+v, want the WebSocket request inside the folder", folder.Requests)
	}
	if folder.Requests[0].Path != "分组 1/Untitled" {
		t.Errorf("request path %q, want it under the folder", folder.Requests[0].Path)
	}
	if got := a.current().kind; got != kindWebSocket {
		t.Errorf("open tab kind %q, want the new folder request", got)
	}

	if err := tt.Click("文件夹 分组 1"); err != nil {
		t.Fatal(err)
	}
	if !a.foldersClosed[folder.Path] {
		t.Error("folder not collapsed after clicking its row")
	}
	if tt.HasText("请求 Untitled") {
		t.Errorf("texts %q, want the folder's request hidden while collapsed", tt.Texts())
	}
	if err := tt.Click("文件夹 根目录"); err != nil {
		t.Fatal(err)
	}
	if !a.foldersClosed[""] {
		t.Error("根目录 not collapsed after clicking its row")
	}
	if tt.HasText("文件夹 分组 1") {
		t.Errorf("texts %q, want the tree hidden while 根目录 is collapsed", tt.Texts())
	}
}

// TestNewInterfaceWritesFile checks that making an interface writes its own
// file, named after it, instead of rewriting one collection file.
func TestNewInterfaceWritesFile(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("WebSocket"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.store.collectionDir("Demo"), "Untitled.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("interface file %s missing: %v", path, err)
	}
	if err := tt.Click("新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("MQTT"); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(a.store.collectionDir("Demo"), "Untitled 2.yaml")
	if _, err := os.Stat(second); err != nil {
		t.Errorf("second interface did not get a file of its own: %v", err)
	}
}

// TestTabSwitchWritesInterfaceFile checks that an edited request keeps its
// edit in its interface file when the workspace moves to another tab.
func TestTabSwitchWritesInterfaceFile(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("请求 Pets"); err != nil {
		t.Fatal(err)
	}
	a.current().url = "https://api.example.com/pets/2"
	if err := tt.Click("新建标签页"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.store.collectionDir("Demo"), "Pets.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("interface file %s missing: %v", path, err)
	}
	if !strings.Contains(string(data), "url: https://api.example.com/pets/2") {
		t.Errorf("interface file %q, want the edited URL", data)
	}
	if got := a.store.collections[0].Root.Requests[0].URL; got != "https://api.example.com/pets/2" {
		t.Errorf("stored URL %q, want the edit", got)
	}
}

// TestCloseTabWritesInterfaceFile checks that closing an edited request
// keeps the edit in its interface file.
func TestCloseTabWritesInterfaceFile(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("请求 Pets"); err != nil {
		t.Fatal(err)
	}
	a.current().params.Rows = []widgets.Row{{Enabled: true, Key: "limit", Value: "10"}}
	if err := tt.Click("关闭标签页"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(a.store.collectionDir("Demo"), "Pets.yaml"))
	if err != nil {
		t.Fatalf("interface file missing after closing the tab: %v", err)
	}
	if !strings.Contains(string(data), "key: limit") {
		t.Errorf("interface file %q, want the edited parameter", data)
	}
}

// TestFailedInterfaceSaveIsReported keeps a failed write from passing
// silently: the window says so.
func TestFailedInterfaceSaveIsReported(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("请求 Pets"); err != nil {
		t.Fatal(err)
	}
	// The interface the tab came from is gone, as editing the tree by hand
	// can leave it.
	a.store.collections[0].Root.Requests = nil
	if err := tt.Click("新建标签页"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("接口保存失败：没有接口 \"Pets\"") {
		t.Errorf("texts %q, want the failed write reported", tt.Texts())
	}
}

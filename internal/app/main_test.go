package app

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"apigo/internal/design"
	"apigo/internal/store"
	"apigo/internal/widgets"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// The view runs without a window in tests, which click and type as a user
// would.
func TestHomeListsCollections(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	a.store.Collections = []collection{{ID: "Demo", Name: "Demo"}}
	tt := ui.NewTester(a.view, 1280, 800)

	if !tt.HasText("Demo") || !tt.HasText("0 个接口") {
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
	a.store = store.New(t.TempDir())
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
	if len(a.store.Collections) != 0 {
		t.Fatal("collection created before the dialog was saved")
	}
	if err := tt.Click("集合名称"); err != nil {
		t.Fatal(err)
	}
	tt.Type("订单服务")
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if len(a.store.Collections) != 1 || a.store.Collections[0].Name != "订单服务" {
		t.Fatalf("collections %+v, want 订单服务", a.store.Collections)
	}
	if a.active == "" {
		t.Fatal("no collection open after saving")
	}
	if envs := a.store.EnvironmentsFor(a.active); len(envs) != 3 {
		t.Errorf("environments %+v, want the three defaults", envs)
	}
}

func TestNewCollectionDialogCancels(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
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
	if len(a.store.Collections) != 0 {
		t.Errorf("collections %+v, want none created", a.store.Collections)
	}
}

func TestLayoutRegions(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	top, ok := tt.Find("主窗口")
	if !ok || top.Y >= design.TabStripHeight {
		t.Errorf("home button at %+v, want it inside the %v DIP collection bar", top, design.TabStripHeight)
	}
	tabs, ok := tt.Find("标签页 Pets")
	if !ok || tabs.Y >= 2*design.TabStripHeight {
		t.Errorf("request tab at %+v, want it inside the second strip", tabs)
	}
	panel, ok := tt.Find("接口面板")
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

	if !tt.HasText("参数名第 1 行") {
		t.Fatalf("texts %q, want the Params key/value list first", tt.Texts())
	}
	// The panes still to come are not shown yet.
	for _, hidden := range []string{"Authorization", "Pre-request", "Scripts", "Tests"} {
		if _, ok := tt.Find(hidden); ok {
			t.Errorf("%q is in the tab strip, want the panes with nothing behind them hidden", hidden)
		}
	}
	if err := tt.Click("请求体"); err != nil {
		t.Fatal(err)
	}
	if a.current().bodyType != "json" || !tt.HasText("美化请求体") {
		t.Errorf("body type %q, texts %q; want the JSON editor with its prettify action",
			a.current().bodyType, tt.Texts())
	}
	if !tt.HasText("application/json") {
		t.Errorf("texts %q, want the Content-Type of the chosen body format", tt.Texts())
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
	newTabThroughTheStrip(t, tt, "REST")
	if len(ws.tabs) != 2 || ws.active != 1 {
		t.Fatalf("tabs %d active %d after the new tab, want 2 tabs with the second active",
			len(ws.tabs), ws.active)
	}
	if got := a.current().url; got != "" {
		t.Errorf("new tab URL %q, want an empty request", got)
	}
	a.current().url = "https://two.example"

	if err := tt.Click("标签页 Pets"); err != nil {
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

// TestCloseTabLeavesNoTabAtTheEnd checks closing the last tab leaves the
// workspace with no tab, so the pane shows what to do again.
func TestCloseTabLeavesNoTabAtTheEnd(t *testing.T) {
	a := testApp(t)
	ws := a.space()
	a.addTab(ws)
	a.closeTab(ws, 1)
	if len(ws.tabs) != 1 || ws.active != 0 {
		t.Fatalf("tabs %d active %d after closing one of two, want one tab", len(ws.tabs), ws.active)
	}
	a.closeTab(ws, 0)
	if !ws.empty() || ws.active != 0 {
		t.Fatalf("tabs %d active %d after closing the last, want no tab left",
			len(ws.tabs), ws.active)
	}
}

// TestTabContextMenuDuplicateAndRename checks the request tab answers a
// right click with the app's own menu, which duplicates and renames.
func TestTabContextMenuDuplicateAndRename(t *testing.T) {
	a := testApp(t)
	ws := a.space()
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.RightClick("标签页 Pets"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Menu() != nil {
		t.Errorf("platform menu %q, want the app's own popover menu", tt.Menu())
	}
	if err := tt.Click("复制当前标签页"); err != nil {
		t.Fatal(err)
	}
	if len(ws.tabs) != 2 || ws.active != 1 {
		t.Fatalf("tabs %d active %d after duplicating, want the copy active",
			len(ws.tabs), ws.active)
	}

	if err := tt.RightClick("标签页 Pets"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("重命名请求"); err != nil {
		t.Fatal(err)
	}
	if !ws.tabs[0].renaming {
		t.Fatal("first tab not renaming after choosing 重命名请求")
	}
}

func TestTabRenameThroughContextMenu(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.RightClick("标签页 Pets"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("重命名请求"); err != nil {
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
	tt.Frame()
	if err := tt.Click("重命名请求"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if got := a.current().title; got != "Pets" || a.current().renaming {
		t.Errorf("title %q renaming %v after Escape, want the old title", got, a.current().renaming)
	}
}

// TestNewRequestAddsToCollection checks a request made with the tree's + opens
// as a draft: nothing is written until it is saved with Ctrl+S, which asks
// where in the collection it goes.
func TestNewRequestAddsToCollection(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("WebSocket"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().kind; got != kindWebSocket {
		t.Errorf("tab kind %q, want websocket", got)
	}
	// A socket opens its own pane: the address, the panes, and the log.
	for _, want := range []string{"通讯", "协议", "信息", "日志"} {
		if !tt.HasText(want) {
			t.Errorf("texts %q, want %q in the socket pane", tt.Texts(), want)
		}
	}
	if _, ok := tt.Find("连接"); !ok {
		t.Error("the socket pane has no connect button")
	}
	if !a.current().unsaved() {
		t.Error("a new request is on disk already, want it a draft until Ctrl+S")
	}
	if col := a.store.Collections[0]; len(col.Root.Requests) != 1 {
		t.Fatalf("requests %+v, want the draft left out of the tree", col.Root.Requests)
	}

	// Ctrl+S asks where it goes and what it is called, and writes it there.
	tt.Key(ui.Cmd, ui.KeyS)
	if !tt.HasText("保存接口") || !tt.HasText("接口名称") {
		t.Fatalf("texts %q, want the question of where to save and its name", tt.Texts())
	}
	if err := tt.Click("存放到 根目录"); err != nil {
		t.Fatal(err)
	}
	// The name of a request that has no file yet can be set before it is saved.
	if err := tt.Click("接口名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("订阅设备")
	if err := tt.Click("确认保存"); err != nil {
		t.Fatal(err)
	}
	col := a.store.Collections[0]
	if len(col.Root.Requests) != 2 || col.Root.Requests[1].Kind != kindWebSocket {
		t.Fatalf("requests %+v, want the draft saved", col.Root.Requests)
	}
	if got := col.Root.Requests[1].Name; got != "订阅设备" {
		t.Errorf("saved as %q, want the name typed before saving", got)
	}
	if a.current().unsaved() {
		t.Error("the tab is unsaved after Ctrl+S, want what it holds on disk")
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
	loaded := store.New(a.store.Dir)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Collections) != 1 || len(loaded.Collections[0].Root.Requests) != 2 {
		t.Fatalf("collections %+v, want the new request saved", loaded.Collections)
	}
	saved := loaded.Collections[0].Root.Requests[1]
	if saved.Name != "新建宠物" || saved.URL != "https://api.example.com/pets/new" {
		t.Errorf("saved %+v, want the active request", saved)
	}
	if got := a.current().title; got != "新建宠物" {
		t.Errorf("tab title %q, want the name the interface file kept", got)
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
	a.current().params.Rows = []widgets.Param{{Enabled: true, Key: "limit", Value: "10"}}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	res := a.current().resp
	if res == nil {
		t.Fatal("no response recorded after sending")
	}
	if res.Status != http.StatusCreated || res.Body != `{"ok":true}` {
		t.Errorf("response %d %q, want 201 and the JSON body", res.Status, res.Body)
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
	a.store.Environments[0].Vars = []savedRow{{Enabled: true, Key: store.BaseURLVariable, Value: server.URL}}
	a.current().url = "{{baseurl}}/pets"
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if path := <-received; path != "/pets" {
		t.Errorf("path %q, want the resolved URL", path)
	}
	if res := a.current().resp; res == nil || res.Status != http.StatusOK {
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

// TestWindowThemeFollowsPalette keeps the window's controls in the app's own
// colors: a window that follows the desktop draws with our palette too, not
// with MyGo's default blue.
func TestWindowThemeFollowsPalette(t *testing.T) {
	a := testApp(t)
	a.mode = design.System
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()

	if a.theme == nil {
		t.Fatal("no theme applied while following the desktop")
	}
	want := design.PaletteOf(a.resolvedMode())
	if a.theme.Accent != want.Accent {
		t.Errorf("theme accent %v, want the palette's %v", a.theme.Accent, want.Accent)
	}
	if a.theme.Border != want.Border || a.theme.Background != want.Background {
		t.Errorf("theme borders %v background %v, want %v and %v",
			a.theme.Border, a.theme.Background, want.Border, want.Background)
	}
	if a.theme.Font != design.FontFamily {
		t.Errorf("theme family %q, want %q", a.theme.Font, design.FontFamily)
	}
}

// TestSettingsModal checks the settings offer the language, the background
// and the accent, and that picking them takes effect in the window's theme.
func TestSettingsModal(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if !a.settingsOpen {
		t.Fatal("settings closed after the sun button")
	}
	for _, want := range []string{"通用", "语言", "简体中文", "主题", "背景", "强调色",
		"系统（亮色）", "数据目录", "更换目录", "打开目录", "版本",
		"响应体上限（MB）", "超出上限会被截断，默认 100 MB",
		"请求超时（秒）", "到点即中止，默认 1800 秒（30 分钟）"} {
		if !tt.HasText(want) {
			t.Errorf("texts %q, want %q in the settings", tt.Texts(), want)
		}
	}
	if err := tt.Click("界面语言 简体中文"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("简体中文"); err != nil {
		t.Fatal(err)
	}
	if a.language != "简体中文" {
		t.Errorf("language %q, want 简体中文", a.language)
	}

	for _, mode := range []design.Mode{design.Dark, design.Black, design.Light} {
		if err := tt.Click(mode.String()); err != nil {
			t.Fatal(err)
		}
		if a.mode != mode {
			t.Fatalf("appearance %v after a click, want %v", a.mode, mode)
		}
	}
	if err := tt.Click("跟随系统"); err != nil {
		t.Fatal(err)
	}
	if a.mode != design.System {
		t.Errorf("appearance %v, want 跟随系统", a.mode)
	}

	if err := tt.Click("强调色 Blue"); err != nil {
		t.Fatal(err)
	}
	want := design.AccentByName("Blue")
	if a.accent != want.Name {
		t.Errorf("accent %q, want %q", a.accent, want.Name)
	}
	if got := a.palette().Accent; got != want.Base {
		t.Errorf("palette accent %v, want %v", got, want.Base)
	}
	tt.Frame()
	if a.theme == nil || a.theme.Accent != want.Base {
		t.Errorf("theme %v, want the accent %v", a.theme, want.Base)
	}
}

// TestChooseDataDir checks 更换目录 moves the app to the collections of the
// directory picked, and remembers it for the next run.
func TestChooseDataDir(t *testing.T) {
	a := testApp(t)
	base, other := t.TempDir(), t.TempDir()
	otherStore := store.New(other)
	if _, err := otherStore.NewCollection("另一个集合"); err != nil {
		t.Fatal(err)
	}
	a.baseDir = base
	a.chooseDir = func() (string, error) { return other, nil }
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("更换目录"); err != nil {
		t.Fatal(err)
	}
	if a.store.Dir != other {
		t.Fatalf("data directory %q, want the chosen %q", a.store.Dir, other)
	}
	tt.Frame()
	if !tt.HasText(other) {
		t.Errorf("texts %q, want the new data directory shown", tt.Texts())
	}
	if collections := a.store.Collections; len(collections) != 1 || collections[0].Name != "另一个集合" {
		t.Errorf("collections %+v, want the ones of the chosen directory", collections)
	}
	if a.active != "" || len(a.open) != 0 || len(a.spaces) != 0 {
		t.Errorf("active %q open %v spaces %d, want the window reset", a.active, a.open, len(a.spaces))
	}
	if got := store.DirAt(base); got != other {
		t.Errorf("remembered %q, want the chosen %q", got, other)
	}
}

// TestChooseDataDirCancelled keeps the data directory when the picker is
// cancelled or fails.
func TestChooseDataDirCancelled(t *testing.T) {
	a := testApp(t)
	before := a.store.Dir
	a.baseDir = t.TempDir()
	tt := ui.NewTester(a.view, 1280, 800)

	a.chooseDir = func() (string, error) { return "", nil }
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("更换目录"); err != nil {
		t.Fatal(err)
	}
	if a.store.Dir != before {
		t.Errorf("data directory %q after cancelling, want %q", a.store.Dir, before)
	}

	a.chooseDir = func() (string, error) { return "", errors.New("没有权限") }
	if err := tt.Click("更换目录"); err != nil {
		t.Fatal(err)
	}
	if a.store.Dir != before {
		t.Errorf("data directory %q after a failure, want %q", a.store.Dir, before)
	}
	tt.Frame()
	if !tt.HasText("选择目录失败：没有权限") {
		t.Errorf("texts %q, want the failed pick reported", tt.Texts())
	}
}

func TestResponseRenderedAsReadOnlyJSON(t *testing.T) {
	a := testApp(t)
	a.showResponse(a.current(), &response{
		Status:     http.StatusOK,
		Headers:    http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		Body:       `{"data":[]}`,
		StatusText: "OK",
	})
	edit := a.current().respEdit
	if !edit.ReadOnly {
		t.Error("response editor is editable")
	}
	if got := edit.Buffer.Language(); got != editor.LanguageFor("response.json") {
		t.Errorf("language %v, want JSON highlighting", got)
	}
	// A JSON body is shown indented, as the pane reads it as code.
	if got, want := edit.Buffer.String(), "{\n  \"data\": []\n}"; got != want {
		t.Errorf("buffer %q, want the response body indented", got)
	}
}

// TestResponseTextIsLeftAsSent checks a body that is not JSON is shown as the
// server sent it, without JSON highlighting.
func TestResponseTextIsLeftAsSent(t *testing.T) {
	a := testApp(t)
	a.showResponse(a.current(), &response{
		Status:  http.StatusOK,
		Headers: http.Header{"Content-Type": {"text/plain"}},
		Body:    "hello {not json}",
	})
	edit := a.current().respEdit
	if got := edit.Buffer.Language(); got != nil {
		t.Errorf("language %v, want none for a text body", got)
	}
	if got, want := edit.Buffer.String(), "hello {not json}"; got != want {
		t.Errorf("buffer %q, want the body as sent", got)
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
	a.store = store.New(t.TempDir())
	a.store.Collections = []collection{{ID: "Demo", Name: "Demo"}}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("新建集合"); err != nil {
		t.Fatal(err)
	}
	if !a.collectionOpen {
		t.Fatal("dialog closed after the header's 新建集合")
	}
	if len(a.store.Collections) != 1 {
		t.Errorf("collections %+v, want none created before saving", a.store.Collections)
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
	// Closing a collection asks first.
	if err := tt.Click("确认关闭集合"); err != nil {
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

// TestCollectionTabContextMenu checks the collection tab answers a right
// click with the app's own menu, not the platform's.
func TestCollectionTabContextMenu(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.RightClick("集合标签 Demo"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Menu() != nil {
		t.Errorf("platform menu %q, want the app's own popover menu", tt.Menu())
	}
	if !tt.HasText("关闭其他集合") {
		t.Fatalf("texts %q, want the collection's menu", tt.Texts())
	}
	if err := tt.Click("关闭集合"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("确认关闭集合"); err != nil {
		t.Fatal(err)
	}
	if a.active != "" {
		t.Errorf("active %q after closing the only collection, want the home window", a.active)
	}
}

// TestHomeCollectionMenu checks the home row's menu is the app's own, with
// icons, and that it opens the collection, its folder, and deletes it.
func TestHomeCollectionMenu(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	a.store.Collections = []collection{{ID: "Demo", Name: "Demo", Root: folder{Name: store.RootFolderName}}}
	var opened []string
	a.opener = func(path string) error {
		opened = append(opened, path)
		return nil
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("集合操作 Demo"); err != nil {
		t.Fatal(err)
	}
	if tt.Menu() != nil {
		t.Errorf("platform menu %q, want the app's own popover menu", tt.Menu())
	}
	for _, want := range []string{"打开文件夹", "打开集合", "删除集合"} {
		if !tt.HasText(want) {
			t.Errorf("texts %q, want %q in the row's menu", tt.Texts(), want)
		}
	}
	if folderItem, ok := tt.Find("打开文件夹"); !ok {
		t.Fatal("打开文件夹 missing from the row's menu")
	} else if collectionItem, ok := tt.Find("打开集合"); !ok {
		t.Fatal("打开集合 missing from the row's menu")
	} else if folderItem.Y >= collectionItem.Y {
		t.Errorf("打开文件夹 at y %.0f, want it above 打开集合 at y %.0f", folderItem.Y, collectionItem.Y)
	}
	if err := tt.Click("打开文件夹"); err != nil {
		t.Fatal(err)
	}
	if want := []string{a.store.CollectionDir("Demo")}; !slices.Equal(opened, want) {
		t.Errorf("opened %v, want the collection's directory %v", opened, want)
	}
	if a.active != "" {
		t.Errorf("active %q after 打开文件夹, want the home window", a.active)
	}
	if err := tt.Click("集合操作 Demo"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("打开集合"); err != nil {
		t.Fatal(err)
	}
	if a.active != "Demo" {
		t.Fatalf("active %q after 打开集合, want Demo", a.active)
	}
	if err := tt.Click("主窗口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("集合操作 Demo"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("删除集合"); err != nil {
		t.Fatal(err)
	}
	// Deleting asks first, saying that the folder stays.
	if !tt.HasText("仍保留在数据目录里") {
		t.Errorf("texts %q, want the warning of what a deletion keeps", tt.Texts())
	}
	if err := tt.Click("确认删除集合"); err != nil {
		t.Fatal(err)
	}
	if len(a.store.Collections) != 0 {
		t.Errorf("collections %+v, want the row's collection deleted", a.store.Collections)
	}
}

// TestRenameCollectionThroughMenu renames a collection from the home row's
// menu: the dialog comes with the current name, and the open workspace
// follows the new id.
func TestRenameCollectionThroughMenu(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("请求 Pets"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("主窗口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("集合操作 Demo"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("重命名集合") {
		t.Fatalf("texts %q, want 重命名集合 in the row's menu", tt.Texts())
	}
	if err := tt.Click("重命名集合"); err != nil {
		t.Fatal(err)
	}
	if !a.collectionOpen {
		t.Fatal("dialog closed after 重命名集合")
	}
	if a.collectionName != "Demo" {
		t.Errorf("dialog name %q, want the collection's own name", a.collectionName)
	}
	if !tt.HasText("重命名集合") || !tt.HasText("保存") {
		t.Errorf("texts %q, want the rename dialog", tt.Texts())
	}
	if err := tt.Click("集合名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("订单服务")
	if a.collectionName != "订单服务" {
		t.Fatalf("name %q after typing, want 订单服务", a.collectionName)
	}
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	col := a.store.Collections[0]
	if col.ID != "订单服务" || col.Name != "订单服务" {
		t.Fatalf("collection %+v, want it renamed", col)
	}
	if _, err := os.Stat(filepath.Join(a.store.Dir, "collections", "订单服务")); err != nil {
		t.Errorf("renamed directory missing: %v", err)
	}
	if want := []string{"订单服务"}; !slices.Equal(a.open, want) {
		t.Errorf("open %v, want the new id in its place", a.open)
	}
	ws := a.spaces["订单服务"]
	if ws == nil {
		t.Fatal("workspace not moved to the new id")
	}
	if _, stale := a.spaces["Demo"]; stale {
		t.Error("workspace still keyed by the old id")
	}
	for _, reqTab := range ws.tabs {
		if reqTab.path == "" {
			continue // a tab that never came from a file belongs to no collection
		}
		if reqTab.collectionID != "订单服务" {
			t.Errorf("tab collection %q, want the new id", reqTab.collectionID)
		}
	}
	if err := tt.Click("打开 订单服务"); err != nil {
		t.Fatal(err)
	}
	if a.active != "订单服务" {
		t.Errorf("active %q after reopening, want the new id", a.active)
	}
	if !tt.HasText("订单服务") {
		t.Errorf("texts %q, want the renamed collection shown", tt.Texts())
	}
}

func TestHomeEmptyGuidesWithOwnActions(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	tt := ui.NewTester(a.view, 1280, 800)

	for _, want := range []string{"还没有集合", "新建集合", "导入文档", "baseurl"} {
		if !tt.HasText(want) {
			t.Errorf("texts %q, want %q in the empty home window", tt.Texts(), want)
		}
	}
	if _, ok := tt.Find("导入集合"); ok {
		t.Error("header 导入集合 shown in the empty window, want its own actions only")
	}
}

// TestRootRowHostsTreeTools checks the 根目录 row carries the kind filter,
// left of the add menu, and that the menu holds making a folder too. No
// title stands above the tree.
func TestRootRowHostsTreeTools(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if tt.HasText("接口管理") {
		t.Errorf("texts %q, want no panel title above the tree", tt.Texts())
	}
	root, ok := tt.Find("文件夹 根目录")
	if !ok {
		t.Fatal("根目录 missing from the tree")
	}
	filter, ok := tt.Find("筛选类型")
	if !ok {
		t.Fatal("筛选类型 missing from the tree")
	}
	add, ok := tt.Find("在根目录里新建接口")
	if !ok {
		t.Fatal("在根目录里新建接口 missing from the tree")
	}
	for _, control := range []ui.Rect{filter, add} {
		if control.Y < root.Y || control.Y >= root.Y+28 {
			t.Errorf("control at %+v, want it on the 根目录 row at %+v", control, root)
		}
	}
	if filter.X >= add.X {
		t.Errorf("x of 筛选 %.0f / 新建接口 %.0f, want the filter left of the add menu",
			filter.X, add.X)
	}
	if err := tt.Click("筛选类型"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("全部类型") {
		t.Errorf("texts %q, want the kind filter menu", tt.Texts())
	}
	if err := tt.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("在根目录里新建文件夹"); err != nil {
		t.Fatal(err)
	}
	addFolderNamed(t, tt, "分组 1")
	if folders := a.store.Collections[0].Root.Folders; len(folders) != 1 {
		t.Errorf("folders %+v, want the one the add menu made", folders)
	}
}

func TestFolderTreeAddsAndNavigates(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if _, ok := tt.Find("文件夹 根目录"); !ok {
		t.Fatal("根目录 missing from the interfaces tree")
	}
	if err := tt.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("在根目录里新建文件夹"); err != nil {
		t.Fatal(err)
	}
	addFolderNamed(t, tt, "分组 1")
	col := a.store.Collections[0]
	if len(col.Root.Folders) != 1 || col.Root.Folders[0].Name != "分组 1" {
		t.Fatalf("folders %+v, want the new one in the tree", col.Root.Folders)
	}

	if err := tt.Click("在分组 1里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("WebSocket"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().kind; got != kindWebSocket {
		t.Errorf("open tab kind %q, want the new folder request", got)
	}
	// The + stood on the folder, which is where Ctrl+S saves it.
	tt.Key(ui.Cmd, ui.KeyS)
	if err := tt.Click("确认保存"); err != nil {
		t.Fatal(err)
	}
	col = a.store.Collections[0]
	folder := col.Root.Folders[0]
	if len(folder.Requests) != 1 || folder.Requests[0].Kind != kindWebSocket {
		t.Fatalf("folder requests %+v, want the WebSocket request inside the folder", folder.Requests)
	}
	if folder.Requests[0].Path != "分组 1/Untitled" {
		t.Errorf("request path %q, want it under the folder", folder.Requests[0].Path)
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

// TestNewInterfaceWritesFile checks that saving an interface writes its own
// file, named after it, instead of rewriting one collection file.
func TestNewInterfaceWritesFile(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("WebSocket"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyS)
	if err := tt.Click("确认保存"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.store.CollectionDir("Demo"), "Untitled.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("interface file %s missing: %v", path, err)
	}
	if err := tt.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("MQTT"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyS)
	if err := tt.Click("确认保存"); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(a.store.CollectionDir("Demo"), "Untitled 2.yaml")
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
	newTabThroughTheStrip(t, tt, "REST")
	path := filepath.Join(a.store.CollectionDir("Demo"), "Pets.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("interface file %s missing: %v", path, err)
	}
	if !strings.Contains(string(data), "url: https://api.example.com/pets/2") {
		t.Errorf("interface file %q, want the edited URL", data)
	}
	if got := a.store.Collections[0].Root.Requests[0].URL; got != "https://api.example.com/pets/2" {
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
	a.current().params.Rows = []widgets.Param{{Enabled: true, Key: "limit", Value: "10"}}
	// The edit marks the tab unsaved, which widens it by the dot: a frame shows
	// where its buttons are now.
	tt.Frame()
	if err := tt.Click("关闭标签页 Pets"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(a.store.CollectionDir("Demo"), "Pets.yaml"))
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
	a.store.Collections[0].Root.Requests = nil
	newTabThroughTheStrip(t, tt, "REST")
	tt.Frame()
	if !tt.HasText("接口保存失败：没有接口 \"Pets\"") {
		t.Errorf("texts %q, want the failed write reported", tt.Texts())
	}
}

// TestCloseTabWhileStripBuilds keeps a tab strip that loses a tab mid-build
// from panicking: it is built from a snapshot of its tabs.
func TestCloseTabWhileStripBuilds(t *testing.T) {
	a := testApp(t)
	ws := a.space()
	for i := 0; i < 3; i++ {
		a.addTab(ws)
	}
	ws.active = 0
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()

	if err := tt.Click("关闭标签页 Pets"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(ws.tabs) != 3 {
		t.Errorf("tabs %d after closing one of four, want 3", len(ws.tabs))
	}
}

// TestDeleteCollectionWhileListBuilds keeps the home list from panicking
// when a row's menu deletes the collection under it.
func TestDeleteCollectionWhileListBuilds(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	for _, name := range []string{"A", "B"} {
		if _, err := a.store.NewCollection(name); err != nil {
			t.Fatal(err)
		}
	}
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("集合操作 A"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("删除集合"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("确认删除集合"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.store.Collections) != 1 || a.store.Collections[0].Name != "B" {
		t.Errorf("collections %+v, want only B left", a.store.Collections)
	}
}

// newTabThroughTheStrip opens a tab of a kind the way a reader does: the strip's
// +, then the kind its menu offers.
func newTabThroughTheStrip(t *testing.T, tt *ui.Tester, kindName string) {
	t.Helper()
	if err := tt.Click("新建标签页"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("新建 " + kindName); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
}

// TestVersionLineNamesTheApp checks the status bar calls the app ApiGo: the
// files it writes keep the old lower case name, the name it shows does not.
func TestVersionLineNamesTheApp(t *testing.T) {
	if got := appVersion(); got != "ApiGo" && !strings.HasPrefix(got, "ApiGo ") {
		t.Errorf("version line %q, want the app named ApiGo", got)
	}
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	if !tt.HasText("ApiGo") {
		t.Errorf("texts %q, want the app's name in the status bar", tt.Texts())
	}
}

// TestSettingsFitAShortWindow checks the settings panel stops inside a short
// window and its sections scroll: the last of them was pushed past the window's
// edge before.
func TestSettingsFitAShortWindow(t *testing.T) {
	a := testApp(t)
	a.settingsOpen = true
	tt := ui.NewTester(a.view, 1280, 560)
	tt.Frame()

	// The panel leaves the window's own edge to the dimmed backdrop.
	if r, _, _, _ := tt.Image().RGBAAt(640, 555).RGBA(); r>>8 > 200 {
		t.Errorf("the bottom of the window is %d, want the backdrop around the panel", r>>8)
	}

	before, ok := tt.Find("版本")
	if !ok {
		t.Fatal("texts of a short window hold no version line")
	}
	tt.Scroll(640, 300, 0, 400)
	tt.Frame()
	after, _ := tt.Find("版本")
	if after.Y >= before.Y {
		t.Errorf("the version line stayed at %.1f after scrolling from %.1f, want it up",
			after.Y, before.Y)
	}
}

// TestSettingsOpenTheProjectPage checks the settings name where the project is
// kept, and that a click opens the page in the desktop's browser.
func TestSettingsOpenTheProjectPage(t *testing.T) {
	a := testApp(t)
	a.settingsOpen = true
	tt := ui.NewTester(a.view, 1280, 800)

	if !tt.HasText(projectURL) {
		t.Fatalf("texts %q, want the address of the project", tt.Texts())
	}
	// The sections scroll: the address of the project sits under 通用, in view
	// without scrolling.
	if err := tt.Click(projectURL); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	opened := tt.OpenedURLs()
	if len(opened) == 0 || opened[len(opened)-1] != projectURL {
		t.Errorf("opened %q, want the project page", opened)
	}
}

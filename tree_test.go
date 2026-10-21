package main

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// treeApp is an app whose collection has its folders and interfaces on disk:
// the 根目录 holds Pets, and 分组 1 holds 列表.
func treeApp(t *testing.T) *app {
	t.Helper()
	s := newStore(t.TempDir())
	if err := s.load(); err != nil {
		t.Fatal(err)
	}
	col, err := s.newCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.addRequest(col.ID, "", "Pets", savedRequest{
		Kind: kindREST, Method: "GET", URL: "https://api.example.com/pets",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.addFolder(col.ID, "", "分组 1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.addRequest(col.ID, "分组 1", "列表", savedRequest{
		Kind: kindREST, Method: "GET",
	}); err != nil {
		t.Fatal(err)
	}
	a := newApp()
	a.store = s
	a.openCollection(col.ID)
	return a
}

// requestFile is where an interface of the tree is kept.
func requestFile(a *app, treePath string) string {
	parent := parentPath(treePath)
	name := treePath
	if parent != "" {
		name = treePath[len(parent)+1:]
	}
	return filepath.Join(a.store.folderDir(a.active, parent), name+".yaml")
}

// TestDeleteRequestAsksAndDeletes checks the tree's menu deletes an interface
// for good, and closes the tab that held it.
func TestDeleteRequestAsksAndDeletes(t *testing.T) {
	a := treeApp(t)
	file := requestFile(a, "Pets")
	openTreeRequest(t, a, "Pets")
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("接口操作 Pets"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("删除接口") {
		t.Fatalf("texts %q, want the interface's menu", tt.Texts())
	}
	if err := tt.Click("删除接口"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("无法恢复") {
		t.Fatalf("texts %q, want the warning of a deletion", tt.Texts())
	}
	if err := tt.Click("确认删除接口"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("file %q: %v, want it deleted", file, err)
	}
	if col := a.store.collection(a.active); len(col.Root.Requests) != 0 {
		t.Errorf("requests %+v, want the interface gone from the tree", col.Root.Requests)
	}
	for _, tab := range a.space().tabs {
		if tab.path == "Pets" {
			t.Errorf("a tab still holds %q, want it closed with the file", tab.path)
		}
	}
}

// TestDeleteFolderAsksAndDeletes checks deleting a folder takes its directory,
// and the interfaces inside it, off the disk.
func TestDeleteFolderAsksAndDeletes(t *testing.T) {
	a := treeApp(t)
	dir := a.store.folderDir(a.active, "分组 1")
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("文件夹操作 分组 1"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("删除文件夹"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("1 个接口") {
		t.Errorf("texts %q, want what the folder holds in the warning", tt.Texts())
	}
	if err := tt.Click("确认删除文件夹"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("directory %q: %v, want it deleted with its folder", dir, err)
	}
	if col := a.store.collection(a.active); len(col.Root.Folders) != 0 {
		t.Errorf("folders %+v, want the folder gone from the tree", col.Root.Folders)
	}
}

// TestMoveRequestIntoFolder checks the menu moves an interface into another
// folder, taking its file and the tab that shows it along.
func TestMoveRequestIntoFolder(t *testing.T) {
	a := treeApp(t)
	openTreeRequest(t, a, "Pets")
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("接口操作 Pets"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("移动接口"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("选择存放的文件夹") {
		t.Fatalf("texts %q, want the folders to move it to", tt.Texts())
	}
	if err := tt.Click("存放到 分组 1"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("移动"); err != nil {
		t.Fatal(err)
	}
	want := "分组 1/Pets"
	if _, err := os.Stat(a.store.folderDir(a.active, "分组 1") + "/Pets.yaml"); err != nil {
		t.Errorf("moved file: %v, want it in the folder", err)
	}
	if _, err := os.Stat(requestFile(a, "Pets")); !os.IsNotExist(err) {
		t.Errorf("file %q: %v, want it moved away", requestFile(a, "Pets"), err)
	}
	if got := a.current().path; got != want {
		t.Errorf("tab path %q, want it to follow the file to %q", got, want)
	}
}

// TestMoveFolder checks the menu moves a folder into another one, with the
// interfaces inside it, and that a folder cannot move into itself.
func TestMoveFolder(t *testing.T) {
	a := treeApp(t)
	if _, err := a.store.addFolder(a.active, "", "分组 2"); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("文件夹操作 分组 1"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("移动文件夹"); err != nil {
		t.Fatal(err)
	}
	// A folder cannot move into itself.
	if _, ok := tt.Find("存放到 分组 1"); ok {
		t.Error("the folder is offered as a target of its own move")
	}
	if err := tt.Click("存放到 分组 2"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("移动"); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(a.store.folderDir(a.active, "分组 2"), "分组 1", "列表.yaml")
	if _, err := os.Stat(moved); err != nil {
		t.Errorf("moved folder: %v, want its interface at %q", err, moved)
	}
	col := a.store.collection(a.active)
	nested := findFolder(&col.Root, "分组 2/分组 1")
	if nested == nil || len(nested.Requests) != 1 || nested.Requests[0].Path != "分组 2/分组 1/列表" {
		t.Errorf("tree %+v, want the folder and its request where they moved", nested)
	}
}

// TestDeleteMenuReadsInDangerColor checks the line that deletes is drawn in the
// danger color, apart from the rest of the menu.
func TestDeleteMenuReadsInDangerColor(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("接口操作 Pets"); err != nil {
		t.Fatal(err)
	}
	row, ok := tt.Find("删除接口")
	if !ok {
		t.Fatal("no line for deleting the interface")
	}
	danger := a.palette().Danger
	img := tt.Image()
	for y := int(row.Y); y < int(row.Y+row.H); y++ {
		for x := int(row.X); x < int(row.X+row.W); x++ {
			if c := img.RGBAAt(x, y); c.R == danger.R && c.G == danger.G && c.B == danger.B {
				return
			}
		}
	}
	t.Errorf("the line for deleting has no %v pixel out of %v, want the danger color", danger, row)
}

// openTreeRequest opens the interface at a path of the tree, as a click does.
func openTreeRequest(t *testing.T, a *app, treePath string) {
	t.Helper()
	col := a.store.collection(a.active)
	if col == nil {
		t.Fatalf("no collection %q", a.active)
	}
	f := findFolder(&col.Root, parentPath(treePath))
	if f == nil {
		t.Fatalf("no folder %q", parentPath(treePath))
	}
	for _, req := range f.Requests {
		if req.Path == treePath {
			a.openRequest(a.active, req)
			return
		}
	}
	t.Fatalf("no interface %q in %+v", treePath, f.Requests)
}

// rowBackground is the color the last frame drew inside a row of the tree,
// right of its text, where only the row's own background shows.
func rowBackground(t *testing.T, tt *ui.Tester, label string) color.RGBA {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("no row %q", label)
	}
	return tt.Image().RGBAAt(int(r.X+r.W-10), int(r.Y+r.H/2))
}

// TestTreeRowsTintUnderThePointer checks the rows of the tree answer the
// pointer, as the rows of a list do.
func TestTreeRowsTintUnderThePointer(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	page, hover := a.palette().Background, a.palette().SurfaceHover

	tt.Move(700, 400)
	tt.Frame()
	if got := rowBackground(t, tt, "请求 Pets"); !sameColor(got, page) {
		t.Errorf("row background %v away from the pointer, want %v", got, page)
	}

	row, _ := tt.Find("请求 Pets")
	tt.Move(row.X+row.W-10, row.Y+row.H/2)
	tt.Frame()
	if got := rowBackground(t, tt, "请求 Pets"); !sameColor(got, hover) {
		t.Errorf("row background %v under the pointer, want %v", got, hover)
	}

	folder, _ := tt.Find("文件夹 分组 1")
	tt.Move(folder.X+folder.W-10, folder.Y+folder.H/2)
	tt.Frame()
	if got := rowBackground(t, tt, "文件夹 分组 1"); !sameColor(got, hover) {
		t.Errorf("folder background %v under the pointer, want %v", got, hover)
	}
}

// TestOpenTabHighlightsItsRow checks the interface a tab shows reads in the
// accent, and that a draft, which has no place in the tree, tints nothing.
func TestOpenTabHighlightsItsRow(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	page := a.palette().Background

	openTreeRequest(t, a, "Pets")
	tt.Move(700, 400)
	tt.Frame()
	if got := rowBackground(t, tt, "请求 Pets"); sameColor(got, page) {
		t.Errorf("the row of the open interface is %v like the page, want the accent tint", got)
	}
	// The name of the open interface is drawn in the accent.
	row, _ := tt.Find("请求 Pets")
	accent := a.palette().Accent
	img := tt.Image()
	found := false
	for y := int(row.Y); y < int(row.Y+row.H) && !found; y++ {
		for x := int(row.X); x < int(row.X+row.W); x++ {
			if c := img.RGBAAt(x, y); c.R == accent.R && c.G == accent.G && c.B == accent.B {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("the open interface's row has no %v pixel, want its name in the accent", accent)
	}

	// Only the tab the workspace shows is tinted: another interface that has a
	// tab of its own is not.
	openTreeRequest(t, a, "分组 1/列表")
	tt.Move(700, 400)
	tt.Frame()
	if got := rowBackground(t, tt, "请求 Pets"); !sameColor(got, page) {
		t.Errorf("row background %v after another request took the workspace, want the page %v", got, page)
	}
	if got := rowBackground(t, tt, "请求 列表"); sameColor(got, page) {
		t.Errorf("the row of the request the workspace shows is %v like the page, want the accent tint", got)
	}

	// A draft tab has no file, so no row of the tree stands for it: the rows
	// are left as they were.
	b := treeApp(t)
	tt2 := ui.NewTester(b.view, 1280, 800)
	if err := tt2.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt2.Click("REST"); err != nil {
		t.Fatal(err)
	}
	if !b.current().unsaved() || b.current().path != "" {
		t.Fatalf("tab %q path %q, want a draft with no file", b.current().title, b.current().path)
	}
	tt2.Move(700, 400)
	tt2.Frame()
	if got := rowBackground(t, tt2, "请求 Pets"); !sameColor(got, page) {
		t.Errorf("row background %v with a draft open, want the page %v", got, page)
	}
}

// TestFolderPickerIndentsByDepth checks the folders of the save question step
// right under the folder they are in, and that the 根目录 sits at the top rather
// than above it.
func TestFolderPickerIndentsByDepth(t *testing.T) {
	a := treeApp(t)
	if _, err := a.store.addFolder(a.active, "分组 1", "子分组"); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("REST"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyS)

	root, ok := tt.Find("存放到 根目录")
	if !ok {
		t.Fatal("no folder row for the 根目录")
	}
	folder, ok := tt.Find("存放到 分组 1")
	if !ok {
		t.Fatal("no folder row for 分组 1")
	}
	nested, ok := tt.Find("存放到 子分组")
	if !ok {
		t.Fatal("no folder row for the nested folder")
	}
	if !(root.Y < folder.Y && folder.Y < nested.Y) {
		t.Errorf("rows at y %.0f / %.0f / %.0f, want the tree in its order", root.Y, folder.Y, nested.Y)
	}
	// The indentation is the depth of each folder: the 根目录 is at the top
	// rather than above it, which once put its name outside the panel.
	for _, c := range []struct {
		path  string
		depth int
	}{{"", 0}, {"分组 1", 1}, {"分组 1/子分组", 2}} {
		if got := depthOf(c.path); got != c.depth {
			t.Errorf("depth of %q is %d, want %d", c.path, got, c.depth)
		}
	}
}

// TestDialogsSitAtTopCenter checks the dialogs that ask something open at the
// top of the window, centered, rather than over its middle.
func TestDialogsSitAtTopCenter(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("接口操作 Pets"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("删除接口"); err != nil {
		t.Fatal(err)
	}
	cancel, ok := tt.Find("取消")
	if !ok {
		t.Fatal("no cancel button in the question")
	}
	if cancel.Y+cancel.H > 200 {
		t.Errorf("the question sits at y %.0f, want it at the top of the window", cancel.Y)
	}
	// Its buttons end near the middle of the window, the panel centered on it.
	confirm, _ := tt.Find("确认删除接口")
	mid := (cancel.X + confirm.X + confirm.W) / 2
	if mid < 500 || mid > 780 {
		t.Errorf("the question is centered on %.0f, want it around the middle of the window", mid)
	}

	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("REST"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyS)
	save, ok := tt.Find("保存")
	if !ok {
		t.Fatal("no save button in the folder question")
	}
	if save.Y+save.H > 400 {
		t.Errorf("the folder question sits at y %.0f, want it at the top of the window", save.Y)
	}
}

// addFolderNamed names the folder the add menu's dialog asks about, with the
// menu already open: it types the name and goes ahead.
func addFolderNamed(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	if err := tt.Click("文件夹名称"); err != nil {
		t.Fatal(err)
	}
	tt.Type(name)
	if err := tt.Click("新建"); err != nil {
		t.Fatal(err)
	}
}

// TestTreeRowOpensWhereverItIsClicked checks the whole row of an interface is
// its button: a click at its start opens the request, as one on its name does,
// and the pointer is the hand anywhere on it.
func TestTreeRowOpensWhereverItIsClicked(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	row, ok := tt.Find("请求 Pets")
	if !ok {
		t.Fatal("no row for Pets")
	}

	tt.Move(row.X+4, row.Y+row.H/2)
	if got := tt.Cursor(); got != ui.CursorPointer {
		t.Errorf("cursor %v at the start of the row, want %v", got, ui.CursorPointer)
	}
	tt.ClickAt(row.X+4, row.Y+row.H/2)
	if got := a.current().path; got != "Pets" {
		t.Errorf("tab path %q after a click at the start of the row, want Pets", got)
	}

	folder, _ := tt.Find("文件夹 分组 1")
	tt.ClickAt(folder.X+4, folder.Y+folder.H/2)
	if !a.foldersClosed["分组 1"] {
		t.Error("the folder stayed open after a click at the start of its row")
	}
}

// TestRequestHasOneTab checks opening an interface that a tab already shows
// brings that tab forward instead of opening another.
func TestRequestHasOneTab(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	ws := a.space()

	openTreeRequest(t, a, "Pets")
	tt.Frame()
	// The blank tab a collection opens with makes way for the interface.
	if tabs := len(ws.tabs); tabs != 1 {
		t.Fatalf("tabs %d after opening one interface, want 1", tabs)
	}
	tabs := len(ws.tabs)
	a.addTab(ws) // another empty request, so the Pets tab is not the active one
	tt.Frame()
	if err := tt.Click("请求 Pets"); err != nil {
		t.Fatal(err)
	}
	if got := len(ws.tabs); got != tabs+1 {
		t.Errorf("tabs %d after opening the interface again, want %d", got, tabs+1)
	}
	if got := a.current().path; got != "Pets" {
		t.Errorf("tab path %q, want the Pets tab brought forward", got)
	}
}

// TestNewFolderAsksForItsName checks making a folder asks for the name, and
// that a level of the tree refuses a name it already holds.
func TestNewFolderAsksForItsName(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("在根目录里新建接口"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("在根目录里新建文件夹"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("新建文件夹") || !tt.HasText("名称不能为空") {
		t.Fatalf("texts %q, want the dialog asking for a name", tt.Texts())
	}

	// The name 分组 1 is taken at this level.
	if err := tt.Click("文件夹名称"); err != nil {
		t.Fatal(err)
	}
	tt.Type("分组 1")
	if err := tt.Click("新建"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("同级目录里已有文件夹「分组 1」") {
		t.Errorf("texts %q, want the refusal of a name already there", tt.Texts())
	}
	if folders := a.store.collection(a.active).Root.Folders; len(folders) != 1 {
		t.Errorf("folders %+v, want the taken name left out", folders)
	}

	// Another name goes ahead.
	if err := tt.Click("文件夹名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("分组 2")
	if err := tt.Click("新建"); err != nil {
		t.Fatal(err)
	}
	folders := a.store.collection(a.active).Root.Folders
	if len(folders) != 2 || folders[1].Name != "分组 2" {
		t.Errorf("folders %+v, want the folder named as typed", folders)
	}
}

// TestRenameTabRenamesTheFile checks renaming a request in its tab renames the
// interface file too, so the tree and the disk say the same name.
func TestRenameTabRenamesTheFile(t *testing.T) {
	a := treeApp(t)
	old := requestFile(a, "Pets")
	openTreeRequest(t, a, "Pets")
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.RightClick("标签页 Pets"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("重命名请求"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("宠物")
	if err := tt.Click("完成重命名"); err != nil {
		t.Fatal(err)
	}

	if got := a.current().title; got != "宠物" {
		t.Errorf("tab title %q after renaming, want 宠物", got)
	}
	if got := a.current().path; got != "宠物" {
		t.Errorf("tab path %q after renaming, want 宠物", got)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("file %q: %v, want it renamed with the interface", old, err)
	}
	if _, err := os.Stat(requestFile(a, "宠物")); err != nil {
		t.Errorf("renamed file: %v, want it on disk", err)
	}
	col := a.store.collection(a.active)
	if req, ok := findRequest(col.Root, "宠物"); !ok || req.Path != "宠物" {
		t.Errorf("tree %+v, want the interface renamed there too", col.Root.Requests)
	}
	if _, ok := tt.Find("请求 宠物"); !ok {
		t.Errorf("texts %q, want the tree row named as renamed", tt.Texts())
	}
}

// TestScanFindsFilesChangedOutside checks the tree follows a file another
// program writes into the open collection.
func TestScanFindsFilesChangedOutside(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	if tt.HasText("请求 新接口") {
		t.Fatal("the tree lists an interface that is not there yet")
	}

	// Another program writes an interface beside the ones the app knows.
	if err := writeYAML(a.store.folderDir(a.active, "")+"/新接口.yaml", savedRequest{
		Kind: kindREST, Method: "GET",
	}); err != nil {
		t.Fatal(err)
	}
	a.scanned = time.Time{} // the next frame looks
	tt.Frame()

	if !tt.HasText("请求 新接口") {
		t.Errorf("texts %q, want the interface another program wrote", tt.Texts())
	}
}

// TestTempFilesStayInTmpDir checks a write leaves no half-written file beside
// the interfaces: it waits in the collection's .tmp directory.
func TestTempFilesStayInTmpDir(t *testing.T) {
	a := treeApp(t)
	if _, err := a.store.newCollection("临时"); err != nil {
		t.Fatal(err)
	}
	col := a.store.collection("临时")
	if _, err := a.store.addRequest(col.ID, "", "写入", savedRequest{Kind: kindREST, Method: "GET"}); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(a.store.collectionDir(col.ID))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("%q beside the interfaces, want the half-written file out of there", entry.Name())
		}
	}
	if _, err := os.Stat(filepath.Join(a.store.collectionDir(col.ID), tmpDirName)); err != nil {
		t.Errorf("temp directory: %v, want the writes kept in %q", err, tmpDirName)
	}
}

// TestCollectionOpensOnAStartHint checks a collection opens on what to do
// rather than an empty request, with no strip of request tabs over it, and
// that the strip comes with the first tab and leaves with the last.
func TestCollectionOpensOnAStartHint(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if !tt.HasText("还没有打开接口") || !tt.HasText("新建接口") || !tt.HasText("Ctrl") {
		t.Fatalf("texts %q, want the start hint of a collection", tt.Texts())
	}
	if tt.HasText("键第 1 行") {
		t.Error("the request panel shows behind the start hint")
	}
	if tt.HasText("新建标签页") || tt.HasText("环境：选择环境") {
		t.Errorf("texts %q, want no strip of request tabs before one is open", tt.Texts())
	}

	openTreeRequest(t, a, "Pets")
	tt.Frame()
	if tt.HasText("还没有打开接口") {
		t.Errorf("texts %q, want the hint gone once an interface is open", tt.Texts())
	}
	if !tt.HasText("键第 1 行") {
		t.Error("the request panel is missing after opening an interface")
	}
	if !tt.HasText("新建标签页") {
		t.Error("the strip of request tabs is missing once a request is open")
	}

	ws := a.space()
	a.closeTab(ws, 0)
	tt.Frame()
	if !tt.HasText("还没有打开接口") || tt.HasText("新建标签页") {
		t.Errorf("texts %q, want the start hint back once the last tab closed", tt.Texts())
	}
}

// TestRowMenuOpensFromTheMoreButton checks the "…" of a row opens its menu,
// and that a right click no longer does.
func TestRowMenuOpensFromTheMoreButton(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.RightClick("请求 Pets"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("删除接口") {
		t.Error("a right click opened the row's menu, want it opened by the … button only")
	}

	if err := tt.Click("接口操作 Pets"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"重命名", "复制接口", "移动接口", "删除接口"} {
		if !tt.HasText(want) {
			t.Errorf("texts %q, want %q in the interface's menu", tt.Texts(), want)
		}
	}

	// The 根目录 is the collection's own directory: it has nothing to rename,
	// move or delete, so its row ends with the + alone.
	if tt.HasText("文件夹操作 根目录") {
		t.Errorf("texts %q, want no … on the 根目录 row", tt.Texts())
	}
	if !tt.HasText("在根目录里新建接口") {
		t.Errorf("texts %q, want the + of the 根目录 row kept", tt.Texts())
	}
}

// TestInterfaceMenuRenamesAndCopies checks the menu's rename asks for a name
// and takes the file with it, and that copying makes another interface.
func TestInterfaceMenuRenamesAndCopies(t *testing.T) {
	a := treeApp(t)
	old := requestFile(a, "Pets")
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("接口操作 Pets"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("重命名"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("接口名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("宠物")
	if err := tt.Click("重命名"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("file %q: %v, want it renamed with the interface", old, err)
	}
	if _, err := os.Stat(requestFile(a, "宠物")); err != nil {
		t.Errorf("renamed file: %v, want it on disk", err)
	}
	if !tt.HasText("请求 宠物") {
		t.Errorf("texts %q, want the tree row renamed", tt.Texts())
	}

	if err := tt.Click("接口操作 宠物"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("复制接口"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("请求 宠物 副本") {
		t.Errorf("texts %q, want the copy in the tree", tt.Texts())
	}
	if _, err := os.Stat(requestFile(a, "宠物 副本")); err != nil {
		t.Errorf("copied file: %v, want it on disk", err)
	}
}

// TestFolderMenuRenames checks a folder's menu renames it, with its directory.
func TestFolderMenuRenames(t *testing.T) {
	a := treeApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("文件夹操作 分组 1"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("重命名"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("文件夹名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("设备")
	if err := tt.Click("重命名"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.store.collectionDir(a.active), "设备", "列表.yaml")); err != nil {
		t.Errorf("renamed folder: %v, want its interface inside it", err)
	}
	col := a.store.collection(a.active)
	if f := findFolder(&col.Root, "设备"); f == nil || len(f.Requests) != 1 {
		t.Errorf("tree %+v, want the folder renamed with its interface", col.Root.Folders)
	}
}

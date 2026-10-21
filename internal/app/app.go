package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"apigo/internal/design"
	"apigo/internal/req"
	"apigo/internal/store"
	"apigo/internal/widgets"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// errNoCollection is what a tree action reports while the home window is
// the active one.
var errNoCollection = errors.New("没有激活的集合")

// errNoOpener is what showing a folder reports where no desktop can show it.
var errNoOpener = errors.New("当前环境不能打开文件夹")

// workspace is what a collection keeps open: its request tabs and the tab
// being edited.
type workspace struct {
	tabs   []*tab
	active int
}

// newWorkspace is the workspace a collection opens with: no request tabs at
// all, so the workspace shows what to do rather than an empty request. A tab
// opens when an interface is opened or a new one is started.
func newWorkspace() *workspace {
	return &workspace{}
}

// empty reports whether the workspace shows no request: the workspace pane
// says what to do in its place.
func (ws *workspace) empty() bool { return len(ws.tabs) == 0 }

// panelMode is what the requests panel lists.
type panelMode int

const (
	panelRequests panelMode = iota
	panelHistory
)

// tab is one request of the tab strip: its own method, URL, parameters,
// headers, body and response, as Hoppscotch keeps one document per tab.
type tab struct {
	kind         kind
	collectionID string // the collection the request came from, if any
	path         string // the interface file it came from, "" when it has none
	folder       string // where a request that has no file yet is saved
	method       string
	url          string
	title        string
	reqTab       int

	params   widgets.State
	headers  widgets.State
	form     widgets.State // the rows of form-data and x-www-form-urlencoded
	bodyType string
	body     *editor.Buffer
	bodyEdit editor.State

	loading  bool
	resp     *response
	respEdit editor.State

	saved     string // the request as it was written, for unsaved
	dirty     bool
	renaming  bool
	renameOld string
}

func newTab() *tab {
	t := &tab{kind: kindREST, method: "GET", title: "Untitled", bodyType: "json"}
	t.body = editor.NewBuffer("")
	t.body.SetLanguage(editor.LanguageFor("body.json"))
	t.bodyEdit.Buffer = t.body
	t.bodyEdit.SoftWrap = true
	t.bodyEdit.Placeholder = "{\n  \"key\": \"value\"\n}"
	t.respEdit.Buffer = editor.NewBuffer("")
	t.respEdit.ReadOnly = true
	t.respEdit.SoftWrap = false
	return t
}

// startRename edits the tab's title in place.
func (t *tab) startRename() {
	t.renameOld, t.renaming = t.title, true
}

// commitRename keeps the edited title.
func (t *tab) commitRename() {
	if strings.TrimSpace(t.title) == "" {
		t.title = "Untitled"
	}
	t.renaming = false
	t.dirty = true
}

// commitTabRename keeps the edited title and renames the interface file the tab
// came from with it, so the tree, the strip and the disk say the same name. A
// tab that has no file yet keeps its title until it is saved.
func (a *app) commitTabRename(c *ui.Context, t *tab) {
	t.commitRename()
	if t.path == "" || t.collectionID == "" {
		return
	}
	old := t.path
	req, err := a.store.RenameRequest(t.collectionID, old, t.title)
	if err != nil {
		c.Toast("重命名失败：" + err.Error())
		return
	}
	t.title = req.Name
	a.retargetTabs(old, req.Path)
	t.markSaved()
}

// cancelRename puts the title back as it was.
func (t *tab) cancelRename() {
	t.title, t.renaming = t.renameOld, false
}

// snapshot is what a tab holds, as one string: saving writes it, and a tab
// whose request reads differently is unsaved.
func (t *tab) snapshot() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\x00%s\x00%s\x00%s\x00%s\x00", t.kind, t.method, t.url, t.title, t.bodyType)
	for _, list := range [][]widgets.Row{t.params.Rows, t.headers.Rows, t.form.Rows} {
		for _, r := range list {
			fmt.Fprintf(&b, "%v|%s|%s|%s|%s\x00", r.Enabled, r.Key, r.Value, r.File, r.FileName)
		}
		b.WriteString("\x01")
	}
	b.WriteString(t.body.String())
	return b.String()
}

// markSaved remembers the request as it stands: what it holds is on disk now.
func (t *tab) markSaved() {
	t.saved, t.dirty = t.snapshot(), false
}

// unsaved reports whether the tab holds a request that is not on disk: one
// that has no file of its own yet, or one that changed since it was written.
func (t *tab) unsaved() bool {
	return t.path == "" || t.snapshot() != t.saved
}

// setBodyLanguage highlights the body buffer as the selected type reads it,
// and says how that type's body is written.
func (t *tab) setBodyLanguage() {
	name, placeholder := "body.txt", "任意文本"
	switch t.bodyType {
	case "json":
		name, placeholder = "body.json", "{\n  \"key\": \"value\"\n}"
	case "xml":
		name, placeholder = "body.xml", "<note>\n  <to>you</to>\n</note>"
	case "graphql":
		name, placeholder = "body.json", "{\n  \"query\": \"{ pets { id } }\",\n  \"variables\": {}\n}"
	}
	t.body.SetLanguage(editor.LanguageFor(name))
	t.bodyEdit.Placeholder = placeholder
}

// app is the state the window shows: the home window or the open
// collections, their request tabs, the panels around them, and the
// appearance. The view rebuilds from it every frame, so it holds data only,
// never elements.
type app struct {
	mode        design.Mode
	applied     design.Mode // the appearance last handed to the window
	systemDark  bool
	theme       *ui.Theme
	themeAccent string // the accent the theme was built with
	accent      string // the accent choice, "" for the default
	language    string // the interface language Picked in the settings

	spaces map[string]*workspace
	open   []string // the open collections, in tab order
	active string   // the active collection, "" for the home window
	closed []string // recently closed collections, newest first

	panelMode    panelMode
	search       string
	kindFilter   kind // "" lists every kind
	envOpen      bool // the environment manager
	settingsOpen bool
	envID        string
	envName      string
	envVars      widgets.State
	// envEdits is how the environments stood when the manager opened: what a
	// cancel puts back.
	envEdits     []environment
	percent      float32 // share of the room the request pane takes
	methodOpen   bool
	filterOpen   bool
	kindMenuOpen bool
	moreOpen     bool
	envMenuOpen  bool

	collectionOpen bool   // the new-collection dialog
	collectionEdit string // the collection the dialog renames, "" for a new one
	collectionName string
	question       confirm          // the question a dialog asks before an action
	where          folderQuestion   // the folder a dialog asks for
	foldersClosed  map[string]bool  // folders the tree shows collapsed
	menus          map[string]*bool // the popover menus open, by their key
	saveErr        string           // the interface write that failed, shown once
	languageOpen   bool             // the language menu of the settings

	// watched is the collection whose directory a watch follows, and
	// watchStop ends that watch.
	watched   string
	watchStop func()

	store *dataStore

	// after schedules work on the window's thread. Tests leave it nil, and
	// the work then runs inline.
	after func(func())

	// opener shows a directory in the desktop's file manager. main fills it
	// with mygo.Shell.OpenPath; tests replace it.
	opener func(string) error

	// baseDir is the default data directory, where the pointer to the one the
	// user picked is kept; main fills it, and chooseDir asks the desktop for
	// another one.
	baseDir   string
	chooseDir func() (string, error)

	// chooseFile asks the desktop for a file to upload. main fills it with the
	// file dialog; tests replace it.
	chooseFile func() (string, error)

	// watch follows a collection's directory for the changes another program
	// makes in it. main fills it with mygo's watch; tests replace it.
	watch watchStarter
}

func newApp() *app {
	return &app{
		mode:     design.System,
		applied:  design.Mode(-1),
		accent:   design.DefaultAccent.Name,
		language: languages[0],
		percent:  55,
		spaces:   map[string]*workspace{},
		store:    store.New(""),
	}
}

// space is the active collection's workspace, nil in the home window.
func (a *app) space() *workspace {
	if a.active == "" {
		return nil
	}
	ws := a.spaces[a.active]
	if ws == nil {
		ws = newWorkspace()
		a.spaces[a.active] = ws
	}
	return ws
}

// current is the request the panes edit, nil in the home window.
func (a *app) current() *tab {
	ws := a.space()
	if ws == nil || len(ws.tabs) == 0 {
		return nil
	}
	return ws.tabs[ws.active]
}

// openCollection opens a collection, creating its workspace on first use.
func (a *app) openCollection(id string) {
	if a.collectionIndex(id) < 0 {
		return
	}
	if _, ok := a.spaces[id]; !ok {
		a.spaces[id] = newWorkspace()
	}
	if !containsString(a.open, id) {
		a.open = append(a.open, id)
	}
	a.closed = removeString(a.closed, id)
	a.active = id
	// Opening a collection reads its files again: another program may have
	// changed them since it was last listed.
	a.reloadActive()
}

// reloadActive reads the open collection's files again, keeping the tabs that
// are open on them: an interface that went away leaves its tab a draft, so
// saving it again makes a new file rather than failing on a missing one.
func (a *app) reloadActive() {
	if a.active == "" {
		return
	}
	if err := a.store.ReloadCollection(a.active); err != nil {
		a.saveErr = "读取集合失败：" + err.Error()
		return
	}
	a.store.RemoveTempFiles(a.active)
	col := a.store.Collection(a.active)
	if col == nil {
		return
	}
	for _, t := range a.space().tabs {
		if t.path != "" && !store.TreeHasRequest(col.Root, t.path) {
			t.path = "" // the file went away: the tab is a draft again
		}
	}
}

// closeCollection closes a collection's workspace, writing its tabs back
// first; its interfaces stay on disk. The home window shows when no
// collection is open.
func (a *app) closeCollection(id string) {
	if ws := a.spaces[id]; ws != nil {
		for _, t := range ws.tabs {
			a.saveTab(t)
			a.dropTabMenu(t)
		}
	}
	a.forgetCollection(id)
}

// forgetCollection drops a collection's workspace, keeping the collection in
// the recently closed list.
func (a *app) forgetCollection(id string) {
	a.open = removeString(a.open, id)
	delete(a.spaces, id)
	a.closed = append([]string{id}, removeString(a.closed, id)...)
	if a.active == id {
		a.active = ""
		if len(a.open) > 0 {
			a.active = a.open[len(a.open)-1]
		}
	}
}

// closeOtherCollections closes every open collection but id.
func (a *app) closeOtherCollections(id string) {
	for _, other := range append([]string(nil), a.open...) {
		if other != id {
			a.closeCollection(other)
		}
	}
	a.openCollection(id)
}

// openHome shows the home window.
func (a *app) openHome() { a.active = "" }

// selectTab switches the workspace to a tab, writing the one being left
// back to its interface file.
func (a *app) selectTab(ws *workspace, i int) {
	if i == ws.active || i < 0 || i >= len(ws.tabs) {
		return
	}
	a.leaveTab(ws)
	ws.active = i
}

// leaveTab writes the active tab back before the workspace moves on.
func (a *app) leaveTab(ws *workspace) {
	if ws == nil || ws.active < 0 || ws.active >= len(ws.tabs) {
		return
	}
	a.saveTab(ws.tabs[ws.active])
}

// saveTab writes a tab's request back to the interface file it was opened
// from; a tab that never came from a file keeps nothing.
func (a *app) saveTab(t *tab) {
	if t == nil || t.path == "" || t.collectionID == "" {
		return
	}
	if err := a.store.UpdateRequest(t.collectionID, t.path, savedFromTab(t)); err != nil {
		a.saveErr = "接口保存失败：" + err.Error()
		return
	}
	t.markSaved()
}

// addTab opens an empty request after the active one.
func (a *app) addTab(ws *workspace) {
	a.openTab(ws, newTab(), ws.active+1)
}

// openTab puts t at position i of the workspace and shows it, writing back the
// tab it leaves when that is another one. An empty workspace takes the tab at
// its only place, whatever i asked for.
func (a *app) openTab(ws *workspace, t *tab, i int) {
	a.leaveTab(ws)
	if i > len(ws.tabs) {
		i = len(ws.tabs)
	}
	ws.tabs = append(ws.tabs, nil)
	copy(ws.tabs[i+1:], ws.tabs[i:])
	ws.tabs[i] = t
	ws.active = i
}

// duplicateTab opens a copy of the tab at i after it, as a new request that
// has no interface file of its own yet.
func (a *app) duplicateTab(ws *workspace, i int) {
	src := ws.tabs[i]
	dup := newTab()
	dup.kind, dup.collectionID = src.kind, src.collectionID
	dup.method, dup.url, dup.title = src.method, src.url, src.title
	dup.bodyType = src.bodyType
	dup.params.Rows = append([]widgets.Row(nil), src.params.Rows...)
	dup.headers.Rows = append([]widgets.Row(nil), src.headers.Rows...)
	dup.form.Rows = append([]widgets.Row(nil), src.form.Rows...)
	dup.body.Set(src.body.String())

	a.openTab(ws, dup, i+1)
}

// closeTab closes the tab at i. Closing the last one leaves the workspace
// with no tab at all, so the pane says what to do again.
func (a *app) closeTab(ws *workspace, i int) {
	if i < 0 || i >= len(ws.tabs) {
		return
	}
	a.saveTab(ws.tabs[i])
	a.dropTabMenu(ws.tabs[i])
	ws.tabs = append(ws.tabs[:i], ws.tabs[i+1:]...)
	if i < ws.active {
		ws.active--
	}
	if ws.active >= len(ws.tabs) {
		ws.active = len(ws.tabs) - 1
	}
	if ws.empty() {
		ws.active = 0
	}
}

// closeOtherTabs closes every tab but the one at i.
func (a *app) closeOtherTabs(ws *workspace, i int) {
	if i < 0 || i >= len(ws.tabs) {
		return
	}
	for j, t := range ws.tabs {
		if j != i {
			a.saveTab(t)
			a.dropTabMenu(t)
		}
	}
	ws.tabs = []*tab{ws.tabs[i]}
	ws.active = 0
}

// closeTabAt closes the tab a pointer names, wherever it sits now.
func (a *app) closeTabAt(ws *workspace, t *tab) {
	if i := a.tabIndex(ws, t); i >= 0 {
		a.closeTab(ws, i)
	}
}

// closeTabsExcept closes every tab but the one a pointer names.
func (a *app) closeTabsExcept(ws *workspace, t *tab) {
	if i := a.tabIndex(ws, t); i >= 0 {
		a.closeOtherTabs(ws, i)
	}
}

// duplicateTabAt opens a copy of the tab a pointer names, after it.
func (a *app) duplicateTabAt(ws *workspace, t *tab) {
	if i := a.tabIndex(ws, t); i >= 0 {
		a.duplicateTab(ws, i)
	}
}

// dropTabMenu forgets the menu of a tab that is going away.
func (a *app) dropTabMenu(t *tab) { delete(a.menus, requestTabMenuKey(t)) }

// newCollection creates a collection with its default environments and
// opens it.
func (a *app) newCollection(name string) error {
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("集合 %d", len(a.store.Collections)+1)
	}
	col, err := a.store.NewCollection(name)
	if err != nil {
		return err
	}
	a.openCollection(col.ID)
	return nil
}

// openCollectionModal asks for the name of a new collection.
func (a *app) openCollectionModal() {
	a.collectionEdit, a.collectionName = "", ""
	a.collectionOpen = true
}

// openRenameModal asks for the new name of a collection.
func (a *app) openRenameModal(id string) {
	i := a.collectionIndex(id)
	if i < 0 {
		return
	}
	a.collectionEdit, a.collectionName = id, a.store.Collections[i].Name
	a.collectionOpen = true
}

// openFolder shows a directory in the desktop's file manager.
func (a *app) openFolder(path string) error {
	if a.opener == nil {
		return errNoOpener
	}
	return a.opener(path)
}

// chooseDataDir asks the desktop for a directory and moves the app's data
// there, keeping the choice for the next run.
func (a *app) chooseDataDir(c *ui.Context) {
	if a.chooseDir == nil {
		c.Toast("当前环境不能选择目录")
		return
	}
	dir, err := a.chooseDir()
	if err != nil {
		c.Toast("选择目录失败：" + err.Error())
		return
	}
	if dir == "" {
		return // the dialog was cancelled
	}
	if err := a.useDataDir(dir); err != nil {
		c.Toast("切换数据目录失败：" + err.Error())
		return
	}
	c.Toast("数据目录已切换到 " + dir)
}

// staticScope is the collection a tab's uploaded files live in: the one the
// request came from, or the one whose workspace holds it before it is saved.
func (a *app) staticScope(t *tab) string {
	if t.collectionID != "" {
		return t.collectionID
	}
	return a.active
}

// formParts turns the form rows into the fields a request sends, resolving the
// file each one holds to the path the collection keeps it at.
func (a *app) formParts(t *tab, rows []widgets.Row) []formPart {
	scope := a.staticScope(t)
	parts := make([]formPart, 0, len(rows))
	for _, row := range rows {
		part := formPart{Key: row.Key, Value: row.Value}
		if row.File != "" && scope != "" {
			part.FilePath = a.store.StaticPath(scope, row.File)
			part.FileName = row.FileName
			if part.FileName == "" {
				part.FileName = filepath.Base(row.File)
			}
		}
		parts = append(parts, part)
	}
	return parts
}

// pickFormFile asks the desktop for a file and keeps it in the tab's
// collection, under static/<day>/<hash>. It returns the path the collection
// keeps it at and the name the file had, which the field records.
func (a *app) pickFormFile(c *ui.Context, t *tab) (string, string, error) {
	scope := a.staticScope(t)
	if scope == "" {
		c.Toast("先打开一个集合，再上传文件")
		return "", "", nil
	}
	if a.chooseFile == nil {
		c.Toast("当前环境不能选择文件")
		return "", "", nil
	}
	path, err := a.chooseFile()
	if err != nil {
		c.Toast("选择文件失败：" + err.Error())
		return "", "", nil
	}
	if path == "" {
		return "", "", nil // the dialog was cancelled
	}
	rel, err := a.store.ImportStatic(scope, path)
	if err != nil {
		c.Toast("保存文件失败：" + err.Error())
		return "", "", nil
	}
	return rel, filepath.Base(path), nil
}

// dropFormFile deletes a file a form field no longer holds.
func (a *app) dropFormFile(t *tab, rel string) {
	scope := a.staticScope(t)
	if scope == "" {
		return
	}
	if err := a.store.RemoveStatic(scope, rel); err != nil {
		a.saveErr = "删除旧文件失败：" + err.Error()
	}
}

// useDataDir reads the collections of another data directory, its
// environments and settings, and keeps the choice for the next run.
func (a *app) useDataDir(dir string) error {
	next := store.New(dir)
	if err := next.Load(); err != nil {
		return err
	}
	if err := store.RememberDir(a.baseDir, dir); err != nil {
		return err
	}
	a.store = next
	a.forgetEverything()
	a.applySettings()
	return nil
}

// forgetEverything drops what the window held for the collections that were
// open, as reading another data directory leaves none of them.
func (a *app) forgetEverything() {
	a.spaces = map[string]*workspace{}
	a.open, a.closed, a.active = nil, nil, ""
	a.envOpen, a.envID, a.envName = false, "", ""
	a.foldersClosed, a.menus = nil, nil
	a.search, a.kindFilter, a.panelMode = "", "", panelRequests
}

// saveNewCollection creates the collection the dialog names, or renames the
// one the dialog was opened for.
func (a *app) saveNewCollection(c *ui.Context) {
	name := strings.TrimSpace(a.collectionName)
	if name == "" {
		return
	}
	editing := a.collectionEdit
	if editing != "" {
		if err := a.renameCollection(editing, name); err != nil {
			c.Toast("重命名集合失败：" + err.Error())
			return
		}
		a.collectionOpen, a.collectionName, a.collectionEdit = false, "", ""
		return
	}
	if err := a.newCollection(name); err != nil {
		c.Toast("新建集合失败：" + err.Error())
		return
	}
	a.collectionOpen, a.collectionName = false, ""
}

// renameCollection renames a collection and keeps whatever is open on it
// pointing at the new id.
func (a *app) renameCollection(id, name string) error {
	col, err := a.store.RenameCollection(id, name)
	if err != nil {
		return err
	}
	newID := col.ID
	if ws := a.spaces[id]; ws != nil && newID != id {
		delete(a.spaces, id)
		a.spaces[newID] = ws
		for _, t := range ws.tabs {
			if t.collectionID == id {
				t.collectionID = newID
			}
		}
	}
	a.open = renameInList(a.open, id, newID)
	a.closed = renameInList(a.closed, id, newID)
	if a.active == id {
		a.active = newID
	}
	return nil
}

// renameInList swaps an id for the one it was renamed to.
func renameInList(ids []string, from, to string) []string {
	for i, id := range ids {
		if id == from {
			ids[i] = to
		}
	}
	return ids
}

// deleteCollection takes a collection off the list, closing it when it is open.
// Its directory, interfaces and environments stay where they are: the next run
// reads hidden.yaml and leaves it out again.
func (a *app) deleteCollection(id string) error {
	if err := a.store.HideCollection(id); err != nil {
		return err
	}
	if a.active == id || containsString(a.open, id) {
		// The collection left the window, so its workspace is dropped without
		// writing its tabs back.
		a.forgetCollection(id)
	}
	return nil
}

// openRequest opens a saved request in its collection, in the tab that already
// shows it when there is one: a request has a single tab, and opening it again
// brings that tab forward rather than making another.
func (a *app) openRequest(collectionID string, r savedRequest) {
	a.openCollection(collectionID)
	ws := a.space()
	if r.Path != "" {
		if i := a.tabOfPath(ws, r.Path); i >= 0 {
			a.selectTab(ws, i)
			return
		}
	}
	t := newTab()
	t.kind = r.Kind
	if t.kind == "" {
		t.kind = kindREST
	}
	t.collectionID = collectionID
	t.title = r.Name
	if t.title == "" {
		t.title = "Untitled"
	}
	if r.Method != "" {
		t.method = r.Method
	}
	t.url = r.URL
	t.params.Rows = rowsFromSaved(r.Params)
	t.headers.Rows = rowsFromSaved(r.Headers)
	t.form.Rows = rowsFromSaved(r.Form)
	if r.BodyType != "" {
		t.bodyType = r.BodyType
		t.setBodyLanguage()
	}
	t.body.Set(r.Body)
	t.path, t.folder = r.Path, store.ParentPath(r.Path)
	t.markSaved()

	a.openTab(ws, t, ws.active+1)
}

// openHistory opens a history entry in the active collection.
func (a *app) openHistory(entry historyEntry) {
	if a.active == "" {
		return
	}
	a.openRequest(a.active, savedRequest{
		Name: entry.Title, Kind: entry.Kind, Method: entry.Method, URL: entry.URL,
	})
}

// addRequest opens a request of the given kind as a draft: nothing is written
// until it is saved with Ctrl+S, which asks where in the collection it goes,
// offering the folder the + that made it stands on.
func (a *app) addRequest(k kind, folderPath string) error {
	if a.active == "" {
		return errNoCollection
	}
	t := newTab()
	t.kind, t.collectionID, t.folder = k, a.active, folderPath
	if k != kindREST {
		t.method = ""
	}
	ws := a.space()
	a.openTab(ws, t, ws.active+1)
	return nil
}

// newFolder adds a folder inside a folder of the active collection's tree,
// the 根目录 when folderPath is empty.
func (a *app) newFolder(folderPath string) error {
	col := a.store.Collection(a.active)
	if col == nil {
		return errNoCollection
	}
	parent := store.FindFolder(&col.Root, folderPath)
	if parent == nil {
		return fmt.Errorf("没有文件夹 %q", folderPath)
	}
	_, err := a.store.AddFolder(a.active, folderPath,
		fmt.Sprintf("分组 %d", len(parent.Folders)+1))
	return err
}

// saveCurrentToCollection writes the active request into a collection as an
// interface file of its own, at the top of the tree.
func (a *app) saveCurrentToCollection(collectionID string) error {
	t := a.current()
	if t == nil {
		return fmt.Errorf("没有激活的请求")
	}
	req, err := a.store.AddRequest(collectionID, "", t.title, savedFromTab(t))
	if err != nil {
		return err
	}
	t.collectionID, t.title, t.path, t.folder = collectionID, req.Name, req.Path, ""
	t.markSaved()
	return nil
}

// savedFromTab is the interface a tab's request stands for.
func savedFromTab(t *tab) savedRequest {
	req := savedRequest{
		Kind:     t.kind,
		Method:   t.method,
		URL:      t.url,
		Params:   rowsToSaved(t.params.Rows),
		Headers:  rowsToSaved(t.headers.Rows),
		BodyType: t.bodyType,
	}
	// A file keeps what its format edits: the text of a code body, the rows of
	// an encoded one.
	switch bodyFormatOf(t.bodyType).Edits {
	case editsRows:
		req.Form = rowsToSaved(t.form.Rows)
	case editsText:
		req.Body = t.body.String()
	}
	return req
}

// collectionIndex returns where a collection is kept, or -1.
func (a *app) collectionIndex(id string) int { return a.store.CollectionIndex(id) }

// collectionEnvironmentList is the ordered scope of the environment menus:
// the active collection's environments.
func (a *app) collectionEnvironments() []environment {
	if a.active == "" {
		return nil
	}
	return a.store.EnvironmentsFor(a.active)
}

// activeEnvironment is the environment the active collection sends with.
func (a *app) activeEnvironment() *environment {
	if a.active == "" {
		return nil
	}
	return a.store.ActiveEnvironment(a.active)
}

// openEnvManager shows the environments of the active collection.
func (a *app) openEnvManager() {
	a.envOpen = true
	// What the manager changes waits for its save: a cancel puts this back.
	a.envEdits = a.store.CloneEnvironments()
	envs := a.collectionEnvironments()
	if len(envs) == 0 {
		a.editEnvironment("")
		return
	}
	id := envs[0].ID
	if active := a.store.ActiveEnvironment(a.active); active != nil {
		id = active.ID
	}
	a.editEnvironment(id)
}

// editEnvironment loads an environment's variables for editing.
func (a *app) editEnvironment(id string) {
	a.envID = id
	env := a.store.Environment(id)
	if env == nil {
		a.envName, a.envVars = "", widgets.State{}
		return
	}
	a.envName = env.Name
	a.envVars = widgets.State{Rows: rowsFromSaved(env.Vars), Compact: true}
}

// saveEnvEdits keeps what the environment manager changed: the variables of
// the environment being edited, its name, and which one is active, all of it
// written in one go. It is the save button's work.
func (a *app) saveEnvEdits(c *ui.Context) {
	if env := a.store.Environment(a.envID); env != nil {
		env.Vars = rowsToSaved(a.envVars.Rows)
	}
	if err := a.store.SaveEnvironments(); err != nil {
		c.Toast("保存环境失败：" + err.Error())
		return
	}
	a.envEdits = nil
	a.envOpen = false
	c.Toast("环境已保存")
}

// discardEnvEdits puts the environments back as they stood when the manager
// opened: what it changed without saving is forgotten.
func (a *app) discardEnvEdits() {
	if a.envEdits == nil {
		return
	}
	a.store.RestoreEnvironments(a.envEdits)
	a.envEdits = nil
	if a.store.Environment(a.envID) != nil {
		a.editEnvironment(a.envID)
		return
	}
	id := ""
	if active := a.store.ActiveEnvironment(a.active); active != nil {
		id = active.ID
	}
	a.editEnvironment(id)
}

// resolvedMode is the palette to draw with: System resolves to the
// desktop's appearance as last seen.
func (a *app) resolvedMode() design.Mode {
	if a.mode != design.System {
		return a.mode
	}
	if a.systemDark {
		return design.Dark
	}
	return design.Light
}

// accent is the accent choice in use.
func (a *app) accentColor() design.Accent { return design.AccentByName(a.accent) }

// applySettings takes the appearance, accent and language the last run kept.
func (a *app) applySettings() {
	s := a.store.Settings
	a.mode = design.ParseMode(s.Mode)
	if s.Accent != "" {
		a.accent = s.Accent
	}
	if s.Language != "" {
		a.language = s.Language
	}
	a.applied = design.Mode(-1) // the theme is built again with them
}

// rememberSettings keeps the appearance, accent and language for the next
// run.
func (a *app) rememberSettings() error {
	a.store.Settings = settings{
		Mode:     a.mode.Key(),
		Accent:   a.accent,
		Language: a.language,
	}
	return a.store.SaveSettings()
}

// palette returns the colors of this frame.
func (a *app) palette() design.Palette {
	return design.PaletteFor(a.resolvedMode(), a.accentColor())
}

// themeFor is the theme the window draws with: the palette in use, so buttons,
// inputs and scroll bars take the app's own colors rather than MyGo's default
// ones. A window that follows the desktop takes the desktop's appearance as it
// stands when the window opens; MyGo cannot hand a window back to the
// desktop's own theme once SetTheme replaced it, so following ends there and
// the next start picks up a change.
func (a *app) themeFor(c *ui.Context) *ui.Theme {
	if a.applied == design.Mode(-1) {
		a.systemDark = c.Theme().Dark
	}
	target := a.resolvedMode()
	accent := a.accentColor()
	if a.theme == nil || a.applied != target || a.themeAccent != accent.Name {
		a.theme = design.Theme(target, accent)
		a.applied, a.themeAccent = target, accent.Name
	}
	return a.theme
}

// send resolves the active tab's variables, builds its request and sends
// it off the UI thread, keeping the response for the tab it came from. A
// JSON body is sent only once it parses, so the editor's text reaches the
// server as written.
func (a *app) send(c *ui.Context) {
	t := a.current()
	if t == nil || t.loading {
		return
	}
	if t.kind != kindREST {
		c.Toast(store.KindName(t.kind) + " 请求尚未实现")
		return
	}
	if t.bodyType == "json" || t.bodyType == "graphql" {
		if _, err := indentJSON(t.body.String()); err != nil {
			c.Toast("请求体不是合法 JSON：" + err.Error())
			return
		}
	}
	vars := a.store.VarsFor(a.active)
	url, missing := req.ResolveVars(t.url, vars)
	params, missingParams := req.ResolveRows(t.params.Enabled(), vars)
	headers, missingHeaders := req.ResolveRows(t.headers.Enabled(), vars)
	form, missingForm := req.ResolveRows(t.form.Enabled(), vars)
	body, missingBody := req.ResolveVars(t.body.String(), vars)
	missing = req.Dedupe(append(append(append(append(missing, missingParams...),
		missingHeaders...), missingForm...), missingBody...))
	if len(missing) > 0 {
		c.Toast("未解析的环境变量：" + strings.Join(missing, "、"))
		return
	}
	built, err := req.BuildRequest(req.Spec{
		Method:   t.method,
		RawURL:   url,
		Params:   params,
		Headers:  headers,
		BodyType: t.bodyType,
		Body:     body,
		Form:     a.formParts(t, form),
	})
	if err != nil {
		c.Toast(err.Error())
		return
	}
	t.loading, t.resp = true, nil
	run := func() { a.apply(t, req.Execute(context.Background(), built)) }
	if a.after == nil {
		run()
		return
	}
	go func() { a.after(run) }()
}

// apply stores a finished response in its tab, records it in the history,
// and shows the body read-only with JSON highlighting when it is JSON.
func (a *app) apply(t *tab, res *response) {
	t.resp, t.loading = res, false
	if res == nil {
		return
	}
	_ = a.store.AddHistory(historyEntry{
		Method: t.method,
		URL:    t.url,
		Status: res.Status,
		At:     time.Now(),
		Title:  t.title,
		Kind:   t.kind,
	})
	if res.Err != nil {
		return
	}
	lang := languageForResponse(res.Headers.Get("Content-Type"), res.Body)
	t.respEdit.Buffer.SetLanguage(lang)
	if lang == nil {
		t.respEdit.Buffer.Set(res.Body)
		return
	}
	// A JSON body is indented as it comes in: the server sends it over one
	// line more often than not, and the pane reads it as code.
	t.respEdit.Buffer.Set(prettyResponse(res.Body))
}

// languageForResponse is the syntax a response is shown in: JSON when the
// server says so, or when the body parses as JSON whatever it says.
func languageForResponse(contentType, body string) *editor.Language {
	if isJSON(contentType, body) {
		return editor.LanguageFor("response.json")
	}
	return nil
}

// prettyResponse indents a JSON body, keeping it as it came when it does not
// parse.
func prettyResponse(body string) string {
	if pretty, err := indentJSON(body); err == nil {
		return pretty
	}
	return body
}

// formatBody rewrites the active tab's body with two-space indentation.
func (a *app) formatBody(c *ui.Context) {
	t := a.current()
	if t == nil {
		return
	}
	pretty, err := indentJSON(t.body.String())
	if err != nil {
		c.Toast("请求体不是合法 JSON：" + err.Error())
		return
	}
	t.body.Set(pretty)
}

// indentJSON re-indents JSON text with two spaces, keeping its keys' order.
func indentJSON(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", nil
	}
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(trimmed), "", "  "); err != nil {
		return "", err
	}
	return out.String(), nil
}

// isJSON reports whether a response is JSON: by its content type, or by
// parsing when the server sent none it can be trusted for.
func isJSON(contentType, body string) bool {
	if strings.Contains(strings.ToLower(contentType), "json") {
		return json.Valid([]byte(body))
	}
	if contentType != "" {
		return false
	}
	return json.Valid([]byte(body))
}

// rowsFromSaved turns stored rows into the rows a key/value list edits,
// with the empty row that ends it.
func rowsFromSaved(rows []savedRow) []widgets.Row {
	out := make([]widgets.Row, 0, len(rows)+1)
	for _, r := range rows {
		out = append(out, widgets.Row{
			Enabled: r.Enabled, Key: r.Key, Value: r.Value,
			File: r.File, FileName: r.FileName,
		})
	}
	return append(out, widgets.Row{Enabled: true})
}

// rowsToSaved turns edited rows into stored ones, leaving out the empty
// rows that only give room to type.
func rowsToSaved(rows []widgets.Row) []savedRow {
	out := make([]savedRow, 0, len(rows))
	for _, r := range rows {
		if r.Key == "" && r.Value == "" && r.File == "" {
			continue
		}
		out = append(out, savedRow{
			Enabled: r.Enabled, Key: r.Key, Value: r.Value,
			File: r.File, FileName: r.FileName,
		})
	}
	return out
}

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func removeString(list []string, want string) []string {
	out := list[:0]
	for _, item := range list {
		if item != want {
			out = append(out, item)
		}
	}
	return out
}

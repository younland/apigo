package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"apigo/internal/design"
	"apigo/widgets"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// errNoCollection is what a tree action reports while the home window is
// the active one.
var errNoCollection = errors.New("没有激活的集合")

// workspace is what a collection keeps open: its request tabs and the tab
// being edited.
type workspace struct {
	tabs   []*tab
	active int
}

func newWorkspace() *workspace { return &workspace{tabs: []*tab{newTab()}} }

// panelMode is what the requests panel lists.
type panelMode int

const (
	panelRequests panelMode = iota
	panelHistory
)

// bodyTypes are the request body formats the Body tab offers.
var bodyTypes = []string{"none", "json", "text"}

// tab is one request of the tab strip: its own method, URL, parameters,
// headers, body and response, as Hoppscotch keeps one document per tab.
type tab struct {
	kind         kind
	collectionID string // the collection the request came from, if any
	path         string // the interface file it came from, "" when it has none
	method       string
	url          string
	title        string
	reqTab       int

	params   widgets.State
	headers  widgets.State
	bodyType string
	body     *editor.Buffer
	bodyEdit editor.State

	loading  bool
	resp     *response
	respEdit editor.State

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

// cancelRename puts the title back as it was.
func (t *tab) cancelRename() {
	t.title, t.renaming = t.renameOld, false
}

// hasContent reports whether the tab holds a request: a URL, a body, or
// key/value rows that would be sent.
func (t *tab) hasContent() bool {
	return strings.TrimSpace(t.url) != "" || t.body.Len() > 0 ||
		len(t.params.Enabled()) > 0 || len(t.headers.Enabled()) > 0
}

// marked reports whether the strip marks the tab with the edit dot.
func (t *tab) marked() bool { return t.dirty || t.hasContent() }

// setBodyLanguage highlights the body buffer as the selected type reads it.
func (t *tab) setBodyLanguage() {
	name := "body.txt"
	if t.bodyType == "json" {
		name = "body.json"
	}
	t.body.SetLanguage(editor.LanguageFor(name))
}

// app is the state the window shows: the home window or the open
// collections, their request tabs, the panels around them, and the
// appearance. The view rebuilds from it every frame, so it holds data only,
// never elements.
type app struct {
	mode       design.Mode
	applied    design.Mode // the appearance last handed to the window
	systemDark bool
	theme      *ui.Theme

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
	percent      float32 // share of the room the request pane takes
	methodOpen   bool
	filterOpen   bool
	kindMenuOpen bool
	moreOpen     bool
	envMenuOpen  bool

	collectionOpen bool // the new-collection dialog
	collectionName string
	foldersClosed  map[string]bool  // folders the tree shows collapsed
	folderMenus    map[string]*bool // the add menus open per folder
	saveErr        string           // the interface write that failed, shown once

	store *store

	// after schedules work on the window's thread. Tests leave it nil, and
	// the work then runs inline.
	after func(func())
}

func newApp() *app {
	return &app{
		mode:    design.System,
		applied: design.Mode(-1),
		percent: 55,
		spaces:  map[string]*workspace{},
		store:   newStore(""),
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
}

// closeCollection closes a collection's workspace, writing its tabs back
// first; its interfaces stay on disk. The home window shows when no
// collection is open.
func (a *app) closeCollection(id string) {
	if ws := a.spaces[id]; ws != nil {
		for _, t := range ws.tabs {
			a.saveTab(t)
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
	if err := a.store.updateRequest(t.collectionID, t.path, savedFromTab(t)); err != nil {
		a.saveErr = "接口保存失败：" + err.Error()
	}
}

// addTab opens an empty request after the active one.
func (a *app) addTab(ws *workspace) {
	a.leaveTab(ws)
	ws.tabs = append(ws.tabs, nil)
	copy(ws.tabs[ws.active+2:], ws.tabs[ws.active+1:])
	ws.tabs[ws.active+1] = newTab()
	ws.active++
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
	dup.body.Set(src.body.String())

	a.leaveTab(ws)
	ws.tabs = append(ws.tabs, nil)
	copy(ws.tabs[i+2:], ws.tabs[i+1:])
	ws.tabs[i+1] = dup
	ws.active = i + 1
}

// closeTab closes the tab at i, keeping one empty tab when it was the last.
func (a *app) closeTab(ws *workspace, i int) {
	if i < 0 || i >= len(ws.tabs) {
		return
	}
	a.saveTab(ws.tabs[i])
	if len(ws.tabs) == 1 {
		ws.tabs[0] = newTab()
		ws.active = 0
		return
	}
	ws.tabs = append(ws.tabs[:i], ws.tabs[i+1:]...)
	if i < ws.active {
		ws.active--
	}
	if ws.active >= len(ws.tabs) {
		ws.active = len(ws.tabs) - 1
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
		}
	}
	ws.tabs = []*tab{ws.tabs[i]}
	ws.active = 0
}

// newCollection creates a collection with its default environments and
// opens it.
func (a *app) newCollection(name string) error {
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("集合 %d", len(a.store.collections)+1)
	}
	col, err := a.store.newCollection(name)
	if err != nil {
		return err
	}
	a.openCollection(col.ID)
	return nil
}

// openCollectionModal asks for the name of a new collection.
func (a *app) openCollectionModal() {
	a.collectionName = ""
	a.collectionOpen = true
}

// saveNewCollection creates the collection the dialog names.
func (a *app) saveNewCollection(c *ui.Context) {
	name := strings.TrimSpace(a.collectionName)
	if name == "" {
		return
	}
	if err := a.newCollection(name); err != nil {
		c.Toast("新建集合失败：" + err.Error())
		return
	}
	a.collectionOpen, a.collectionName = false, ""
}

// deleteCollection removes a collection, closing it when it is open.
func (a *app) deleteCollection(id string) error {
	if err := a.store.deleteCollection(id); err != nil {
		return err
	}
	if a.active == id || containsString(a.open, id) {
		// The collection is gone, so its workspace is dropped without writing
		// its tabs back.
		a.forgetCollection(id)
	}
	return nil
}

// openRequest opens a saved request in a new tab of its collection.
func (a *app) openRequest(collectionID string, r savedRequest) {
	a.openCollection(collectionID)
	ws := a.space()
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
	if r.BodyType != "" {
		t.bodyType = r.BodyType
		t.setBodyLanguage()
	}
	t.body.Set(r.Body)
	t.path = r.Path

	a.leaveTab(ws)
	ws.tabs = append(ws.tabs, nil)
	copy(ws.tabs[ws.active+2:], ws.tabs[ws.active+1:])
	ws.tabs[ws.active+1] = t
	ws.active++
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

// addRequest adds a request of the given kind to the active collection, in
// the folder folderPath names, and opens it.
func (a *app) addRequest(k kind, folderPath string) error {
	if a.active == "" {
		return errNoCollection
	}
	method := "GET"
	if k != kindREST {
		method = ""
	}
	req, err := a.store.addRequest(a.active, folderPath, "Untitled", savedRequest{
		Kind: k, Method: method, BodyType: "json",
	})
	if err != nil {
		return err
	}
	a.openRequest(a.active, req)
	return nil
}

// newFolder adds a folder to the 根目录 of the active collection's tree.
func (a *app) newFolder() error {
	col := a.store.collection(a.active)
	if col == nil {
		return errNoCollection
	}
	_, err := a.store.addFolder(a.active, "", fmt.Sprintf("分组 %d", len(col.Root.Folders)+1))
	return err
}

// saveCurrentToCollection writes the active request into a collection as an
// interface file of its own.
func (a *app) saveCurrentToCollection(collectionID string) error {
	t := a.current()
	if t == nil {
		return fmt.Errorf("没有激活的请求")
	}
	req, err := a.store.addRequest(collectionID, "", t.title, savedFromTab(t))
	if err != nil {
		return err
	}
	t.collectionID, t.title, t.path = collectionID, req.Name, req.Path
	return nil
}

// savedFromTab is the interface a tab's request stands for.
func savedFromTab(t *tab) savedRequest {
	return savedRequest{
		Kind:     t.kind,
		Method:   t.method,
		URL:      t.url,
		Params:   rowsToSaved(t.params.Rows),
		Headers:  rowsToSaved(t.headers.Rows),
		BodyType: t.bodyType,
		Body:     t.body.String(),
	}
}

// collectionIndex returns where a collection is kept, or -1.
func (a *app) collectionIndex(id string) int { return a.store.collectionIndex(id) }

// collectionEnvironmentList is the ordered scope of the environment menus:
// the active collection's environments.
func (a *app) collectionEnvironments() []environment {
	if a.active == "" {
		return nil
	}
	return a.store.environmentsFor(a.active)
}

// activeEnvironment is the environment the active collection sends with.
func (a *app) activeEnvironment() *environment {
	if a.active == "" {
		return nil
	}
	return a.store.activeEnvironment(a.active)
}

// openEnvManager shows the environments of the active collection.
func (a *app) openEnvManager() {
	a.envOpen = true
	envs := a.collectionEnvironments()
	if len(envs) == 0 {
		a.editEnvironment("")
		return
	}
	id := envs[0].ID
	if active := a.store.activeEnvironment(a.active); active != nil {
		id = active.ID
	}
	a.editEnvironment(id)
}

// editEnvironment loads an environment's variables for editing.
func (a *app) editEnvironment(id string) {
	a.envID = id
	env := a.store.environment(id)
	if env == nil {
		a.envName, a.envVars = "", widgets.State{}
		return
	}
	a.envName = env.Name
	a.envVars = widgets.State{Rows: rowsFromSaved(env.Vars), Compact: true}
	a.envVars.OnChange = func() { a.saveEnvironmentVars() }
}

// saveEnvironmentVars writes the edited variables back into the environment.
func (a *app) saveEnvironmentVars() {
	env := a.store.environment(a.envID)
	if env == nil {
		return
	}
	env.Vars = rowsToSaved(a.envVars.Rows)
	_ = a.store.saveEnvironments()
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

// palette returns the colors of this frame, noticing the desktop's
// appearance for as long as the view still follows it.
func (a *app) palette(c *ui.Context) design.Palette {
	if a.mode == design.System && a.applied == design.System {
		a.systemDark = c.Theme().Dark
	}
	return design.PaletteOf(a.resolvedMode())
}

// themeFor returns the theme the window should draw with, nil while it
// follows the desktop's own theme. In System it leaves the window's theme
// alone; picking System again after a fixed appearance keeps the desktop's
// last seen appearance for the rest of the session, as MyGo has no way to
// hand the window back to the desktop once SetTheme replaced its theme.
func (a *app) themeFor(c *ui.Context) *ui.Theme {
	if a.mode == design.System && (a.applied == design.Mode(-1) || a.applied == design.System) {
		a.applied = design.System
		a.systemDark = c.Theme().Dark
		return nil
	}
	target := a.resolvedMode()
	if a.theme == nil || a.applied != target {
		a.theme = design.Theme(target)
		a.applied = target
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
		c.Toast(kindName(t.kind) + " 请求尚未实现")
		return
	}
	if t.bodyType == "json" {
		if _, err := indentJSON(t.body.String()); err != nil {
			c.Toast("请求体不是合法 JSON：" + err.Error())
			return
		}
	}
	vars := a.store.varsFor(a.active)
	url, missing := resolveVars(t.url, vars)
	params, missingParams := resolveRows(t.params.Enabled(), vars)
	headers, missingHeaders := resolveRows(t.headers.Enabled(), vars)
	body, missingBody := resolveVars(t.body.String(), vars)
	missing = dedupe(append(append(append(missing, missingParams...), missingHeaders...), missingBody...))
	if len(missing) > 0 {
		c.Toast("未解析的环境变量：" + strings.Join(missing, "、"))
		return
	}
	req, err := buildRequest(requestSpec{
		method:  t.method,
		rawURL:  url,
		params:  params,
		headers: headers,
		body:    body,
	})
	if err != nil {
		c.Toast(err.Error())
		return
	}
	t.loading, t.resp = true, nil
	run := func() { a.apply(t, execute(context.Background(), req)) }
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
	_ = a.store.addHistory(historyEntry{
		Method: t.method,
		URL:    t.url,
		Status: res.status,
		At:     time.Now(),
		Title:  t.title,
		Kind:   t.kind,
	})
	if res.err != nil {
		return
	}
	t.respEdit.Buffer.Set(res.body)
	if isJSON(res.headers.Get("Content-Type"), res.body) {
		t.respEdit.Buffer.SetLanguage(editor.LanguageFor("response.json"))
	} else {
		t.respEdit.Buffer.SetLanguage(nil)
	}
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
		out = append(out, widgets.Row{Enabled: r.Enabled, Key: r.Key, Value: r.Value})
	}
	return append(out, widgets.Row{Enabled: true})
}

// rowsToSaved turns edited rows into stored ones, leaving out the empty
// rows that only give room to type.
func rowsToSaved(rows []widgets.Row) []savedRow {
	out := make([]savedRow, 0, len(rows))
	for _, r := range rows {
		if r.Key == "" && r.Value == "" {
			continue
		}
		out = append(out, savedRow{Enabled: r.Enabled, Key: r.Key, Value: r.Value})
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

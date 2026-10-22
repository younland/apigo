package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"apigo/internal/openapi"
	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// importState is the 导入项目 dialog: the way it reads the document (a file
// or a URL), what the last read said, and the document itself once it is
// parsed.
type importState struct {
	open bool
	// source is the tab of the dialog shown: the file of the desktop, or a URL.
	source int
	url    string
	// file is the path the document came from, "" when it came from a URL.
	file string
	// doc is the parsed document, nil until one is read.
	doc *openapi.Document
	// note says what the last read did, or what failed: the dialog shows it
	// under the source it came from.
	note string
	// busy says a document is on its way over the network.
	busy bool
	// details unfolds what the dialog makes of a document.
	details bool
}

// The sources the dialog reads a document from, in the order its tabs show
// them.
const (
	importFromFile = iota
	importFromURL
)

// importWidth is how wide the 导入项目 dialog is, importDropHeight how tall its
// drop area stands, and importFormatHeight the row the format is chosen in.
const (
	importWidth        float32 = 680
	importPadding      float32 = 20
	importGap          float32 = 12
	importDropHeight   float32 = 220
	importTabHeight    float32 = 36
	importFormatHeight float32 = 56
)

// importDocLimit caps the document read from a file, in bytes: a document is a
// description of interfaces, not a body to hold in memory.
const importDocLimit = 32 << 20

// openImportDialog shows the 导入项目 dialog, reading nothing until a document
// is chosen.
func (a *app) openImportDialog() {
	a.importDialog = importState{open: true}
}

// closeImportDialog puts the dialog away with what it read.
func (a *app) closeImportDialog() { a.importDialog = importState{} }

// chooseImportDoc asks the desktop for a document file and reads it.
func (a *app) chooseImportDoc(c *ui.Context) {
	if a.chooseDoc == nil {
		c.Toast("当前环境不能选择文件")
		return
	}
	path, err := a.chooseDoc()
	if err != nil {
		a.importDialog.note = "选择文件失败：" + err.Error()
		return
	}
	if path == "" {
		return // the dialog was cancelled
	}
	a.readImportFile(path)
}

// readImportFile reads a document from a file of the desktop, as the click of
// the drop area or a file dropped on it does.
func (a *app) readImportFile(path string) {
	info, err := os.Stat(path)
	if err != nil {
		a.importDialog.note = "读取文件失败：" + err.Error()
		return
	}
	if info.Size() > importDocLimit {
		a.importDialog.note = fmt.Sprintf("文件超过 %d MB，读不下", importDocLimit>>20)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		a.importDialog.note = "读取文件失败：" + err.Error()
		return
	}
	a.importDialog.note = ""
	a.parseImportDoc(data, path)
}

// fetchImportDoc reads the document a URL serves. The read reaches the
// network, so it runs off the window's thread and comes back to it.
func (a *app) fetchImportDoc() {
	url := strings.TrimSpace(a.importDialog.url)
	if url == "" || a.importDialog.busy {
		return
	}
	fetch := a.fetchDoc
	if fetch == nil {
		fetch = openapi.Fetch
	}
	if a.after == nil {
		// No window to hand the work to: the test reads it here.
		data, err := fetch(context.Background(), url)
		a.noteImportDoc(data, err, "")
		return
	}
	a.importDialog.busy, a.importDialog.note = true, "正在读取文档…"
	go func() {
		data, err := fetch(context.Background(), url)
		a.after(func() { a.noteImportDoc(data, err, "") })
	}()
}

// noteImportDoc keeps what reading a document came to: the parsed document, or
// why it could not be read.
func (a *app) noteImportDoc(data []byte, err error, source string) {
	a.importDialog.busy = false
	if err != nil {
		a.importDialog.doc, a.importDialog.note = nil, "读取文档失败："+err.Error()
		return
	}
	a.parseImportDoc(data, source)
}

// parseImportDoc reads the interfaces a document describes, keeping what it
// says for the dialog to show before anything is written.
func (a *app) parseImportDoc(data []byte, source string) {
	doc, err := openapi.Parse(data)
	if err != nil {
		a.importDialog.doc, a.importDialog.note = nil, err.Error()
		return
	}
	if doc.RequestCount() == 0 {
		a.importDialog.doc, a.importDialog.note = nil, "文档里没有可导入的接口"
		return
	}
	a.importDialog.doc, a.importDialog.file = doc, source
	a.importDialog.note = ""
}

// importTarget is the collection a parsed document becomes: the one of the
// document's title, and whether it is already there to be updated rather than
// made again.
func (a *app) importTarget() (string, bool) {
	if a.importDialog.doc == nil {
		return "", false
	}
	name := strings.TrimSpace(a.importDialog.doc.Title)
	for _, col := range a.store.Collections {
		if col.Name == name {
			return name, true
		}
	}
	return name, false
}

// applyImport writes the parsed document into its collection: the collection of
// the title is updated when it is there, made when it is not, and the dialog
// puts itself away on the collection it wrote.
func (a *app) applyImport(c *ui.Context) {
	doc := a.importDialog.doc
	if doc == nil {
		return
	}
	name, _ := a.importTarget()
	res, err := a.store.Import(store.ImportOptions{
		Name: name, BaseURL: doc.BaseURL, Root: importTree(doc),
	})
	if err != nil {
		a.importDialog.note = "导入失败：" + err.Error()
		return
	}
	a.closeImportDialog()
	c.Toast(importReport(res))
	a.openCollection(res.Collection.ID)
}

// importReport says what an import did, as the toast under the window reads it.
func importReport(res store.ImportResult) string {
	if res.Created {
		return fmt.Sprintf("已导入集合「%s」，%d 个接口", res.Collection.Name, res.Added)
	}
	return fmt.Sprintf("已更新集合「%s」：新增 %d 个接口，覆盖 %d 个",
		res.Collection.Name, res.Added, res.Updated)
}

// importTree turns a parsed document into the tree of a collection: the
// interfaces of its tags as folders, and the interfaces outside any tag in the
// 根目录.
func importTree(doc *openapi.Document) folder {
	root := folder{Name: store.RootFolderName}
	for _, req := range doc.Requests {
		root.Requests = append(root.Requests, importedRequest(req))
	}
	for _, group := range doc.Folders {
		sub := folder{Name: group.Name}
		for _, req := range group.Requests {
			sub.Requests = append(sub.Requests, importedRequest(req))
		}
		root.Folders = append(root.Folders, sub)
	}
	return root
}

// importedRequest is the interface one operation of a document becomes: a REST
// request with what the document says it takes and sends.
func importedRequest(req openapi.Request) savedRequest {
	return savedRequest{
		Kind:     kindREST,
		Name:     req.Name,
		Method:   req.Method,
		URL:      req.URL,
		Params:   rowsFromImport(req.Params),
		Headers:  rowsFromImport(req.Headers),
		BodyType: req.BodyType,
		Body:     req.Body,
		Form:     rowsFromImport(req.Form),
	}
}

// rowsFromImport turns the rows of a document into the rows an interface
// keeps, each one enabled: what the document suggests is what it sends.
func rowsFromImport(rows []openapi.Row) []savedRow {
	out := make([]savedRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, savedRow{Enabled: true, Key: row.Key, Value: row.Value})
	}
	return out
}

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// importDoc is a small OpenAPI document: one tagged operation, one untagged,
// and a server the collection takes as its BaseURL.
const importDoc = `{
  "openapi": "3.0.3",
  "info": {"title": "宠物商店", "version": "1.0.0"},
  "servers": [{"url": "https://api.example.com"}],
  "paths": {
    "/pets": {
      "get": {
        "summary": "列出宠物",
        "tags": ["宠物"],
        "parameters": [{"name": "limit", "in": "query", "schema": {"type": "integer", "default": 10}}]
      }
    },
    "/health": {"get": {"summary": "健康检查"}}
  }
}`

// importFile writes a document to the test's directory and returns its path.
func importFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doc.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportDialogOpens(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("导入文档"); err != nil {
		t.Fatal(err)
	}
	if !a.importDialog.open {
		t.Fatal("dialog closed after 导入文档")
	}
	for _, want := range []string{"导入项目", "OpenAPI/Swagger", "上传文件", "URL 导入",
		"点击或拖拽文件到本区域导入"} {
		if !tt.HasText(want) {
			t.Errorf("texts %q, want %s", tt.Texts(), want)
		}
	}
	if err := tt.Click("关闭导入"); err != nil {
		t.Fatal(err)
	}
	if a.importDialog.open {
		t.Error("dialog open after 关闭导入")
	}
}

func TestImportReadsAFile(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	if err := a.store.Load(); err != nil {
		t.Fatal(err)
	}
	a.chooseDoc = func() (string, error) { return importFile(t, importDoc), nil }
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("导入文档"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("选择文档文件"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("已读取 2 个接口") {
		t.Fatalf("texts %q, want the summary of the document", tt.Texts())
	}
	if err := tt.Click("导入"); err != nil {
		t.Fatal(err)
	}
	if a.importDialog.open {
		t.Error("dialog open after importing")
	}
	col := a.store.Collection("宠物商店")
	if col == nil {
		t.Fatalf("collections %+v, want the one of the document", a.store.Collections)
	}
	if a.active != col.ID {
		t.Errorf("active %q, want the imported collection opened", a.active)
	}
	if store.CountRequests(col.Root) != 2 {
		t.Fatalf("collection holds %d interfaces, want the two of the document", store.CountRequests(col.Root))
	}
	folder := store.FindFolder(&col.Root, "宠物")
	if folder == nil || len(folder.Requests) != 1 || folder.Requests[0].Method != "GET" {
		t.Fatalf("folder %+v, want the tagged interface", folder)
	}
	if url := folder.Requests[0].URL; url != "https://api.example.com/pets" {
		t.Errorf("url %q, want the path under the server", url)
	}
	if params := folder.Requests[0].Params; len(params) != 1 || params[0].Key != "limit" {
		t.Errorf("params %+v, want the query parameter", params)
	}
	// The server of the document is the BaseURL of the environment it opens on.
	envs := a.store.EnvironmentsFor(col.ID)
	if len(envs) == 0 || envs[0].Vars[0].Value != "https://api.example.com" {
		t.Errorf("environments %+v, want the server as the BaseURL", envs)
	}
	// An imported interface opens as a request of its own, parameters and all.
	req, ok := findRequest(col.Root, "列出宠物")
	if !ok {
		t.Fatalf("collection %+v, want the interface the document names", col.Root)
	}
	a.openRequest(col.ID, req)
	if open := a.current(); open == nil || open.url != "https://api.example.com/pets" ||
		len(open.params.Rows) == 0 || open.params.Rows[0].Key != "limit" {
		t.Errorf("tab %+v, want the imported request opened", open)
	}
}

func TestImportReadsAURL(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	if err := a.store.Load(); err != nil {
		t.Fatal(err)
	}
	asked := ""
	a.fetchDoc = func(_ context.Context, url string) ([]byte, error) {
		asked = url
		return []byte(importDoc), nil
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("导入文档"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("导入方式 URL 导入"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("文档 URL"); err != nil {
		t.Fatal(err)
	}
	tt.Type("https://api.example.com/openapi.json")
	if err := tt.Click("读取"); err != nil {
		t.Fatal(err)
	}
	if asked != "https://api.example.com/openapi.json" {
		t.Fatalf("read %q, want the url of the field", asked)
	}
	if !tt.HasText("已读取 2 个接口") {
		t.Fatalf("texts %q, want the summary of the document", tt.Texts())
	}
	if err := tt.Click("导入"); err != nil {
		t.Fatal(err)
	}
	if a.store.Collection("宠物商店") == nil {
		t.Errorf("collections %+v, want the one of the document", a.store.Collections)
	}
}

func TestImportSaysWhatFailed(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	if err := a.store.Load(); err != nil {
		t.Fatal(err)
	}
	a.fetchDoc = func(context.Context, string) ([]byte, error) {
		return nil, errors.New("连不上")
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("导入文档"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("导入方式 URL 导入"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("文档 URL"); err != nil {
		t.Fatal(err)
	}
	tt.Type("https://api.example.com/openapi.json")
	if err := tt.Click("读取"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("连不上") {
		t.Errorf("texts %q, want the failure the read hit", tt.Texts())
	}
	// A file that is no document is refused the same way.
	a.readImportFile(importFile(t, "hello: world"))
	tt.Frame()
	if a.importDialog.doc != nil {
		t.Error("document parsed from what is no document")
	}
	if !tt.HasText("不是 OpenAPI 或 Swagger 文档") {
		t.Errorf("texts %q, want what the file is missing", tt.Texts())
	}
	if len(a.store.Collections) != 0 {
		t.Errorf("collections %+v, want none written", a.store.Collections)
	}
}

func TestImportUpdatesTheCollectionOfTheTitle(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	if err := a.store.Load(); err != nil {
		t.Fatal(err)
	}
	col, err := a.store.NewCollection("宠物商店")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.store.AddFolder(col.ID, "", "宠物"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.store.AddRequest(col.ID, "宠物", "列出宠物", savedRequest{
		Kind: kindREST, Method: "GET", URL: "http://localhost/pets",
	}); err != nil {
		t.Fatal(err)
	}
	a.openImportDialog()
	a.readImportFile(importFile(t, importDoc))
	tt := ui.NewTester(a.view, 1280, 800)
	if !tt.HasText("已有同名集合") {
		t.Errorf("texts %q, want the note about the collection being updated", tt.Texts())
	}
	if err := tt.Click("导入"); err != nil {
		t.Fatal(err)
	}

	if len(a.store.Collections) != 1 {
		t.Fatalf("collections %+v, want the one the document updates", a.store.Collections)
	}
	updated := a.store.Collection(col.ID)
	if updated == nil || store.CountRequests(updated.Root) != 2 {
		t.Fatalf("collection %+v, want the interface written over and the one added", updated)
	}
	if url := store.FindFolder(&updated.Root, "宠物").Requests[0].URL; !strings.HasPrefix(url, "https://api.example.com") {
		t.Errorf("url %q, want the url of the document", url)
	}
}

func TestImportTakesADroppedFile(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	if err := a.store.Load(); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("导入文档"); err != nil {
		t.Fatal(err)
	}
	zone, ok := tt.Find("选择文档文件")
	if !ok {
		t.Fatalf("texts %q, want the area a file is dropped on", tt.Texts())
	}
	if !tt.DropFiles(zone.X+zone.W/2, zone.Y+zone.H/2, importFile(t, importDoc)) {
		t.Fatal("the drop area took no files")
	}
	if !tt.HasText("已读取 2 个接口") {
		t.Fatalf("texts %q, want the document dropped on the area", tt.Texts())
	}
}

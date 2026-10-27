package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apigo/internal/openapi"
	"apigo/internal/widgets"

	"github.com/egoist/mygo/ui"
)

// TestParamsPaneSplitsQueryAndPath checks the 参数 pane holds the query table, and
// the path table only when the address takes a path parameter, and that a row is
// added and deleted from the query one.
func TestParamsPaneSplitsQueryAndPath(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	// An address of no {name} takes no path parameter: no path table at all.
	for _, want := range []string{"Query 参数", "参数名", "参数值", "说明"} {
		if !tt.HasText(want) {
			t.Errorf("texts %q, want %q in the parameters pane", tt.Texts(), want)
		}
	}
	if tt.HasText("Path 参数") {
		t.Errorf("texts %q, want no path table for an address that takes none", tt.Texts())
	}

	a.current().url = "https://api.example.com/pets/{petId}"
	tt.Frame()
	if !tt.HasText("Path 参数") {
		t.Fatalf("texts %q, want the path table once the address takes a name", tt.Texts())
	}
	// The path name reads as an input too, set to what the address says.
	if _, ok := tt.Find("路径参数名第 1 行"); !ok {
		t.Errorf("texts %q, want the path name in a field of its own", tt.Texts())
	}

	if rows := a.current().params.Rows; len(rows) != 1 || rows[0].Key != "" {
		t.Fatalf("query rows %+v, want the one empty row a table starts with", rows)
	}
	if err := tt.Click("添加参数"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if rows := a.current().params.Rows; len(rows) != 2 {
		t.Errorf("query rows %+v, want the new row beside the empty one", rows)
	}
	if err := tt.Click("删除参数第 1 行"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if rows := a.current().params.Rows; len(rows) != 1 {
		t.Errorf("query rows %+v after the delete, want the empty one alone", rows)
	}
}

// TestPathParamsFollowTheAddress checks the path table is the names the address
// takes: a row comes with a placeholder, keeps its value as long as the name is
// there, and goes when the address loses it.
func TestPathParamsFollowTheAddress(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	a.current().url = "/pets/{petId}"
	tt.Frame()
	rows := a.current().pathParams.Rows
	if len(rows) != 1 || rows[0].Key != "petId" {
		t.Fatalf("path rows %+v, want the name the address takes", rows)
	}
	if _, ok := tt.Find("路径参数名第 1 行"); !ok {
		t.Errorf("texts %q, want the path parameter named", tt.Texts())
	}

	a.current().pathParams.Rows[0].Value = "42"
	a.current().url = "/pets/{petId}/toys/{toyId}"
	tt.Frame()
	rows = a.current().pathParams.Rows
	if len(rows) != 2 || rows[0].Key != "petId" || rows[1].Key != "toyId" {
		t.Fatalf("path rows %+v, want both names of the address", rows)
	}
	if rows[0].Value != "42" {
		t.Errorf("value %q, want the one the row already held", rows[0].Value)
	}

	a.current().url = "/pets"
	tt.Frame()
	if rows := a.current().pathParams.Rows; len(rows) != 0 {
		t.Errorf("path rows %+v, want none once the address takes none", rows)
	}
	// With nothing for it to show, the path table goes with its rows.
	if tt.HasText("Path 参数") {
		t.Errorf("texts %q, want no path table once the address takes no name", tt.Texts())
	}
}

// TestSendFillsPathParameters checks a path parameter is filled into the address
// a request goes out to, and that an empty one refuses the request.
func TestSendFillsPathParameters(t *testing.T) {
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := testApp(t)
	a.current().url = server.URL + "/pets/{petId}"
	a.current().pathParams.Rows = []widgets.Param{{Enabled: true, Key: "petId", Value: "7"}}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if path := <-received; path != "/pets/7" {
		t.Errorf("path %q, want the value filled into the address", path)
	}

	// The same request without the value says what it is missing.
	a.current().pathParams.Rows = []widgets.Param{{Enabled: true, Key: "petId"}}
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("路径参数未填：petId") {
		t.Errorf("texts %q, want the missing path parameter reported", tt.Texts())
	}
}

// TestPathParametersAreSaved checks the interface file keeps the path
// parameters, their type and what they are for, beside the query ones.
func TestPathParametersAreSaved(t *testing.T) {
	a := testApp(t)
	a.current().url = "https://api.example.com/pets/{petId}"
	a.current().pathParams.Rows = []widgets.Param{{
		Enabled: true, Key: "petId", Value: "7", Description: "宠物 ID",
	}}
	a.current().params.Rows = []widgets.Param{{
		Enabled: true, Key: "page", Value: "2", Description: "第几页",
	}}

	saved := savedFromTab(a.current())
	if len(saved.PathParams) != 1 || saved.PathParams[0].Value != "7" {
		t.Fatalf("path params %+v, want the one the tab holds", saved.PathParams)
	}
	if len(saved.Params) != 1 || saved.Params[0].Description != "第几页" {
		t.Fatalf("params %+v, want the description kept beside the value", saved.Params)
	}

	// The file keeps both tables, and the interface reads them back.
	req, err := a.store.AddRequest("Demo", "", "宠物详情", saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.store.ReloadCollection("Demo"); err != nil {
		t.Fatal(err)
	}
	found, ok := findRequest(a.store.Collection("Demo").Root, req.Name)
	if !ok {
		t.Fatalf("no interface %q after the reload", req.Name)
	}
	if len(found.PathParams) != 1 || found.PathParams[0].Value != "7" {
		t.Errorf("path params %+v, want them read back from the file", found.PathParams)
	}
	if len(found.Params) != 1 || found.Params[0].Description != "第几页" {
		t.Errorf("params %+v, want the description read back from the file", found.Params)
	}
}

// TestImportedRequestKeepsPathParameters checks an imported document's path
// parameters become the values its address takes.
func TestImportedRequestKeepsPathParameters(t *testing.T) {
	req := importedRequest(openapi.Request{
		Name: "宠物详情", Method: "GET", URL: "https://api.example.com/pets/{petId}",
		Path: []openapi.Row{{Key: "petId", Description: "宠物 ID"}},
	})
	if len(req.PathParams) != 1 {
		t.Fatalf("path params %+v, want the one of the operation", req.PathParams)
	}
	if got := req.PathParams[0]; got.Key != "petId" || got.Description != "宠物 ID" || !got.Enabled {
		t.Errorf("path param %+v, want the name and description of the document", got)
	}
	if !strings.Contains(req.URL, "{petId}") {
		t.Errorf("url %q, want the placeholder the value fills", req.URL)
	}
}

// TestResponseNamedAfterTheFilledAddress checks a response file is named after
// the address the request went to, its path parameters filled in rather than
// left as placeholders.
func TestResponseNamedAfterTheFilledAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("not really a video"))
	}))
	defer server.Close()

	a := testApp(t)
	a.current().url = server.URL + "/media/{uuid}.mp4"
	a.current().pathParams.Rows = []widgets.Param{
		{Enabled: true, Key: "uuid", Value: "ce19zr9qs9"},
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("ce19zr9qs9.mp4") {
		t.Errorf("texts %q, want the file named after the filled address", tt.Texts())
	}
	if tt.HasText("{uuid}.mp4") {
		t.Errorf("texts %q, want no placeholder in the name", tt.Texts())
	}
	if got := downloadName(a.current().resp, a.current().sentAddress()); got != "ce19zr9qs9.mp4" {
		t.Errorf("download name %q, want the path filled in", got)
	}
}

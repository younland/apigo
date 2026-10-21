package app

import (
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apigo/internal/store"
	"apigo/internal/widgets"

	"github.com/egoist/mygo/ui"
)

// sentRequest is one request as the server read it.
type sentRequest struct {
	contentType string
	body        string
	form        map[string]string
	// files are the file fields of a multipart form by field name: the name
	// the file was sent with and its bytes.
	files     map[string]string
	fileBytes map[string]string
}

// bodyServer starts a server that reports what a send brought, and returns it
// with the channel those reports come on.
func bodyServer(t *testing.T) (*httptest.Server, chan sentRequest) {
	t.Helper()
	got := make(chan sentRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sent sentRequest
		sent.contentType = r.Header.Get("Content-Type")
		if strings.HasPrefix(sent.contentType, "multipart/form-data") {
			if err := r.ParseMultipartForm(1 << 20); err == nil {
				sent.form = map[string]string{}
				for key, values := range r.MultipartForm.Value {
					if len(values) > 0 {
						sent.form[key] = values[0]
					}
				}
				sent.files, sent.fileBytes = map[string]string{}, map[string]string{}
				for key, headers := range r.MultipartForm.File {
					if len(headers) == 0 {
						continue
					}
					sent.files[key] = headers[0].Filename
					if f, err := headers[0].Open(); err == nil {
						data, _ := io.ReadAll(f)
						sent.fileBytes[key] = string(data)
						f.Close()
					}
				}
			}
		} else {
			body, _ := io.ReadAll(r.Body)
			sent.body = string(body)
		}
		got <- sent
		io.WriteString(w, "{}")
	}))
	t.Cleanup(server.Close)
	return server, got
}

// TestSendBodyFormats checks each format sends the body its Content-Type says,
// and that those without a body of their own send none.
func TestSendBodyFormats(t *testing.T) {
	cases := []struct {
		format  string
		body    string
		ctype   string
		payload string
	}{
		{"json", `{"name":"Ada"}`, "application/json", `{"name":"Ada"}`},
		{"xml", "<note/>", "application/xml", "<note/>"},
		{"text", "hello", "text/plain", "hello"},
		{"graphql", `{"query":"{ pets { id } }"}`, "application/json", `{"query":"{ pets { id } }"}`},
		{"none", "ignored", "", ""},
		{"binary", "ignored", "", ""},
		{"msgpack", "ignored", "", ""},
	}
	for _, c := range cases {
		t.Run(c.format, func(t *testing.T) {
			server, got := bodyServer(t)
			a := testApp(t)
			a.current().url = server.URL
			a.current().method = "POST"
			a.current().bodyType = c.format
			a.current().body.Set(c.body)
			tt := ui.NewTester(a.view, 1280, 800)

			if err := tt.Click("发送"); err != nil {
				t.Fatal(err)
			}
			sent := <-got
			if sent.contentType != c.ctype {
				t.Errorf("Content-Type %q, want %q", sent.contentType, c.ctype)
			}
			if sent.body != c.payload {
				t.Errorf("body %q, want %q", sent.body, c.payload)
			}
		})
	}
}

// TestSendURLEncodedForm checks the x-www-form-urlencoded rows reach the server
// encoded, and that a disabled row stays behind.
func TestSendURLEncodedForm(t *testing.T) {
	server, got := bodyServer(t)
	a := testApp(t)
	a.current().url = server.URL
	a.current().method = "POST"
	a.current().bodyType = "urlencoded"
	a.current().form.Rows = []widgets.Row{
		{Enabled: true, Key: "name", Value: "Ada Lovelace"},
		{Enabled: false, Key: "debug", Value: "1"},
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	sent := <-got
	if want := "application/x-www-form-urlencoded"; sent.contentType != want {
		t.Errorf("Content-Type %q, want %q", sent.contentType, want)
	}
	if want := "name=Ada+Lovelace"; sent.body != want {
		t.Errorf("body %q, want %q", sent.body, want)
	}
}

// TestSendMultipartForm checks the form-data rows reach the server as a
// multipart form with its own boundary.
func TestSendMultipartForm(t *testing.T) {
	server, got := bodyServer(t)
	a := testApp(t)
	a.current().url = server.URL
	a.current().method = "POST"
	a.current().bodyType = "form-data"
	a.current().form.Rows = []widgets.Row{{Enabled: true, Key: "name", Value: "Ada"}}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	sent := <-got
	mediaType, params, err := mime.ParseMediaType(sent.contentType)
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("Content-Type %q, want a multipart form", sent.contentType)
	}
	if params["boundary"] == "" {
		t.Errorf("Content-Type %q, want the boundary the form was written with", sent.contentType)
	}
	if got := sent.form["name"]; got != "Ada" {
		t.Errorf("form field name %q, want Ada", got)
	}
}

// TestBodyPaneOffersTheFormats checks the format row shows every format, that
// the chosen one fills the pane the way its format holds a body, and that the
// Content-Type beside it follows.
func TestBodyPaneOffersTheFormats(t *testing.T) {
	a := testApp(t)
	a.current().reqTab = 2
	tt := ui.NewTester(a.view, 1280, 800)

	for _, format := range bodyFormats {
		if _, ok := tt.Find("请求体格式 " + format.Label); !ok {
			t.Errorf("no pill for %q", format.Label)
		}
	}

	if err := tt.Click("请求体格式 form-data"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("multipart/form-data") || !tt.HasText("选择文件第 1 行") {
		t.Errorf("texts %q, want the form rows and their Content-Type", tt.Texts())
	}
	if err := tt.Click("请求体格式 Binary"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("暂未实现") {
		t.Errorf("texts %q, want what Binary waits for", tt.Texts())
	}
	if err := tt.Click("请求体格式 none"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("该请求不发送请求体") || tt.HasText("application/json") {
		t.Errorf("texts %q, want the empty body of none and no Content-Type", tt.Texts())
	}
}

// TestFormBodyRoundTrips checks a saved interface keeps the rows of an encoded
// body, so opening it brings them back.
func TestFormBodyRoundTrips(t *testing.T) {
	a := testApp(t)
	cur := a.current()
	cur.title = "表单"
	cur.bodyType = "urlencoded"
	cur.form.Rows = []widgets.Row{{Enabled: true, Key: "name", Value: "Ada"}}
	if err := a.saveCurrentToCollection(a.active); err != nil {
		t.Fatal(err)
	}

	reopened := store.New(a.store.Dir)
	if err := reopened.Load(); err != nil {
		t.Fatal(err)
	}
	col := reopened.Collection(a.active)
	if col == nil {
		t.Fatal("no collection after the reload")
	}
	req, ok := findRequest(col.Root, "表单")
	if !ok {
		t.Fatalf("no interface 表单 in %+v", col.Root)
	}
	if req.BodyType != "urlencoded" || len(req.Form) != 1 || req.Form[0].Key != "name" {
		t.Errorf("saved %+v, want the encoded body rows", req)
	}
}

// findRequest looks a request up by name in a folder and the ones inside it.
func findRequest(f folder, name string) (savedRequest, bool) {
	for _, req := range f.Requests {
		if req.Name == name {
			return req, true
		}
	}
	for _, sub := range f.Folders {
		if req, ok := findRequest(sub, name); ok {
			return req, true
		}
	}
	return savedRequest{}, false
}

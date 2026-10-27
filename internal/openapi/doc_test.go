package openapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// petDocument is an OpenAPI 3 document with a tagged operation, an untagged
// one, a body that points at its components, and the parameters of both panes.
const petDocument = `{
  "openapi": "3.0.3",
  "info": {"title": "宠物商店", "version": "1.0.0"},
  "servers": [{"url": "https://api.example.com/v1/"}],
  "paths": {
    "/pets": {
      "get": {
        "summary": "列出宠物",
        "tags": ["宠物"],
        "parameters": [
          {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 10}},
          {"name": "X-Trace", "in": "header", "schema": {"type": "string", "example": "abc"}},
          {"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "responses": {"200": {"description": "ok"}}
      },
      "post": {
        "summary": "新建宠物",
        "tags": ["宠物"],
        "requestBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Pet"}}}}
      }
    },
    "/health": {"get": {"operationId": "health"}}
  },
  "components": {
    "schemas": {
      "Pet": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "age": {"type": "integer"},
          "tags": {"type": "array", "items": {"type": "string"}},
          "owner": {"$ref": "#/components/schemas/Owner"}
        }
      },
      "Owner": {"type": "object", "properties": {"email": {"type": "string", "format": "email"}}}
    }
  }
}`

func TestParseV3(t *testing.T) {
	out, err := Parse([]byte(petDocument))
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "宠物商店" || out.Spec != "3.0.3" || out.APIVersion != "1.0.0" {
		t.Errorf("document %+v, want the info of the document", out)
	}
	if out.BaseURL != "https://api.example.com/v1" {
		t.Errorf("base url %q, want the server without its trailing slash", out.BaseURL)
	}
	if out.RequestCount() != 3 || out.FolderCount() != 1 {
		t.Fatalf("counts %d/%d, want 3 interfaces in 1 folder", out.RequestCount(), out.FolderCount())
	}
	if name := out.Folders[0].Name; name != "宠物" {
		t.Errorf("folder %q, want the tag of its operations", name)
	}
	list, create := out.Folders[0].Requests[0], out.Folders[0].Requests[1]
	if list.Name != "列出宠物" || list.Method != "GET" {
		t.Errorf("request %+v, want the summary and method of the operation", list)
	}
	if list.URL != "https://api.example.com/v1/pets" {
		t.Errorf("url %q, want the path under the server", list.URL)
	}
	if len(list.Params) != 1 || list.Params[0] != (Row{Key: "limit", Value: "10"}) {
		t.Errorf("params %+v, want the query parameter with its default", list.Params)
	}
	if len(list.Path) != 1 || list.Path[0] != (Row{Key: "id"}) {
		t.Errorf("path params %+v, want the name the path takes", list.Path)
	}
	if len(list.Headers) != 1 || list.Headers[0] != (Row{Key: "X-Trace", Value: "abc"}) {
		t.Errorf("headers %+v, want the header parameter with its example", list.Headers)
	}
	if create.BodyType != "json" {
		t.Errorf("body type %q, want json", create.BodyType)
	}
	for _, want := range []string{`"name": "string"`, `"age": 0`, `"string"`, `"email": "user@example.com"`} {
		if !strings.Contains(create.Body, want) {
			t.Errorf("body %q, want it to hold %s", create.Body, want)
		}
	}
	health := out.Requests[0]
	if health.Name != "health" || health.URL != "https://api.example.com/v1/health" {
		t.Errorf("untagged request %+v, want it in the root with the server's url", health)
	}
}

// orderDocument is a Swagger 2 document: its server written apart, its form
// body as formData parameters, and no tags at all.
const orderDocument = `
swagger: "2.0"
info:
  title: 订单服务
  version: "1.2.0"
host: api.shop.test
basePath: /api
schemes: [http]
paths:
  /orders:
    get:
      summary: 订单列表
      parameters:
        - name: page
          in: query
          type: integer
          default: 1
    post:
      summary: 新建订单
      consumes: ["application/x-www-form-urlencoded"]
      parameters:
        - name: sku
          in: formData
          type: string
          default: "A-1"
`

func TestParseV2(t *testing.T) {
	out, err := Parse([]byte(orderDocument))
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "订单服务" || out.Spec != "2.0" || out.FolderCount() != 0 {
		t.Fatalf("document %+v, want the info of a v2 document with no tags", out)
	}
	if out.BaseURL != "http://api.shop.test/api" {
		t.Errorf("base url %q, want the scheme, host and base path", out.BaseURL)
	}
	get, post := out.Requests[0], out.Requests[1]
	if get.URL != "http://api.shop.test/api/orders" {
		t.Errorf("url %q, want the path under the base path", get.URL)
	}
	if len(get.Params) != 1 || get.Params[0] != (Row{Key: "page", Value: "1"}) {
		t.Errorf("params %+v, want the query parameter of a Swagger document", get.Params)
	}
	if post.BodyType != "urlencoded" || len(post.Form) != 1 {
		t.Fatalf("body %q %+v, want the form of the operation", post.BodyType, post.Form)
	}
	if post.Form[0] != (Row{Key: "sku", Value: "A-1"}) {
		t.Errorf("form %+v, want the formData parameter", post.Form)
	}
}

func TestParseRefusesWhatIsNotADocument(t *testing.T) {
	for _, data := range []string{"", "hello: world", `{"openapi": 3}`, ": not yaml :"} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("Parse(%q) succeeded, want an error", data)
		}
	}
}

func TestParseWithoutServerKeepsThePath(t *testing.T) {
	out, err := Parse([]byte(`{"openapi": "3.1.0", "info": {"title": "x"}, "paths": {"/a": {"get": {}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if out.BaseURL != "" {
		t.Errorf("base url %q, want none", out.BaseURL)
	}
	if url := out.Requests[0].URL; url != "/a" {
		t.Errorf("url %q, want the path the baseurl follows", url)
	}
}

func TestParseRelativeServerKeepsThePath(t *testing.T) {
	out, err := Parse([]byte(`{"openapi": "3.0.0", "info": {"title": "x"},
		"servers": [{"url": "/api/v3"}], "paths": {"/pet": {"get": {}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	// A server that is a bare path says nothing of the host: the environment
	// holds it, and the path follows the variable.
	if out.BaseURL != "" {
		t.Errorf("base url %q, want none for a relative server", out.BaseURL)
	}
	if url := out.Requests[0].URL; url != "/api/v3/pet" {
		t.Errorf("url %q, want the path under the baseurl", url)
	}
}

func TestParseNamesEveryInterfaceOfAGroupDifferently(t *testing.T) {
	out, err := Parse([]byte(`{"openapi": "3.0.0", "info": {"title": "x"}, "paths": {
		"/a": {"get": {"summary": "同一个"}, "post": {"summary": "同一个"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Requests[1].Name; got != "同一个 2" {
		t.Errorf("second name %q, want the first one numbered", got)
	}
}

func TestFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/doc.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(petDocument))
	}))
	defer server.Close()

	data, err := Fetch(context.Background(), server.URL+"/doc.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(data); err != nil {
		t.Fatalf("parsing what the server serves: %v", err)
	}
	if _, err := Fetch(context.Background(), server.URL+"/missing.json"); err == nil {
		t.Error("fetching a missing document succeeded, want an error")
	}
	if _, err := Fetch(context.Background(), "file:///etc/passwd"); err == nil {
		t.Error("fetching a file url succeeded, want an error")
	}
}

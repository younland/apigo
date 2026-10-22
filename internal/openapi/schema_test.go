package openapi

import (
	"strings"
	"testing"
)

// schemaDocument is a document whose examples need a little of everything:
// an enum, a format, an allOf, a schema that reaches itself, and a body that
// names no media type but JSON.
const schemaDocument = `{
  "openapi": "3.0.0",
  "info": {"title": "示例"},
  "paths": {
    "/things": {
      "post": {
        "requestBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Thing"}}}}
      }
    },
    "/text": {
      "post": {
        "requestBody": {"content": {"text/plain": {"example": "hello"}}}
      }
    },
    "/form": {
      "post": {
        "requestBody": {"content": {"multipart/form-data": {"schema": {
          "type": "object",
          "properties": {"file": {"type": "string", "format": "binary"}, "note": {"type": "string", "default": "x"}}
        }}}}
      }
    }
  },
  "components": {
    "schemas": {
      "Thing": {
        "allOf": [
          {"type": "object", "properties": {"kind": {"type": "string", "enum": ["a", "b"]}}},
          {"type": "object", "properties": {
            "when": {"type": "string", "format": "date-time"},
            "child": {"$ref": "#/components/schemas/Thing"},
            "ratio": {"type": "number"},
            "ok": {"type": "boolean"}
          }}
        ]
      }
    }
  }
}`

func TestExamples(t *testing.T) {
	out, err := Parse([]byte(schemaDocument))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Requests) != 3 {
		t.Fatalf("requests %+v, want the three operations", out.Requests)
	}
	body := byPath(t, out, "/things").Body
	for _, want := range []string{
		`"kind": "a"`, `"when": "2024-01-01T00:00:00Z"`,
		`"ratio": 0`, `"ok": false`, `"child": {`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body %q, want it to hold %s", body, want)
		}
	}
	// The schema reaches itself through child: the expansion stops there
	// rather than reading forever, a few levels in.
	if depth := strings.Count(body, `"child"`); depth < 2 || depth >= exampleDepth {
		t.Errorf("body holds %d levels of child, want a few and fewer than %d", depth, exampleDepth)
	}
	if text := byPath(t, out, "/text"); text.BodyType != "text" || text.Body != "hello" {
		t.Errorf("text body %q %q, want the example it gives", text.BodyType, text.Body)
	}
	form := byPath(t, out, "/form")
	if form.BodyType != "form-data" || len(form.Form) != 2 {
		t.Fatalf("form %q %+v, want the fields of the schema", form.BodyType, form.Form)
	}
	if form.Form[1] != (Row{Key: "note", Value: "x"}) {
		t.Errorf("form %+v, want the default of the field", form.Form)
	}
}

// byPath finds the interface a path of the document became.
func byPath(t *testing.T, out *Document, path string) Request {
	t.Helper()
	for _, req := range out.Requests {
		if strings.HasSuffix(req.URL, path) {
			return req
		}
	}
	t.Fatalf("no interface for %s among %+v", path, out.Requests)
	return Request{}
}

func TestExampleLeavesOutWhatItCannotDescribe(t *testing.T) {
	root := map[string]any{"openapi": "3.0.0", "info": map[string]any{"title": "x"}}
	d := &doc{root: node(root), v3: true, refs: map[string]bool{}}
	if value := d.example(node(map[string]any{"$ref": "#/components/schemas/Missing"}), 0); value != nil {
		t.Errorf("example %v, want nothing for a reference that points nowhere", value)
	}
	if text := marshalIndented(nil); text != "" {
		t.Errorf("marshalIndented(nil) = %q, want nothing", text)
	}
}

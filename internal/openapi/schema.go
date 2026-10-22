package openapi

import (
	"encoding/json"
	"strings"
)

// exampleDepth caps how deep an example digs into a schema: nested objects read
// as deep enough, and a document that nests them deeply makes no body of
// thousands of lines.
const exampleDepth = 6

// example reads what a schema says a value looks like: the example it gives, its
// default, the first of its enum, or a value of the type it describes.
func (d *doc) example(schema node, depth int) any {
	if schema == nil || depth > exampleDepth {
		return nil
	}
	if ref := schema.text("$ref"); ref != "" {
		if d.refs[ref] {
			return nil // a schema that reaches itself again reads as nothing there
		}
		d.refs[ref] = true
		defer delete(d.refs, ref)
		return d.example(d.lookup(ref), depth+1)
	}
	for _, key := range []string{"example", "default"} {
		if value, ok := schema[key]; ok && value != nil {
			return value
		}
	}
	if enum := schema.list("enum"); len(enum) > 0 {
		return enum[0]
	}
	if combined := d.combined(schema, depth); combined != nil {
		return combined
	}
	return d.typed(schema, depth)
}

// combined reads the schemas a schema is made of: an allOf as one object of all
// its parts, a oneOf or an anyOf as its first choice.
func (d *doc) combined(schema node, depth int) any {
	if parts := d.nodesOf(schema, "allOf"); len(parts) > 0 {
		merged := map[string]any{}
		for _, part := range parts {
			if value, ok := d.example(part, depth+1).(map[string]any); ok {
				for key, item := range value {
					merged[key] = item
				}
			}
		}
		return merged
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		if parts := d.nodesOf(schema, key); len(parts) > 0 {
			return d.example(parts[0], depth+1)
		}
	}
	return nil
}

// typed reads a value of the type a schema describes, its properties one by one
// for an object and its items once for an array.
func (d *doc) typed(schema node, depth int) any {
	switch schema.text("type") {
	case "object", "":
		if props := schema.obj("properties"); props != nil {
			return d.objectExample(props, depth)
		}
		if schema.text("type") == "" {
			return nil
		}
		return map[string]any{}
	case "array":
		if item := d.example(d.deref(schema.obj("items")), depth+1); item != nil {
			return []any{item}
		}
		return []any{}
	case "string":
		return sampleString(schema)
	case "integer":
		return 0
	case "number":
		return 0
	case "boolean":
		return false
	}
	return nil
}

// objectExample writes the properties of a document's object as they are
// declared: a property whose value is nothing at all is left out.
func (d *doc) objectExample(props node, depth int) map[string]any {
	out := make(map[string]any, len(props))
	for _, name := range sortedKeys(props) {
		if value := d.example(d.deref(props.obj(name)), depth+1); value != nil {
			out[name] = value
		}
	}
	return out
}

// sampleString writes a string of the format a schema names: a date reads as a
// date, a mail as a mail.
func sampleString(schema node) string {
	switch schema.text("format") {
	case "date-time":
		return "2024-01-01T00:00:00Z"
	case "date":
		return "2024-01-01"
	case "time":
		return "00:00:00"
	case "uuid":
		return "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	case "email":
		return "user@example.com"
	case "uri", "url":
		return "https://example.com"
	case "hostname":
		return "example.com"
	case "ipv4":
		return "127.0.0.1"
	case "ipv6":
		return "::1"
	case "byte":
		return "c3RyaW5n"
	case "password":
		return "password"
	}
	return "string"
}

// marshalIndented writes a value as the indented JSON a body reads as, "" for a
// value there is nothing to write.
func marshalIndented(value any) string {
	if value == nil {
		return ""
	}
	text, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	if string(text) == "null" {
		return ""
	}
	return strings.TrimSpace(string(text))
}

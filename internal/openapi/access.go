package openapi

import (
	"sort"
	"strconv"
	"strings"
)

// node is one mapping of a document, as the decoder reads an object into.
type node map[string]any

// newNode reads a value as a mapping, nil when it is not one.
func newNode(v any) node {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return node(m)
}

// text reads a field as text: a number or a flag reads as it is written.
func (n node) text(key string) string {
	switch v := n[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}
	return ""
}

// obj reads a field as a mapping.
func (n node) obj(key string) node { return newNode(n[key]) }

// list reads a field as a list of values.
func (n node) list(key string) []any {
	items, _ := n[key].([]any)
	return items
}

// nodes reads a field as a list of mappings, leaving out what is not one.
func (n node) nodes(key string) []node {
	var out []node
	for _, item := range n.list(key) {
		if m := newNode(item); m != nil {
			out = append(out, m)
		}
	}
	return out
}

// has reports whether a field is there and holds something.
func (n node) has(key string) bool {
	v, ok := n[key]
	return ok && v != nil
}

// sortedKeys lists the fields of a mapping in their order, so a document reads
// the same way every time it is imported.
func sortedKeys(n node) []string {
	keys := make([]string, 0, len(n))
	for key := range n {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// lookup reads what a reference of the document points at, nil when it points
// outside the document or at nothing.
func (d *doc) lookup(ref string) node {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var at any = map[string]any(d.root)
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		switch v := at.(type) {
		case map[string]any:
			at = v[unescapePointer(part)]
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(v) {
				return nil
			}
			at = v[i]
		default:
			return nil
		}
	}
	return newNode(at)
}

// unescapePointer reads a segment of a JSON pointer: its escapes stand for the
// separator and the escape themselves.
func unescapePointer(part string) string {
	part = strings.ReplaceAll(part, "~1", "/")
	return strings.ReplaceAll(part, "~0", "~")
}

// deref follows the references of a node to the mapping they name: a
// parameter or a request body of a document is often only a reference.
func (d *doc) deref(n node) node {
	for i := 0; n != nil && i < maxRefDepth; i++ {
		ref := n.text("$ref")
		if ref == "" {
			return n
		}
		next := d.lookup(ref)
		if next == nil {
			return nil
		}
		n = next
	}
	return nil
}

// maxRefDepth caps how many references one node follows, so a document that
// points its references in a circle reads as nothing rather than forever.
const maxRefDepth = 8

// nodesOf reads a list of mappings whose items may be references.
func (d *doc) nodesOf(n node, key string) []node {
	var out []node
	for _, item := range n.nodes(key) {
		if resolved := d.deref(item); resolved != nil {
			out = append(out, resolved)
		}
	}
	return out
}

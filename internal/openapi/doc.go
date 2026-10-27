// Package openapi reads the OpenAPI and Swagger documents the app imports:
// every operation of their paths becomes an interface of the collection, and
// every tag groups the interfaces it names into a folder.
package openapi

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document is what a parsed document says a collection holds.
type Document struct {
	// Title names the collection the document becomes.
	Title string
	// Spec is the version of the document itself (3.0.3, 2.0), APIVersion the
	// version of the API it describes.
	Spec       string
	APIVersion string
	// BaseURL is the server the interfaces were written against, "" when the
	// document names none.
	BaseURL string
	// Requests are the operations outside any tag: the 根目录 of the collection.
	Requests []Request
	// Folders group the tagged operations, one folder per tag.
	Folders []Folder
}

// Folder is one tag of the document and the interfaces it groups.
type Folder struct {
	Name     string
	Requests []Request
}

// Request is one operation of the document: the interface it becomes.
type Request struct {
	Name     string
	Method   string
	URL      string
	Params   []Row
	Path     []Row
	Headers  []Row
	BodyType string
	Body     string
	Form     []Row
}

// Row is one parameter, header or form field: its name, the value the document
// suggests for it, and what it is for.
type Row struct {
	Key, Value  string
	Description string
}

// RequestCount counts the interfaces of the document, folders and all.
func (d *Document) RequestCount() int {
	total := len(d.Requests)
	for _, folder := range d.Folders {
		total += len(folder.Requests)
	}
	return total
}

// FolderCount counts the folders the document groups its interfaces in.
func (d *Document) FolderCount() int { return len(d.Folders) }

// defaultTitle names the collection of a document that names none.
const defaultTitle = "导入的接口"

// Parse reads a document as OpenAPI 3 or Swagger 2, in JSON or in YAML: YAML
// reads JSON too, so both come in through the same decoder.
func Parse(data []byte) (*Document, error) {
	root, err := decode(data)
	if err != nil {
		return nil, err
	}
	d := &doc{root: root, refs: map[string]bool{}}
	switch {
	case strings.HasPrefix(root.text("openapi"), "3."):
		d.v3, d.spec = true, root.text("openapi")
	case strings.HasPrefix(root.text("swagger"), "2."):
		d.v3, d.spec = false, root.text("swagger")
	default:
		return nil, errors.New("不是 OpenAPI 或 Swagger 文档：缺少 openapi 或 swagger 版本字段")
	}
	info := root.obj("info")
	out := &Document{
		Title:      info.text("title"),
		Spec:       d.spec,
		APIVersion: info.text("version"),
		BaseURL:    d.absoluteServer(),
	}
	if out.Title == "" {
		out.Title = defaultTitle
	}
	out.Requests, out.Folders = d.operations()
	return out, nil
}

// doc is a document being read: its root, the flavour of spec it is, and the
// schemas whose expansion is under way.
type doc struct {
	root node
	v3   bool
	spec string
	// refs are the references being expanded: a schema that reaches itself
	// again stops there rather than reading forever.
	refs map[string]bool
}

// decode reads a document into its root mapping, dropping a byte order mark
// and refusing what is not a mapping at all.
func decode(data []byte) (node, error) {
	text := bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(text)) == 0 {
		return nil, errors.New("文档是空的")
	}
	var root map[string]any
	if err := yaml.Unmarshal(text, &root); err != nil {
		return nil, fmt.Errorf("文档既不是合法的 JSON 也不是合法的 YAML：%w", err)
	}
	n := node(root)
	if len(n) == 0 {
		return nil, errors.New("文档的根不是一个对象")
	}
	return n, nil
}

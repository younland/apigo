package openapi

import (
	"strconv"
	"strings"
)

// httpMethods are the operations a path holds, in the order the collection
// lists them: what a path offers under these keys is an interface, everything
// else (parameters, summary) belongs to the path itself.
var httpMethods = []string{"get", "post", "put", "patch", "delete", "head", "options", "trace"}

// operations turns every operation of the document into an interface, in the
// folder of its first tag: what is outside any tag stays in the 根目录.
func (d *doc) operations() ([]Request, []Folder) {
	var root []Request
	folders := &folderList{}
	paths := d.root.obj("paths")
	for _, path := range sortedKeys(paths) {
		at := paths.obj(path)
		for _, method := range httpMethods {
			op := at.obj(method)
			if op == nil {
				continue
			}
			req := d.request(path, method, op, at)
			if tag := firstTag(op); tag != "" {
				folders.add(tag, req)
				continue
			}
			req.Name = uniqueName(requestNames(root), req.Name)
			root = append(root, req)
		}
	}
	return root, folders.out()
}

// request is the interface one operation stands for: its name, method and URL,
// the parameters and headers it takes, and the body it sends.
func (d *doc) request(path, method string, op, at node) Request {
	params, headers := d.parameters(op, at)
	bodyType, body, form := d.body(op, at)
	return Request{
		Name:     operationName(op, method, path),
		Method:   strings.ToUpper(method),
		URL:      d.requestURL(path),
		Params:   params,
		Headers:  headers,
		BodyType: bodyType,
		Body:     body,
		Form:     form,
	}
}

// operationName names an interface: the summary its author wrote, else the id
// of the operation, else the method and the path it stands for.
func operationName(op node, method, path string) string {
	for _, name := range []string{op.text("summary"), op.text("operationId")} {
		if name != "" {
			return oneLine(name)
		}
	}
	return strings.ToUpper(method) + " " + path
}

// oneLine folds the line breaks of a summary into spaces: a name is a file
// name, and a file name has one line.
func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// firstTag is the tag an operation is grouped under: the first of the list,
// as a document lists its tags in the order they matter.
func firstTag(op node) string {
	for _, tag := range op.list("tags") {
		if name, ok := tag.(string); ok && strings.TrimSpace(name) != "" {
			return oneLine(name)
		}
	}
	return ""
}

// folderList keeps the folders of a document in the order their tags are first
// used, so a collection reads the way the document does.
type folderList struct {
	names  []string
	byName map[string]*Folder
}

func (f *folderList) add(name string, req Request) {
	if f.byName == nil {
		f.byName = map[string]*Folder{}
	}
	folder := f.byName[name]
	if folder == nil {
		f.names = append(f.names, name)
		f.byName[name] = &Folder{Name: name}
		folder = f.byName[name]
	}
	req.Name = uniqueName(requestNames(folder.Requests), req.Name)
	folder.Requests = append(folder.Requests, req)
}

// out lists the folders that hold something, in the order they were first
// used.
func (f *folderList) out() []Folder {
	var out []Folder
	for _, name := range f.names {
		if folder := f.byName[name]; folder != nil && len(folder.Requests) > 0 {
			out = append(out, *folder)
		}
	}
	return out
}

// requestNames lists the names a group of interfaces already holds.
func requestNames(reqs []Request) []string {
	names := make([]string, 0, len(reqs))
	for _, req := range reqs {
		names = append(names, req.Name)
	}
	return names
}

// uniqueName numbers a name another interface already uses, so one document
// makes no two interfaces that write to the same file.
func uniqueName(taken []string, name string) string {
	if name == "" {
		name = "未命名接口"
	}
	used := make(map[string]bool, len(taken))
	for _, have := range taken {
		used[have] = true
	}
	unique := name
	for n := 2; used[unique]; n++ {
		unique = name + " " + strconv.Itoa(n)
	}
	return unique
}

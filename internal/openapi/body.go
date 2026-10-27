package openapi

import "strings"

// parameters reads what an operation takes: its query parameters as rows of the
// Query pane, the values its path takes as rows of the Path pane, and its header
// parameters as rows of the Headers pane. A parameter of the path itself applies
// to every operation under it, and the operation's own definition of the same
// name wins.
func (d *doc) parameters(op, at node) (params, pathParams, headers []Row) {
	seen := map[string]bool{}
	for _, owner := range []node{op, at} {
		for _, param := range d.nodesOf(owner, "parameters") {
			key := param.text("in") + " " + param.text("name")
			if param.text("name") == "" || seen[key] {
				continue
			}
			seen[key] = true
			row := Row{
				Key:         param.text("name"),
				Value:       d.paramValue(param),
				Description: oneLine(param.text("description")),
			}
			switch param.text("in") {
			case "query":
				params = append(params, row)
			case "path":
				pathParams = append(pathParams, row)
			case "header":
				headers = append(headers, row)
			}
		}
	}
	return params, pathParams, headers
}

// paramValue is the value a parameter is written with: the example the document
// gives, else its default, else its schema's example, else nothing.
func (d *doc) paramValue(param node) string {
	for _, source := range []node{param, d.deref(param.obj("schema"))} {
		if source == nil {
			continue
		}
		for _, value := range []any{source["example"], source["default"]} {
			if text := valueText(value); text != "" {
				return text
			}
		}
	}
	return ""
}

// body reads what an operation sends: the JSON of its body schema as the text
// of the body pane, the fields of a form as the rows of its form, and nothing
// for an operation that sends no body.
func (d *doc) body(op, at node) (string, string, []Row) {
	if d.v3 {
		return d.bodyOfV3(op)
	}
	return d.bodyOfV2(op, at)
}

// bodyOfV3 reads the request body of an OpenAPI operation: the JSON its media
// type describes, the rows of a form, or the example of the type it offers.
func (d *doc) bodyOfV3(op node) (string, string, []Row) {
	requestBody := d.deref(op.obj("requestBody"))
	if requestBody == nil {
		return "", "", nil
	}
	mediaType, media := mediaOf(requestBody.obj("content"))
	if media == nil {
		return "", "", nil
	}
	switch {
	case isJSONMedia(mediaType):
		return "json", d.jsonExample(media, 0), nil
	case strings.HasPrefix(mediaType, "application/x-www-form-urlencoded"):
		return "urlencoded", "", d.formRows(media)
	case strings.HasPrefix(mediaType, "multipart/form-data"):
		return "form-data", "", d.formRows(media)
	case isBinaryMedia(mediaType):
		return "binary", "", nil
	}
	return "text", d.textExample(media), nil
}

// isBinaryMedia reports whether a media type carries a file rather than the text
// a body pane edits.
func isBinaryMedia(mediaType string) bool {
	base, _, _ := strings.Cut(mediaType, ";")
	base = strings.TrimSpace(base)
	if base == "application/octet-stream" {
		return true
	}
	for _, prefix := range []string{"image/", "audio/", "video/"} {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}

// bodyOfV2 reads the body of a Swagger operation: its body parameter, or the
// form fields it declares.
func (d *doc) bodyOfV2(op, at node) (string, string, []Row) {
	consumes := d.mediaTypes(op, at)
	var form []Row
	for _, param := range d.nodesOf(op, "parameters") {
		switch param.text("in") {
		case "body":
			if isJSONMedia(consumes) {
				return "json", d.jsonExample(param, 0), nil
			}
			return "text", d.textExample(param), nil
		case "formData":
			form = append(form, Row{Key: param.text("name"), Value: d.paramValue(param)})
		}
	}
	if len(form) == 0 {
		return "", "", nil
	}
	if strings.HasPrefix(consumes, "multipart/form-data") {
		return "form-data", "", form
	}
	return "urlencoded", "", form
}

// mediaTypes is the media type a Swagger operation sends with: the first of its
// own consumes, else the first of the document's.
func (d *doc) mediaTypes(op, at node) string {
	for _, owner := range []node{op, at, d.root} {
		for _, item := range owner.list("consumes") {
			if mediaType, ok := item.(string); ok && mediaType != "" {
				return mediaType
			}
		}
	}
	return "application/json"
}

// formRows reads the fields of a form body from the properties of its schema,
// each written with the value the document suggests.
func (d *doc) formRows(media node) []Row {
	schema := d.deref(media.obj("schema"))
	if schema == nil {
		return nil
	}
	var rows []Row
	for _, name := range sortedKeys(schema.obj("properties")) {
		field := d.deref(schema.obj("properties").obj(name))
		rows = append(rows, Row{Key: name, Value: d.paramValue(field)})
	}
	return rows
}

// jsonExample is the body an example stands for: the document's own example
// when it gives one, else the JSON its schema describes.
func (d *doc) jsonExample(holder node, depth int) string {
	if example, ok := holder["example"]; ok {
		if text := marshalIndented(example); text != "" {
			return text
		}
	}
	schema := d.deref(holder.obj("schema"))
	if schema == nil {
		schema = holder
	}
	return marshalIndented(d.example(schema, depth))
}

// textExample is a body that is not JSON: the example as it was written, or
// the text its type suggests.
func (d *doc) textExample(holder node) string {
	if example, ok := holder["example"]; ok {
		if text := valueText(example); text != "" {
			return text
		}
	}
	if value := d.example(d.deref(holder.obj("schema")), 0); value != nil {
		return valueText(value)
	}
	return ""
}

// valueText writes a value as the text of a row: a string as it is, anything
// else as the JSON it reads as.
func valueText(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case bool:
		if value {
			return "true"
		}
		return "false"
	}
	return marshalIndented(v)
}

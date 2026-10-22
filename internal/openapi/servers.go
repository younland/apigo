package openapi

import "strings"

// baseURL is the server the interfaces are written against: the first of the
// servers of an OpenAPI document, or the scheme, host and base path a Swagger
// document names apart. A document that names none reads as "", and one that
// names a bare path reads as that path.
func (d *doc) baseURL() string {
	if d.v3 {
		servers := d.root.nodes("servers")
		if len(servers) == 0 {
			return ""
		}
		return trimSlash(servers[0].text("url"))
	}
	host := d.root.text("host")
	if host == "" {
		return ""
	}
	scheme := "https"
	if schemes := d.root.list("schemes"); len(schemes) > 0 {
		if first, ok := schemes[0].(string); ok && first != "" {
			scheme = first
		}
	}
	return trimSlash(scheme + "://" + host + d.root.text("basePath"))
}

// absoluteServer is the server a collection takes as its baseurl: the one the
// document names with its scheme and host, "" for a relative path, which reads
// against the baseurl the reader fills in.
func (d *doc) absoluteServer() string {
	if server := d.baseURL(); strings.Contains(server, "://") {
		return server
	}
	return ""
}

// trimSlash drops the trailing separators of a server URL, so a path follows
// it with one separator rather than two.
func trimSlash(url string) string { return strings.TrimRight(strings.TrimSpace(url), "/") }

// requestURL puts a path of the document under the server it goes out with. A
// document whose server is a bare path, or which names none, keeps an address
// that starts with a slash: the app reads it against the baseurl of the
// environment in use, as it reads any address the reader writes that way.
func (d *doc) requestURL(path string) string {
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	server := d.baseURL()
	if strings.Contains(server, "://") {
		return server + path
	}
	if server != "" && server[0] != '/' {
		server = "/" + server
	}
	return server + path
}

// mediaOf picks the media type of a content mapping: JSON when it is offered
// under any name, else a form, else the first of what it holds.
func mediaOf(content node) (mediaType string, media node) {
	if content == nil {
		return "", nil
	}
	keys := sortedKeys(content)
	for _, key := range keys {
		if isJSONMedia(key) {
			return key, content.obj(key)
		}
	}
	for _, key := range keys {
		switch {
		case strings.HasPrefix(key, "application/x-www-form-urlencoded"):
			return key, content.obj(key)
		case strings.HasPrefix(key, "multipart/form-data"):
			return key, content.obj(key)
		}
	}
	if len(keys) == 0 {
		return "", nil
	}
	return keys[0], content.obj(keys[0])
}

// isJSONMedia reports whether a media type carries JSON, as +json types do.
func isJSONMedia(mediaType string) bool {
	base, _, _ := strings.Cut(mediaType, ";")
	base = strings.TrimSpace(base)
	return base == "application/json" || strings.HasSuffix(base, "+json")
}

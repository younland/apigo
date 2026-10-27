package app

import (
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// responseBody is how the pane shows a response body: as code, as a picture, as
// a player, or as a file to keep.
type responseBody int

const (
	bodyText  responseBody = iota // JSON, XML, HTML, plain text: the editor
	bodyImage                     // a picture the pane draws
	bodyAudio                     // a sound the pane plays
	bodyVideo                     // a video the app cannot play: a file to keep
	bodyFile                      // anything else: a file to keep
)

// responseBodyOf says how a response is shown, by what the server calls it and,
// when it says nothing the pane knows, by what its bytes are.
func responseBodyOf(res *response) responseBody {
	kind := mediaType(responseContentType(res))
	switch {
	case isJSON(responseContentType(res), res.Body):
		return bodyText
	case kind == "":
		return bodyText // nothing to go on: read it as text
	case strings.HasPrefix(kind, "image/"):
		return bodyImage
	case strings.HasPrefix(kind, "audio/"):
		return bodyAudio
	case strings.HasPrefix(kind, "video/"):
		return bodyVideo
	case strings.HasPrefix(kind, "text/"), kind == "application/xml",
		strings.HasSuffix(kind, "+xml"), kind == "application/javascript",
		kind == "application/x-www-form-urlencoded":
		return bodyText
	default:
		return bodyFile
	}
}

// responseContentType is what the response says it holds, or what its bytes
// look like when it says nothing.
func responseContentType(res *response) string {
	if said := res.Headers.Get("Content-Type"); said != "" {
		return said
	}
	if len(res.Raw) == 0 {
		return ""
	}
	return http.DetectContentType(res.Raw)
}

// mediaType is a Content-Type without its parameters, in lower case.
func mediaType(contentType string) string {
	if parsed, _, err := mime.ParseMediaType(contentType); err == nil {
		return parsed
	}
	if at := strings.IndexByte(contentType, ';'); at >= 0 {
		contentType = contentType[:at]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

// downloadName is the file a response saves as: the name the server suggests,
// else the last part of the address, else a name from the media type. The
// address is the one the request went out to, its path filled in, so a file is
// never named after a {placeholder}.
func downloadName(res *response, address string) string {
	if said := suggestedName(res.Headers.Get("Content-Disposition")); said != "" {
		return safeDownloadName(said)
	}
	if parsed, err := url.Parse(address); err == nil {
		if base := path.Base(parsed.Path); base != "" && base != "." && base != "/" {
			return safeDownloadName(base)
		}
	}
	return "response" + extensionOf(mediaType(responseContentType(res)))
}

// suggestedName reads the name a Content-Disposition header gives: filename*
// first, which carries its encoding, then filename.
func suggestedName(disposition string) string {
	if disposition == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		return ""
	}
	return params["filename"]
}

// extensionOf is the usual extension of a media type, ".bin" for one it does
// not know.
func extensionOf(kind string) string {
	if kind == "" {
		return ".bin"
	}
	if extensions, err := mime.ExtensionsByType(kind); err == nil && len(extensions) > 0 {
		return extensions[0]
	}
	if at := strings.IndexByte(kind, '/'); at >= 0 {
		return "." + kind[at+1:]
	}
	return ".bin"
}

// safeDownloadName keeps a suggested name a name: no path of its own, and no
// character Windows refuses.
func safeDownloadName(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	name = path.Base(name)
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < ' ' {
			return '_'
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "response"
	}
	return name
}

// sentAddress is the address the last request went out to, its variables, its
// baseurl and its path filled in. A tab that has not sent anything reads its own
// address: the response pane names a file after whichever it has.
func (t *tab) sentAddress() string {
	if t.sentURL != "" {
		return t.sentURL
	}
	return t.url
}

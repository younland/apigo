package req

// Format is one of the formats the Body tab offers: the key an interface file
// keeps, the name the strip shows, and the Content-Type a request sends with
// it.
type Format struct {
	Key   string
	Label string
	MIME  string
	Edits Edits
}

// Edits is how the pane edits a format.
type Edits int

const (
	EditsNone Edits = iota // nothing to send
	EditsRows              // key/value rows, sent encoded
	EditsText              // the code editor's text
	EditsFile              // a file the app does not pick yet
)

// Formats are the formats of the Body tab, in the order it shows them.
var Formats = []Format{
	{"none", "none", "", EditsNone},
	{"form-data", "form-data", "multipart/form-data", EditsRows},
	{"urlencoded", "x-www-form-urlencoded", "application/x-www-form-urlencoded", EditsRows},
	{"json", "JSON", "application/json", EditsText},
	{"xml", "XML", "application/xml", EditsText},
	{"text", "Text", "text/plain", EditsText},
	{"binary", "Binary", "application/octet-stream", EditsFile},
	{"graphql", "GraphQL", "application/json", EditsText},
	{"msgpack", "msgpack", "application/msgpack", EditsFile},
}

// FormatOf returns the format a key names, the default for one it does not
// know.
func FormatOf(key string) Format {
	for _, format := range Formats {
		if format.Key == key {
			return format
		}
	}
	return Formats[3]
}

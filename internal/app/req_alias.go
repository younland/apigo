package app

import "apigo/internal/req"

// The request model lives in internal/req: the formats the Body tab offers, the
// spec a send builds, and the response it leaves. These aliases keep the panes
// and the tests reading by the names the app has always used.
type (
	response   = req.Response
	formPart   = req.Part
	bodyFormat = req.Format
	bodyEdits  = req.Edits
)

// The ways the pane edits a body are the request package's, so what a format
// says and what the pane shows are the same.
const (
	editsNone = req.EditsNone
	editsRows = req.EditsRows
	editsText = req.EditsText
	editsFile = req.EditsFile
)

// bodyFormats are the formats of the Body tab, in the order it shows them.
var bodyFormats = req.Formats

// bodyFormatOf returns the format a key names, the default for one it does not
// know.
func bodyFormatOf(key string) bodyFormat { return req.FormatOf(key) }

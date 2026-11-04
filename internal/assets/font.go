package assets

import (
	_ "embed"
	"log"

	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// interfaceFont is the font the app draws with, embedded rather than
// installed: every platform shows the same Latin shapes, with glyphs the font
// does not cover falling back to the system's fonts.
//
//go:embed InterVariable.ttf
var interfaceFont []byte

// fontReady reports whether the embedded font could be added to the text
// system, so tests can tell a system without a text engine from a broken
// embed.
var fontReady bool

func init() {
	err := ui.RegisterFont(interfaceFont, design.FontFamily)
	fontReady = err == nil
	if err != nil {
		// The theme still names the family: text falls back to the system's.
		log.Printf("读取界面字体失败，改用系统字体：%v", err)
	}
}

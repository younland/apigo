package design

import "github.com/egoist/mygo/ui"

// Accent is one accent choice of the interface, as the settings page offers
// them: the color of buttons, links, selections and the focus ring.
type Accent struct {
	Name string
	Base ui.Color
}

// Accents are the accent choices, green first as the app's own.
var Accents = []Accent{
	{"Green", ui.Hex("#10b981")},
	{"Teal", ui.Hex("#14b8a6")},
	{"Blue", ui.Hex("#3b82f6")},
	{"Indigo", ui.Hex("#6366f1")},
	{"Purple", ui.Hex("#a855f7")},
	{"Amber", ui.Hex("#f59e0b")},
	{"Orange", ui.Hex("#f97316")},
	{"Red", ui.Hex("#ef4444")},
	{"Pink", ui.Hex("#ec4899")},
}

// DefaultAccent is the accent a window draws with until one is picked.
var DefaultAccent = Accents[0]

// AccentByName returns the accent of a name, the default for an unknown one.
func AccentByName(name string) Accent {
	for _, accent := range Accents {
		if accent.Name == name {
			return accent
		}
	}
	return DefaultAccent
}

// PaletteFor returns the palette of an appearance drawn with an accent.
func PaletteFor(m Mode, accent Accent) Palette { return PaletteOf(m).WithAccent(accent) }

// WithAccent returns the palette drawn with another accent: its hover shade
// mixes toward white, its pressed shade toward black, and its text is black
// or white by the color's own lightness.
func (p Palette) WithAccent(accent Accent) Palette {
	p.Accent = accent.Base
	p.AccentHover = accent.Base.Mix(ui.Hex("#ffffff"), 0.18)
	p.AccentPressed = accent.Base.Mix(ui.Hex("#000000"), 0.18)
	p.AccentText = ui.Hex("#ffffff")
	if luminance(accent.Base) > 150 {
		p.AccentText = ui.Hex("#111827")
	}
	return p
}

// luminance is the color's perceived brightness, from 0 (black) to 255
// (white), as a threshold for the text that reads on it.
func luminance(c ui.Color) float32 {
	return 0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B)
}

// Package design holds the colors, metrics and type sizes sampled from
// Hoppscotch's themes, so apigo's screens can be matched against it.
package design

import "github.com/egoist/mygo/ui"

// Sizes of the app's regions and controls, in DIPs.
const (
	HeaderHeight     float32 = 48
	TabStripHeight   float32 = 40
	SidenavWidth     float32 = 56
	WorkspacePadding float32 = 12
	FooterHeight     float32 = 32
	SidebarWidth     float32 = 240
	MethodWidth      float32 = 104
	RequestRowHeight float32 = 40
	PanelPadding     float32 = 16
	RowGap           float32 = 8
	Radius           float32 = 4
	BorderWidth      float32 = 1
)

// Type sizes, in DIPs. Hoppscotch sets body text at 12px on a 16px line.
const (
	FontSizeBody    float32 = 12
	FontSizeTiny    float32 = 10
	FontSizeTitle   float32 = 14
	FontSizeDisplay float32 = 18
	LineHeight      float32 = 16
)

// Palette is one appearance of the app.
type Palette struct {
	Dark bool

	Background    ui.Color
	Surface       ui.Color
	SurfaceHover  ui.Color
	Divider       ui.Color
	DividerStrong ui.Color

	Text          ui.Color
	TextSecondary ui.Color
	TextMuted     ui.Color

	Accent        ui.Color
	AccentHover   ui.Color
	AccentPressed ui.Color
	AccentText    ui.Color

	Danger ui.Color

	Get     ui.Color
	Post    ui.Color
	Put     ui.Color
	Patch   ui.Color
	Delete  ui.Color
	Head    ui.Color
	Options ui.Color
	Other   ui.Color

	StatusInfo        ui.Color
	StatusSuccess     ui.Color
	StatusRedirect    ui.Color
	StatusClientError ui.Color
	StatusServerError ui.Color
	StatusMissing     ui.Color
}

var (
	light = Palette{
		Background:    ui.Hex("#ffffff"),
		Surface:       ui.Hex("#fafafa"),
		SurfaceHover:  ui.Hex("#f3f4f6"),
		Divider:       ui.Hex("#f3f4f6"),
		DividerStrong: ui.Hex("#d1d5db"),

		Text:          ui.Hex("#111827"),
		TextSecondary: ui.Hex("#6b7280"),
		TextMuted:     ui.Hex("#9ca3af"),

		Accent:        ui.Hex("#10b981"),
		AccentHover:   ui.Hex("#34d399"),
		AccentPressed: ui.Hex("#059669"),
		AccentText:    ui.Hex("#ffffff"),

		Danger: ui.Hex("#ef4444"),

		Get:     ui.Hex("#22c55e"),
		Post:    ui.Hex("#f59e0b"),
		Put:     ui.Hex("#3b82f6"),
		Patch:   ui.Hex("#a855f7"),
		Delete:  ui.Hex("#ef4444"),
		Head:    ui.Hex("#84cc16"),
		Options: ui.Hex("#ec4899"),
		Other:   ui.Hex("#6b7280"),

		StatusInfo:        ui.Hex("#3b82f6"),
		StatusSuccess:     ui.Hex("#22c55e"),
		StatusRedirect:    ui.Hex("#f59e0b"),
		StatusClientError: ui.Hex("#ef4444"),
		StatusServerError: ui.Hex("#f43f5e"),
		StatusMissing:     ui.Hex("#64748b"),
	}

	dark = Palette{
		Dark:          true,
		Background:    ui.Hex("#181818"),
		Surface:       ui.Hex("#1c1c1e"),
		SurfaceHover:  ui.Hex("#262626"),
		Divider:       ui.Hex("#1f1f1f"),
		DividerStrong: ui.Hex("#27272a"),

		Text:          ui.Hex("#fafafa"),
		TextSecondary: ui.Hex("#a3a3a3"),
		TextMuted:     ui.Hex("#737373"),

		Accent:        ui.Hex("#10b981"),
		AccentHover:   ui.Hex("#34d399"),
		AccentPressed: ui.Hex("#059669"),
		AccentText:    ui.Hex("#ffffff"),

		Danger: ui.Hex("#f43f5e"),

		Get:     ui.Hex("#10b981"),
		Post:    ui.Hex("#eab308"),
		Put:     ui.Hex("#0ea5e9"),
		Patch:   ui.Hex("#8b5cf6"),
		Delete:  ui.Hex("#f43f5e"),
		Head:    ui.Hex("#14b8a6"),
		Options: ui.Hex("#6366f1"),
		Other:   ui.Hex("#737373"),

		StatusInfo:        ui.Hex("#3b82f6"),
		StatusSuccess:     ui.Hex("#22c55e"),
		StatusRedirect:    ui.Hex("#f59e0b"),
		StatusClientError: ui.Hex("#ef4444"),
		StatusServerError: ui.Hex("#f43f5e"),
		StatusMissing:     ui.Hex("#64748b"),
	}

	black = func() Palette {
		p := dark
		p.Background = ui.Hex("#0f0f0f")
		p.Surface = ui.Hex("#171717")
		p.Divider = ui.Hex("#171717")
		p.Text = ui.Hex("#fafafa")
		p.Other = ui.Hex("#71717a")
		return p
	}()
)

// PaletteOf returns the palette of an appearance; System draws light until
// the view resolves it against the desktop.
func PaletteOf(m Mode) Palette {
	switch m {
	case Dark:
		return dark
	case Black:
		return black
	default:
		return light
	}
}

// Method colors a request method as Hoppscotch's badges do.
func (p Palette) Method(method string) ui.Color {
	switch method {
	case "GET":
		return p.Get
	case "POST":
		return p.Post
	case "PUT":
		return p.Put
	case "PATCH":
		return p.Patch
	case "DELETE":
		return p.Delete
	case "HEAD":
		return p.Head
	case "OPTIONS":
		return p.Options
	default:
		return p.Other
	}
}

// Status colors an HTTP status code by its class.
func (p Palette) Status(code int) ui.Color {
	switch {
	case code == 0:
		return p.StatusMissing
	case code < 200:
		return p.StatusInfo
	case code < 300:
		return p.StatusSuccess
	case code < 400:
		return p.StatusRedirect
	case code < 500:
		return p.StatusClientError
	default:
		return p.StatusServerError
	}
}

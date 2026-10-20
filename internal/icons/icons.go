// Package icons holds the SVGs the app draws. Each one is a 24×24 outline
// icon in currentColor, so `ui.Icon` sizes it by the font size and colors
// it by the text color.
package icons

import "github.com/egoist/mygo/ui"

func parse(svg string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + svg + `</svg>`))
}

var (
	Send     = parse(`<path d="M22 2 11 13"/><path d="M22 2 15 22l-4-9-9-4Z"/>`)
	Search   = parse(`<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>`)
	Chevron  = parse(`<path d="m6 9 6 6 6-6"/>`)
	Plus     = parse(`<path d="M12 5v14M5 12h14"/>`)
	Close    = parse(`<path d="M18 6 6 18M6 6l12 12"/>`)
	Folder   = parse(`<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>`)
	Clock    = parse(`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`)
	Sliders  = parse(`<path d="M4 6h16M4 12h16M4 18h16"/><circle cx="9" cy="6" r="2"/><circle cx="15" cy="12" r="2"/><circle cx="8" cy="18" r="2"/>`)
	Gear     = parse(`<circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M4.9 19.1 7 17M17 7l2.1-2.1"/>`)
	Braces   = parse(`<path d="M8 4H6a2 2 0 0 0-2 2v4l-2 2 2 2v4a2 2 0 0 0 2 2h2"/><path d="M16 4h2a2 2 0 0 1 2 2v4l2 2-2 2v4a2 2 0 0 1-2 2h-2"/>`)
	Triangle = parse(`<circle cx="12" cy="4" r="2"/><circle cx="4" cy="19" r="2"/><circle cx="20" cy="19" r="2"/><path d="M12 6 5 17M12 6l7 11M6 19h12"/>`)
	Radio    = parse(`<circle cx="12" cy="12" r="2"/><path d="M7.8 7.8a6 6 0 0 0 0 8.4M16.2 16.2a6 6 0 0 0 0-8.4M4.9 4.9a10 10 0 0 0 0 14.2M19.1 19.1a10 10 0 0 0 0-14.2"/>`)
	Copy     = parse(`<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15V5a2 2 0 0 1 2-2h10"/>`)
	Trash    = parse(`<path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2M6 7l1 13h10l1-13"/>`)
	Grip     = parse(`<circle cx="9" cy="6" r="1"/><circle cx="9" cy="12" r="1"/><circle cx="9" cy="18" r="1"/><circle cx="15" cy="6" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="15" cy="18" r="1"/>`)
	Alert    = parse(`<circle cx="12" cy="12" r="9"/><path d="M12 8v5M12 16h.01"/>`)
	Check    = parse(`<path d="m4 12 5 5L20 6"/>`)
	Eye      = parse(`<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>`)
	Home     = parse(`<path d="m3 10 9-7 9 7v9a2 2 0 0 1-2 2h-4v-6H9v6H5a2 2 0 0 1-2-2Z"/>`)
	More     = parse(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`)
	Plug     = parse(`<path d="M9 2v6M15 2v6M7 8h10v3a5 5 0 0 1-10 0ZM12 16v6"/>`)
	Bolt     = parse(`<path d="M13 2 4 14h6l-1 8 9-12h-6Z"/>`)
	Stack    = parse(`<path d="m12 3 9 5-9 5-9-5ZM3 13l9 5 9-5"/>`)
)

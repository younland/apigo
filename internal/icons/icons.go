// Package icons holds the SVGs the app draws. Each one is a 24×24 outline
// icon in currentColor, so `ui.Icon` sizes it by the font size and colors
// it by the text color.
package icons

import "github.com/egoist/mygo/ui"

func parse(svg string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + svg + `</svg>`))
}

var (
	Send      = parse(`<path d="M22 2 11 13"/><path d="M22 2 15 22l-4-9-9-4Z"/>`)
	Search    = parse(`<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>`)
	Chevron   = parse(`<path d="m6 9 6 6 6-6"/>`)
	Plus      = parse(`<path d="M12 5v14M5 12h14"/>`)
	Close     = parse(`<path d="M18 6 6 18M6 6l12 12"/>`)
	Folder    = parse(`<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>`)
	Sliders   = parse(`<path d="M4 6h16M4 12h16M4 18h16"/><circle cx="9" cy="6" r="2"/><circle cx="15" cy="12" r="2"/><circle cx="8" cy="18" r="2"/>`)
	Sun       = parse(`<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>`)
	Gear      = parse(`<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z"/>`)
	Monitor   = parse(`<rect x="2" y="4" width="20" height="14" rx="2"/><path d="M8 21h8M12 18v3"/>`)
	Moon      = parse(`<path d="M20 14a8 8 0 0 1-10-10 8 8 0 1 0 10 10Z"/>`)
	MoonLit   = parse(`<path d="M20 14a8 8 0 0 1-10-10 8 8 0 1 0 10 10Z" fill="currentColor"/>`)
	Translate = parse(`<path d="M4 5h10M9 3v2M12 5c0 4-3 8-7 10M6 9c1 3 3 5 6 6"/><path d="M13 21l4-10 4 10M15 17h5"/>`)
	Braces    = parse(`<path d="M8 4H6a2 2 0 0 0-2 2v4l-2 2 2 2v4a2 2 0 0 0 2 2h2"/><path d="M16 4h2a2 2 0 0 1 2 2v4l2 2-2 2v4a2 2 0 0 1-2 2h-2"/>`)
	Triangle  = parse(`<circle cx="12" cy="4" r="2"/><circle cx="4" cy="19" r="2"/><circle cx="20" cy="19" r="2"/><path d="M12 6 5 17M12 6l7 11M6 19h12"/>`)
	Radio     = parse(`<circle cx="12" cy="12" r="2"/><path d="M7.8 7.8a6 6 0 0 0 0 8.4M16.2 16.2a6 6 0 0 0 0-8.4M4.9 4.9a10 10 0 0 0 0 14.2M19.1 19.1a10 10 0 0 0 0-14.2"/>`)
	Copy      = parse(`<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15V5a2 2 0 0 1 2-2h10"/>`)
	Trash     = parse(`<path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2M6 7l1 13h10l1-13"/>`)
	Grip      = parse(`<circle cx="9" cy="6" r="1"/><circle cx="9" cy="12" r="1"/><circle cx="9" cy="18" r="1"/><circle cx="15" cy="6" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="15" cy="18" r="1"/>`)
	Alert     = parse(`<circle cx="12" cy="12" r="9"/><path d="M12 8v5M12 16h.01"/>`)
	Check     = parse(`<path d="m4 12 5 5L20 6"/>`)
	Eye       = parse(`<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>`)
	Home      = parse(`<path d="m3 10 9-7 9 7v9a2 2 0 0 1-2 2h-4v-6H9v6H5a2 2 0 0 1-2-2Z"/>`)
	More      = parse(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`)
	// MoreVertical is the "…" of a list row, standing on its end as rows
	// stack: the tree's row menus read that way.
	MoreVertical = parse(`<circle cx="12" cy="5" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="12" cy="19" r="1"/>`)
	Plug         = parse(`<path d="M9 2v6M15 2v6M7 8h10v3a5 5 0 0 1-10 0ZM12 16v6"/>`)
	// SidebarClose and SidebarOpen are the toggle of a collection's tree: a page
	// with its tree on the left, and an arrow for the way the tree goes.
	SidebarClose = parse(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/><path d="m17 9-3 3 3 3"/>`)
	SidebarOpen  = parse(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/><path d="m14 9 3 3-3 3"/>`)
	Bolt         = parse(`<path d="M13 2 4 14h6l-1 8 9-12h-6Z"/>`)
	Stack        = parse(`<path d="m12 3 9 5-9 5-9-5ZM3 13l9 5 9-5"/>`)
	External     = parse(`<path d="M15 3h6v6"/><path d="M21 3l-9 9"/><path d="M19 14v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h6"/>`)
	Pencil       = parse(`<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/>`)
	Wand         = parse(`<path d="M15 4V2M15 16v-2M8 9h2M20 9h2M17.8 11.8 19 13M17.8 6.2 19 5M3 21l9-9M12.2 6.2 11 5"/>`)
	File         = parse(`<path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9Z"/><path d="M13 2v7h7"/>`)
	// The response pane's media icons: a body is shown as code, as a picture,
	// as a player, or as a file to keep.
	Play     = parse(`<path d="m7 4 13 8-13 8Z"/>`)
	Pause    = parse(`<path d="M9 4v16M15 4v16"/>`)
	Music    = parse(`<path d="M9 18V5l10-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="16" cy="16" r="3"/>`)
	Download = parse(`<path d="M12 3v12M7 11l5 5 5-5M4 20h16"/>`)
	Picture  = parse(`<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/>`)
	Video    = parse(`<rect x="2" y="6" width="14" height="12" rx="2"/><path d="m16 10 6-3v10l-6-3Z"/>`)
	Save     = parse(`<path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2Z"/><path d="M17 21v-8H7v8M7 3v5h8"/>`)
)

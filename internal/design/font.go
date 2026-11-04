package design

import "strings"

const FontFallback = "sans-serif"

// FontWithFallback names the selected interface font and the system's generic
// sans-serif families for glyphs it does not cover.
func FontWithFallback(family string) string {
	family = strings.TrimSpace(family)
	if family == "" {
		family = FontFamily
	}
	return family + ", " + FontFallback
}

package main

import (
	"bytes"
	"testing"

	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// TestInterfaceFontEmbedded checks the font the app draws with is in the
// binary and, where the system can add fonts, that text can use it under the
// family the theme names.
func TestInterfaceFontEmbedded(t *testing.T) {
	// A TrueType font starts with version 1.0.
	if !bytes.HasPrefix(interfaceFont, []byte{0x00, 0x01, 0x00, 0x00}) {
		t.Fatalf("embedded font starts with % x, want a TrueType header", interfaceFont[:4])
	}
	if !fontReady {
		t.Skip("this system has no text engine to add fonts to")
	}
	families := ui.FontFamilies()
	for _, family := range families {
		if family.Name != design.FontFamily {
			continue
		}
		if !family.Registered {
			t.Errorf("family %q is the system's, want the app's own", family.Name)
		}
		return
	}
	t.Errorf("families %+v, want %q among them", families, design.FontFamily)
}

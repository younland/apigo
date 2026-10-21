package design

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMethodColors(t *testing.T) {
	p := PaletteOf(Light)
	tests := []struct {
		method string
		want   ui.Color
	}{
		{"GET", p.Get},
		{"POST", p.Post},
		{"PUT", p.Put},
		{"PATCH", p.Patch},
		{"DELETE", p.Delete},
		{"HEAD", p.Head},
		{"OPTIONS", p.Options},
		{"TRACE", p.Other},
	}
	for _, test := range tests {
		if got := p.Method(test.method); got != test.want {
			t.Errorf("Method(%q) = %v, want %v", test.method, got, test.want)
		}
	}
}

func TestStatusClasses(t *testing.T) {
	p := PaletteOf(Dark)
	tests := []struct {
		code int
		want ui.Color
	}{
		{0, p.StatusMissing},
		{101, p.StatusInfo},
		{204, p.StatusSuccess},
		{304, p.StatusRedirect},
		{404, p.StatusClientError},
		{503, p.StatusServerError},
	}
	for _, test := range tests {
		if got := p.Status(test.code); got != test.want {
			t.Errorf("Status(%d) = %v, want %v", test.code, got, test.want)
		}
	}
}

func TestThemeFollowsPalette(t *testing.T) {
	for _, mode := range []Mode{Light, Dark, Black} {
		p := PaletteOf(mode)
		th := Theme(mode, DefaultAccent)
		if th.Dark != p.Dark {
			t.Errorf("%v theme Dark = %v, want %v", mode, th.Dark, p.Dark)
		}
		if th.Background != p.Background {
			t.Errorf("%v theme background %v, want %v", mode, th.Background, p.Background)
		}
		if th.FontSize != FontSizeBody || th.Radius != Radius {
			t.Errorf("%v theme font %v radius %v, want %v and %v",
				mode, th.FontSize, th.Radius, FontSizeBody, Radius)
		}
		if th.Font != FontFamily {
			t.Errorf("%v theme family %q, want the embedded %q", mode, th.Font, FontFamily)
		}
		if th.Accent != DefaultAccent.Base {
			t.Errorf("%v theme accent %v, want the default %v", mode, th.Accent, DefaultAccent.Base)
		}
	}
}

func TestAccentNames(t *testing.T) {
	for _, accent := range Accents {
		if got := AccentByName(accent.Name); got != accent {
			t.Errorf("AccentByName(%q) = %v, want %v", accent.Name, got, accent)
		}
	}
	if got := AccentByName("no such color"); got != DefaultAccent {
		t.Errorf("AccentByName of an unknown name = %v, want the default", got)
	}
	if Accents[0] != DefaultAccent {
		t.Errorf("Accents[0] = %v, want the default first", Accents[0])
	}
}

func TestWithAccentDerivesShades(t *testing.T) {
	for _, accent := range Accents {
		p := PaletteFor(Light, accent)
		if p.Accent != accent.Base {
			t.Errorf("%s accent %v, want %v", accent.Name, p.Accent, accent.Base)
		}
		if luminance(p.AccentHover) <= luminance(p.Accent) {
			t.Errorf("%s hover %v is not lighter than %v", accent.Name, p.AccentHover, p.Accent)
		}
		if luminance(p.AccentPressed) >= luminance(p.Accent) {
			t.Errorf("%s pressed %v is not darker than %v", accent.Name, p.AccentPressed, p.Accent)
		}
	}
}

func TestModeCycle(t *testing.T) {
	m := System
	for i, want := range []Mode{Light, Dark, Black, System} {
		if m = m.Next(); m != want {
			t.Fatalf("Next() #%d = %v, want %v", i, m, want)
		}
	}
}

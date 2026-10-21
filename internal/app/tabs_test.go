package app

import (
	"strings"
	"testing"

	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// TestRequestTabCapsItsWidth checks a long title, a URL usually, is shortened
// so the tab stays inside its cap and the tabs beside it stay in view.
func TestRequestTabCapsItsWidth(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	a.current().title = strings.Repeat("很长", 40)
	tt.Frame()

	tab, ok := tt.Find("标签页 " + a.current().title)
	if !ok {
		t.Fatal("no tab for the long title")
	}
	if tab.W > design.RequestTabMaxWidth {
		t.Errorf("tab width %v, want no more than the %v cap", tab.W, design.RequestTabMaxWidth)
	}
	shortened := false
	for _, text := range tt.Texts() {
		if strings.HasPrefix(text, "很长") && strings.HasSuffix(text, "…") {
			shortened = true
		}
	}
	if !shortened {
		t.Errorf("texts %q, want the long title shortened with an ellipsis", tt.Texts())
	}
}

// TestRequestTabStripHidesItsBar checks the strip of request tabs scrolls
// without drawing a bar over its tabs.
func TestRequestTabStripHidesItsBar(t *testing.T) {
	a := testApp(t)
	ws := a.space()
	for i := 0; i < 5; i++ {
		a.addTab(ws)
	}
	// Narrow enough that the tabs overflow the strip.
	tt := ui.NewTester(a.view, 420, 800)
	first, ok := tt.Find("标签页 Untitled")
	if !ok {
		t.Fatal("no request tab")
	}
	tt.Move(first.X+first.W/2, first.Y+first.H/2)
	tt.Frame()

	surface, page := a.palette().Surface, a.palette().Background
	img := tt.Image()
	// The band under the tab's text: a bar of the strip would be drawn there.
	for y := int(first.Y+first.H) - 5; y < int(first.Y+first.H)-1; y++ {
		for x := int(first.X) + 2; x < int(first.X+first.W)-2; x++ {
			if c := img.RGBAAt(x, y); !sameColor(c, surface) && !sameColor(c, page) {
				t.Fatalf("pixel (%d,%d) is %v, want a tab background: a scroll bar shows over the tabs",
					x, y, c)
			}
		}
	}
}

// TestEnvironmentPickerHasNoEye checks the strip ends at the environment
// picker: the eye that hid environment values is gone.
func TestEnvironmentPickerHasNoEye(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	if _, ok := tt.Find("隐藏环境值"); ok {
		t.Error("the strip still shows the eye that hid environment values")
	}
	if _, ok := tt.Find("环境：本地"); !ok {
		t.Error("the environment picker is missing from the strip")
	}
}

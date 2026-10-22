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

// TestEnvironmentPickerCentersInTheStrip checks the picker sits halfway down
// the strip rather than against its top edge, as the tabs themselves do not.
func TestEnvironmentPickerCentersInTheStrip(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	picker, ok := tt.Find("环境：本地")
	if !ok {
		t.Fatal("no environment picker")
	}
	// The "+" that opens a tab is as tall as the strip, so it measures it.
	strip, ok := tt.Find("新建标签页")
	if !ok {
		t.Fatal("no new tab button")
	}
	above, below := picker.Y-strip.Y, strip.H-(picker.Y-strip.Y)-picker.H
	if above < 1 || above != below {
		t.Errorf("the picker leaves %.0f px above and %.0f px below in a %.0f px strip, want the same on both",
			above, below, strip.H)
	}
}

// TestNewTabMenuOffersKinds checks the strip's + asks which kind of request the
// tab holds, as the tree's + does, with no folder to make inside a strip.
func TestNewTabMenuOffersKinds(t *testing.T) {
	a := testApp(t)
	ws := a.space()
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("新建标签页"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("新建 WebSocket") || !tt.HasText("新建 GraphQL") {
		t.Errorf("texts %q, want the kinds the + offers", tt.Texts())
	}
	if tt.HasText("新建文件夹") {
		t.Errorf("texts %q, want no folder in the strip's menu", tt.Texts())
	}
	if err := tt.Click("新建 WebSocket"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(ws.tabs) != 2 || ws.active != 1 {
		t.Fatalf("tabs %d active %d after the socket tab, want 2 tabs with the second active",
			len(ws.tabs), ws.active)
	}
	if got := a.current().kind; got != kindWebSocket {
		t.Errorf("kind %v of the new tab, want a socket", got)
	}
	if got := a.current().bodyType; got != "json" {
		t.Errorf("message format %q of the new socket, want json", got)
	}
}

// TestSidebarFoldsAway checks the tree of a collection folds away from the
// corner of the request pane and comes back from the same corner.
func TestSidebarFoldsAway(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	if !tt.HasText("文件夹 根目录") {
		t.Fatalf("texts %q, want the tree of the collection", tt.Texts())
	}

	if err := tt.Click("收起接口面板"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.sidebarCollapsed {
		t.Error("the tree is not folded after 收起接口面板")
	}
	if tt.HasText("文件夹 根目录") {
		t.Errorf("texts %q, want the tree gone", tt.Texts())
	}
	if !tt.HasText("标签页 Pets") {
		t.Errorf("texts %q, want the request pane still", tt.Texts())
	}

	if err := tt.Click("展开接口面板"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.sidebarCollapsed || !tt.HasText("文件夹 根目录") {
		t.Errorf("folded=%v texts %q, want the tree back", a.sidebarCollapsed, tt.Texts())
	}
}

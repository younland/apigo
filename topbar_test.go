package main

import (
	"fmt"
	"strings"
	"testing"

	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

// collectionsApp is an app with the collections named, all of them open, the
// first one shown.
func collectionsApp(t *testing.T, names ...string) *app {
	t.Helper()
	a := newApp()
	a.store = newStore(t.TempDir())
	for i, name := range names {
		a.store.collections = append(a.store.collections,
			collection{ID: fmt.Sprintf("c%d", i), Name: name, Root: folder{Name: rootFolderName}})
		a.open = append(a.open, fmt.Sprintf("c%d", i))
	}
	a.active = "c0"
	return a
}

// accentPixels counts the pixels of want in the box, as the icon and the name
// of a tab colored with the accent draw them.
func accentPixels(tt *ui.Tester, box ui.Rect, want ui.Color) int {
	n := 0
	img := tt.Image()
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if c := img.RGBAAt(x, y); c.R == want.R && c.G == want.G && c.B == want.B {
				n++
			}
		}
	}
	return n
}

// TestOpenCollectionTabShowsAccent checks the tab of the collection the window
// shows is drawn with the accent, and the tabs not shown are not.
func TestOpenCollectionTabShowsAccent(t *testing.T) {
	a := collectionsApp(t, "测试", "设备后端")
	tt := ui.NewTester(a.view, 1280, 800)

	accent := a.palette().Accent
	open, ok := tt.Find("集合标签 测试")
	if !ok {
		t.Fatal("no tab for the collection being shown")
	}
	if n := accentPixels(tt, open, accent); n == 0 {
		t.Errorf("the shown collection's tab has no accent pixel out of %v, want its icon and name in %v",
			open, accent)
	}
	closed, ok := tt.Find("集合标签 设备后端")
	if !ok {
		t.Fatal("no tab for the other collection")
	}
	if n := accentPixels(tt, closed, accent); n != 0 {
		t.Errorf("the tab of the collection not shown has %d accent pixels, want none", n)
	}

	// The home tab takes the accent too while the window shows it.
	if err := tt.Click("主窗口"); err != nil {
		t.Fatal(err)
	}
	home, _ := tt.Find("主窗口")
	if n := accentPixels(tt, home, accent); n == 0 {
		t.Errorf("the home tab has no accent pixel while the window shows it, want its icon and name in %v", accent)
	}
	if n := accentPixels(tt, open, accent); n != 0 {
		t.Errorf("the collection tab keeps %d accent pixels after the window went home, want none", n)
	}
}

// TestCollectionTabsShowFullName checks a tab keeps the width its name needs:
// the strip scrolls rather than squeeze the names into an ellipsis.
func TestCollectionTabsShowFullName(t *testing.T) {
	const long = "示例接口服务后台"
	a := collectionsApp(t, "测试", long, "设备后端")

	wide := ui.NewTester(a.view, 1280, 800)
	want, ok := wide.Find("集合标签 " + long)
	if !ok {
		t.Fatal("no tab for the long name")
	}
	narrow := ui.NewTester(a.view, 640, 800)
	got, ok := narrow.Find("集合标签 " + long)
	if !ok {
		t.Fatal("no tab for the long name in the narrow window")
	}
	if got.W != want.W {
		t.Errorf("tab width %v in the narrow window, want the %v its name needs", got.W, want.W)
	}

	// Wider than the window, the strip keeps every tab at the size and place
	// its name needs and clips the ones past the edge, rather than squeezing
	// them into an ellipsis.
	tight := ui.NewTester(a.view, 420, 800)
	clipped, ok := tight.Find("集合标签 " + long)
	if !ok {
		t.Fatal("no tab for the long name in the tight window")
	}
	if clipped.X != want.X || clipped.W != want.W {
		t.Errorf("tab %v in the tight window, want the %v its name needs, at the same place",
			clipped, want)
	}

	// The tabs past the edge come into view by scrolling the strip.
	before, ok := tight.Find("集合标签 设备后端")
	if !ok {
		t.Fatal("no tab for the last collection")
	}
	tight.Scroll(200, 20, 120, 0)
	tight.Frame()
	after, _ := tight.Find("集合标签 设备后端")
	if after.X >= before.X {
		t.Errorf("the last tab stayed at x=%v after scrolling the strip, want it moved left", after.X)
	}
}

// TestTopBarScrollsWithoutBar checks the strip of collection tabs keeps
// scrolling its tabs and draws no bar over them: there is no room under the
// tabs for one.
func TestTopBarScrollsWithoutBar(t *testing.T) {
	a := collectionsApp(t, "测试", "示例接口服务后台", "设备后端")
	tt := ui.NewTester(a.view, 420, 800)

	first, ok := tt.Find("集合标签 测试")
	if !ok {
		t.Fatal("no tab for the first collection")
	}
	more, ok := tt.Find("更多")
	if !ok {
		t.Fatal("no more button")
	}
	// The pointer on the strip is what shows a bar of a scroll container.
	tt.Move(first.X+first.W/2, first.Y+first.H/2)
	tt.Frame()

	// The band under the tabs, where a bar of the strip would go.
	surface := a.palette().Surface
	img := tt.Image()
	for y := int(first.Y+first.H) + 1; y < int(first.Y+first.H)+8; y++ {
		for x := int(first.X); x < int(more.X)-4; x++ {
			if c := img.RGBAAt(x, y); c.R != surface.R || c.G != surface.G || c.B != surface.B {
				t.Fatalf("pixel (%d,%d) is %v, want the strip surface %v: a scroll bar shows under the tabs",
					x, y, c, surface)
			}
		}
	}
}

// measureBodyText is how wide s draws in the interface font, through a window of
// its own.
func measureBodyText(s string) float32 {
	var w float32
	ui.Render(func(c *ui.Context) { w = textWidth(c, s) }, 10, 10, 1)
	return w
}

// TestCollectionTabCapsItsWidth checks a long collection name stays inside the
// cap of a tab, shortened with an ellipsis, while a short one keeps the room
// its name draws in.
func TestCollectionTabCapsItsWidth(t *testing.T) {
	const long = "测试3111111111111111111111111111111111111111111111111111111"
	a := collectionsApp(t, "测试1", long)
	tt := ui.NewTester(a.view, 1280, 800)

	capped, ok := tt.Find("集合标签 " + long)
	if !ok {
		t.Fatal("no tab for the long name")
	}
	if capped.W > design.CollectionTabMaxWidth {
		t.Errorf("tab width %v, want no more than the %v cap", capped.W, design.CollectionTabMaxWidth)
	}
	shortened := false
	for _, text := range tt.Texts() {
		if strings.HasPrefix(text, "测试311") && strings.HasSuffix(text, "…") {
			shortened = true
		}
	}
	if !shortened {
		t.Errorf("texts %q, want the long name shortened with an ellipsis", tt.Texts())
	}

	short, ok := tt.Find("集合标签 测试1")
	if !ok {
		t.Fatal("no tab for the short name")
	}
	if short.W >= design.CollectionTabMaxWidth {
		t.Errorf("tab width %v for a short name, want less than the %v cap",
			short.W, design.CollectionTabMaxWidth)
	}
	text, ok := tt.Find("测试1")
	if !ok {
		t.Fatal("no text of the short name")
	}
	if want := measureBodyText("测试1"); text.W < want {
		t.Errorf("text width %v, want at least the %v the name draws in", text.W, want)
	}
}

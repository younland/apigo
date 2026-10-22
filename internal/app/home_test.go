package app

import (
	"fmt"
	"image"
	"math"
	"os"
	"strings"
	"testing"

	"apigo/internal/design"
	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

// homeApp is the home window with n collections, none of them open.
func homeApp(t *testing.T, n int) *app {
	t.Helper()
	a := newApp()
	a.store = store.New(t.TempDir())
	for i := 0; i < n; i++ {
		a.store.Collections = append(a.store.Collections, collection{
			ID: fmt.Sprintf("c%d", i), Name: fmt.Sprintf("集合 %d", i+1), Root: folder{Name: store.RootFolderName},
		})
	}
	return a
}

// TestHomeCollectionListIsCentered checks the rows take the middle two thirds
// of the window: the same stretch of page on both sides of them.
func TestHomeCollectionListIsCentered(t *testing.T) {
	// The row's own padding, its icon and the gap to the name.
	const chrome float32 = 8 + 32 + 12
	for _, width := range []int{1280, 640} {
		a := homeApp(t, 2)
		tt := ui.NewTester(a.view, width, 800)
		name, ok := tt.Find("集合 1")
		if !ok {
			t.Fatalf("no name of the first collection in a %d wide window", width)
		}
		more, ok := tt.Find("集合操作 集合 1")
		if !ok {
			t.Fatalf("no menu of the first collection in a %d wide window", width)
		}
		// The window pads the page by 32 on each side; the rows take two thirds
		// of what is left, in the middle of it.
		content := float32(width) - 64
		wantLeft, wantRight := 32+content/6, 32+content*5/6
		if got := name.X - chrome; !closeEnough(got, wantLeft) {
			t.Errorf("rows start at %v in a %d wide window, want %v", got, width, wantLeft)
		}
		if got := more.X + more.W + 8; !closeEnough(got, wantRight) {
			t.Errorf("rows end at %v in a %d wide window, want %v", got, width, wantRight)
		}
	}
}

// TestHomeCollectionListDrawsNoBar checks a list too long for the window keeps
// scrolling and draws no bar over its right edge.
func TestHomeCollectionListDrawsNoBar(t *testing.T) {
	const width, height = 1280, 800
	a := homeApp(t, 14)
	tt := ui.NewTester(a.view, width, height)

	// The pointer on the list is what shows a bar of a scroll container.
	tt.Move(300, 300)
	tt.Frame()

	// The lane where the bar of the list would go: its right edge, six DIPs
	// wide, and no row there, as the rows take two thirds of the width.
	background := a.palette().Background
	img := tt.Image()
	for x := width - 44; x < width-34; x++ {
		for y := 200; y < height-60; y++ {
			if c := img.RGBAAt(x, y); c.R != background.R || c.G != background.G || c.B != background.B {
				t.Fatalf("pixel (%d,%d) is %v, want the page background %v: a scroll bar shows beside the list",
					x, y, c, background)
			}
		}
	}
}

// closeEnough reports whether two DIPs are the same, as layout rounds them.
func closeEnough(got, want float32) bool { return math.Abs(float64(got-want)) < 0.5 }

// TestHomeCollectionRowHoverTints checks a collection row answers the pointer:
// it takes the hover surface and shows the hand, as a row that opens does.
func TestHomeCollectionRowHoverTints(t *testing.T) {
	a := homeApp(t, 2)
	tt := ui.NewTester(a.view, 1280, 800)
	name, ok := tt.Find("集合 1")
	if !ok {
		t.Fatal("no name of the first collection")
	}
	// The row's own padding, left of its icon: background only.
	at := [2]float32{name.X - 48, name.Y + name.H/2}

	tt.Move(2, 2)
	tt.Frame()
	if got, want := tt.Image().RGBAAt(int(at[0]), int(at[1])), a.palette().Background; !sameColor(got, want) {
		t.Errorf("row background %v away from the pointer, want %v", got, want)
	}
	if got := tt.Cursor(); got == ui.CursorPointer {
		t.Error("the pointer shows the hand away from the row")
	}

	tt.Move(at[0], at[1])
	tt.Frame()
	if got, want := tt.Image().RGBAAt(int(at[0]), int(at[1])), a.palette().SurfaceHover; !sameColor(got, want) {
		t.Errorf("row background %v under the pointer, want the hover surface %v", got, want)
	}
	if got := tt.Cursor(); got != ui.CursorPointer {
		t.Errorf("cursor %v over the row, want %v", got, ui.CursorPointer)
	}
}

// TestHomeCollectionRowOpensOnDoubleClick checks a row opens its collection on
// a double click, and not on a single one: the row is the file of a list.
func TestHomeCollectionRowOpensOnDoubleClick(t *testing.T) {
	a := homeApp(t, 2)
	tt := ui.NewTester(a.view, 1280, 800)
	name, ok := tt.Find("集合 2")
	if !ok {
		t.Fatal("no name of the second collection")
	}
	at := [2]float32{name.X - 48, name.Y + name.H/2}

	tt.ClickAt(at[0], at[1])
	if a.active != "" {
		t.Fatalf("active %q after one click, want the home window still", a.active)
	}
	tt.ClickAt(at[0], at[1])
	if got := a.active; got != "c1" {
		t.Errorf("active %q after a double click, want c1", got)
	}
}

// TestHomeCollectionNameFillsTheRow checks a collection's name takes the room
// its row has — shortened at the 打开 button rather than after a few characters
// — and that the kind it holds is no longer tagged beside it.
func TestHomeCollectionNameFillsTheRow(t *testing.T) {
	name := strings.Repeat("统计接口集合", 8)
	var widths []float32
	for _, window := range []int{1280, 640} {
		a := newApp()
		a.store = store.New(t.TempDir())
		a.store.Collections = []collection{{ID: "c0", Name: name}}
		tt := ui.NewTester(a.view, window, 800)
		tt.Frame()
		text, ok := tt.Find(name)
		if !ok {
			t.Fatalf("no name of the collection at %d wide", window)
		}
		open, ok := tt.Find("打开 " + name)
		if !ok {
			t.Fatalf("no 打开 button at %d wide", window)
		}
		if gap := open.X - (text.X + text.W); gap < 0 || gap > 20 {
			t.Errorf("the name leaves %.0f px before 打开 at %d wide, want it to fill the row",
				gap, window)
		}
		if tt.HasText("HTTP") {
			t.Errorf("texts %q, want no kind tag beside the name", tt.Texts())
		}
		widths = append(widths, text.W)
	}
	if widths[0] <= widths[1] {
		t.Errorf("name widths %v, want the wider window to give it more room", widths)
	}
}

// advanceWidth is how wide s draws in the title size, as the text engine
// advances it.
func advanceWidth(s string) float32 {
	var total float32
	font := ui.Font{Family: design.FontFamily, Size: design.FontSizeTitle, Weight: 400}
	for _, glyph := range ui.Shape(s, font) {
		total += glyph.Advance
	}
	return total
}

// longestInkRun is the longest unbroken run of ink on a row of r: a line struck
// through the text draws one across all of it, where the strokes of its glyphs
// leave gaps.
func longestInkRun(img *image.RGBA, r ui.Rect) int {
	longest := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		run := 0
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if red, _, _, _ := img.RGBAAt(x, y).RGBA(); red < 0xE000 {
				run++
				longest = max(longest, run)
				continue
			}
			run = 0
		}
	}
	return longest
}

// TestHomeMarksMissingCollections checks a collection whose directory another
// program deleted stays on the list, struck through, and that opening it asks
// to take it off rather than opening what is not there.
func TestHomeMarksMissingCollections(t *testing.T) {
	a := newApp()
	a.store = store.New(t.TempDir())
	if _, err := a.store.NewCollection("没了的集合"); err != nil {
		t.Fatal(err)
	}
	col := a.store.Collections[0]
	if err := os.RemoveAll(a.store.CollectionDir(col.ID)); err != nil {
		t.Fatal(err)
	}
	a.refreshMissing()
	if !a.collectionMissing(col.ID) {
		t.Fatal("the collection whose directory is gone is not marked missing")
	}

	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	name, ok := tt.Find(col.Name)
	if !ok {
		t.Fatalf("texts %q, want the collection still on the list", tt.Texts())
	}
	if got, want := longestInkRun(tt.Image(), name), advanceWidth(col.Name); float32(got) < want/2 {
		t.Errorf("the name's longest ink run is %d px of %.0f, want a line struck through it", got, want)
	}

	// Opening it opens nothing: the question comes instead.
	if err := tt.Click("打开 " + col.Name); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.active != "" {
		t.Errorf("active %q, want the home window still", a.active)
	}
	if !tt.HasText("集合不存在") {
		t.Errorf("texts %q, want the question about the missing collection", tt.Texts())
	}
	if err := tt.Click("确认移除集合"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.store.Collections) != 0 {
		t.Errorf("%d collections after removing it, want it off the list", len(a.store.Collections))
	}
}

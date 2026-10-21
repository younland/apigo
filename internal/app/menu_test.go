package app

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// sameColor reports whether the frame drew want at c, ignoring alpha.
func sameColor(c color.RGBA, want ui.Color) bool {
	return c.R == want.R && c.G == want.G && c.B == want.B
}

// TestMenuPanelDrawsBorder checks that a popover menu stands out from the page
// around it: the border of its panel is the light border of every control, not
// the divider color that all but vanishes on a white window.
func TestMenuPanelDrawsBorder(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("请求方法"); err != nil {
		t.Fatal(err)
	}
	row, ok := tt.Find("PATCH")
	if !ok {
		t.Fatal("no PATCH row: the method menu did not open")
	}
	want := a.palette().Border
	// The panel pads its rows by four and draws its border outside them, so
	// the edge is within a few pixels left of a row.
	for x := int(row.X) - 8; x < int(row.X); x++ {
		if sameColor(tt.Image().RGBAAt(x, int(row.Y+row.H/2)), want) {
			return
		}
	}
	t.Errorf("no %v border left of the menu row %v, want the control border of the panel", want, row)
}

// TestMenuRowTintsOnHover checks the row the pointer is on takes the surface
// hover color across the whole width of the menu, so it says which line a
// click would run.
func TestMenuRowTintsOnHover(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("请求方法"); err != nil {
		t.Fatal(err)
	}
	row, ok := tt.Find("PATCH")
	if !ok {
		t.Fatal("no PATCH row: the method menu did not open")
	}
	// The right end of the row, past its text: Find returns the row itself,
	// whose background is the only thing drawn there.
	at := [2]float32{row.X + row.W - 6, row.Y + row.H/2}
	tt.Move(2, 2)
	tt.Frame()
	calm := tt.Image().RGBAAt(int(at[0]), int(at[1]))

	tt.Move(at[0], at[1])
	tt.Frame()
	over := tt.Image().RGBAAt(int(at[0]), int(at[1]))

	if calmp := a.palette().Surface; !sameColor(calm, calmp) {
		t.Errorf("row background %v away from the pointer, want the panel surface %v", calm, calmp)
	}
	if want := a.palette().SurfaceHover; !sameColor(over, want) {
		t.Errorf("row background %v under the pointer, want the hover surface %v", over, want)
	}
}

// TestMenuStillRunsWhatItLists guards the actions of a menu after the rows were
// restyled: the row is the button that runs them.
func TestMenuStillRunsWhatItLists(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("请求方法"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("DELETE"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().method; got != "DELETE" {
		t.Errorf("method %q after choosing DELETE from the menu, want DELETE", got)
	}

	open := a.menuFlag("folder/")
	*open = true
	tt.Frame()
	if err := tt.Click("在根目录里新建文件夹"); err != nil {
		t.Fatal(err)
	}
	addFolderNamed(t, tt, "分组 1")
	if want := 1; len(a.activeCollection().Root.Folders) != want {
		t.Errorf("folders %d after the add menu made one, want %d",
			len(a.activeCollection().Root.Folders), want)
	}
}

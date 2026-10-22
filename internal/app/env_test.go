package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// envApp is the demo app with the environment manager open.
func envApp(t *testing.T) *app {
	t.Helper()
	a := demoApp()
	// Writes land in the test's directory rather than beside the repository.
	a.store.Dir = t.TempDir()
	a.envOpen = true
	a.openEnvManager()
	return a
}

// TestEnvironmentMenuMarksTheActiveOne checks the environments the strip lists
// mark the active one with a tick and the accent, and leave the others plain.
func TestEnvironmentMenuMarksTheActiveOne(t *testing.T) {
	a := demoApp()
	a.envMenuOpen = true
	tt := ui.NewTester(a.view, 1280, 400)
	tt.Frame()

	active, ok := tt.Find("切换到 本地")
	if !ok {
		t.Fatal("no row for the active environment")
	}
	other, ok := tt.Find("切换到 测试")
	if !ok {
		t.Fatal("no row for another environment")
	}
	accent, surface := a.palette().Accent, a.palette().Surface
	img := tt.Image()

	// The active one is tinted, and shows the accent of the tick and name.
	tinted, marked := false, false
	for y := int(active.Y); y < int(active.Y+active.H); y++ {
		for x := int(active.X); x < int(active.X+active.W); x++ {
			c := img.RGBAAt(x, y)
			if !sameColor(c, surface) {
				tinted = true
			}
			if c.R == accent.R && c.G == accent.G && c.B == accent.B {
				marked = true
			}
		}
	}
	if !tinted || !marked {
		t.Errorf("the active environment's row tinted=%v marked=%v, want the accent and a tick",
			tinted, marked)
	}

	// The other one is left on the panel's surface: its band beside the name
	// shows no tint.
	plain := true
	for x := int(other.X+other.W) - 50; x < int(other.X+other.W)-30; x++ {
		if !sameColor(img.RGBAAt(x, int(other.Y+other.H/2)), surface) {
			plain = false
		}
	}
	if !plain {
		t.Error("the row of another environment is marked, want only the active one marked")
	}
}

// TestEnvironmentManagerHasTwoColumns checks the manager lists the
// environments on the left and edits the chosen one on the right, its
// variables one to a line.
func TestEnvironmentManagerHasTwoColumns(t *testing.T) {
	a := envApp(t)
	tt := ui.NewTester(a.view, 1280, 640)

	list, ok := tt.Find("环境 测试")
	if !ok {
		t.Fatal("no environment in the list")
	}
	name, ok := tt.Find("变量名第 1 行")
	if !ok {
		t.Fatal("no variable row in the editor")
	}
	if list.X+list.W >= name.X {
		t.Errorf("the list ends at x %.0f and the editor starts at %.0f, want the editor beside the list",
			list.X+list.W, name.X)
	}
	// One line per variable: its value and its cross share its row.
	value, _ := tt.Find("变量值第 1 行")
	remove, _ := tt.Find("删除变量第 1 行")
	if absFloat(value.Y-name.Y) > 4 || absFloat(remove.Y-name.Y) > 4 {
		t.Errorf("name at y %.0f, value at %.0f, cross at %.0f; want them on one line",
			name.Y, value.Y, remove.Y)
	}
	if _, ok := tt.Find("添加行"); ok {
		t.Error("the manager still shows the button that adds a row")
	}

	// The cross of a variable reads in the danger color.
	danger := a.palette().Danger
	img := tt.Image()
	marked := false
	for y := int(remove.Y); y < int(remove.Y+remove.H); y++ {
		for x := int(remove.X); x < int(remove.X+remove.W); x++ {
			if c := img.RGBAAt(x, y); c.R == danger.R && c.G == danger.G && c.B == danger.B {
				marked = true
			}
		}
	}
	if !marked {
		t.Error("the cross of a variable is not drawn in the danger color")
	}
}

// TestEnvironmentVariableRowDeletesItself checks a variable's cross takes the
// row off the environment, and that the empty row for the next one stays. The
// change waits for the save button to be written.
func TestEnvironmentVariableRowDeletesItself(t *testing.T) {
	a := envApp(t)
	tt := ui.NewTester(a.view, 1280, 640)
	if len(a.envVars.Rows) != 3 {
		t.Fatalf("rows %d, want the two variables and the empty one", len(a.envVars.Rows))
	}

	if err := tt.Click("删除变量第 1 行"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.envVars.Rows) != 2 {
		t.Errorf("rows %d after the deletion, want the one left and the empty one", len(a.envVars.Rows))
	}
	// The environment itself is untouched until the manager is saved.
	env := a.store.Environment(a.envID)
	if env == nil {
		t.Fatal("no environment being edited")
	}
	if len(env.Vars) != 2 {
		t.Errorf("variables %+v before saving, want them left as they were", env.Vars)
	}

	if err := tt.Click("保存环境"); err != nil {
		t.Fatal(err)
	}
	env = a.store.Environment(a.envID)
	if env == nil {
		t.Fatal("no environment after saving")
	}
	for _, v := range env.Vars {
		if v.Key == "BaseURL" {
			t.Errorf("variables %+v, want BaseURL deleted by the save", env.Vars)
		}
	}
}

// TestEnvironmentManagerCancelDropsEdits checks cancelling the manager puts the
// environments back as they stood: nothing it changed is written.
func TestEnvironmentManagerCancelDropsEdits(t *testing.T) {
	a := envApp(t)
	tt := ui.NewTester(a.view, 1280, 640)

	if err := tt.Click("环境 测试"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := a.store.ActiveEnvironment(a.active).Name; got != "测试" {
		t.Fatalf("active environment %q in memory, want the choice", got)
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.envOpen {
		t.Error("the manager stayed open after 取消")
	}
	if got := a.store.ActiveEnvironment(a.active).Name; got != "本地" {
		t.Errorf("active environment %q after 取消, want the one from before", got)
	}
}

// absFloat is a distance, for rows that only need to share a line.
func absFloat(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// TestEnvSelectionFollowsTheClick checks choosing an environment in the
// manager makes it the active one, so the tick follows the choice.
func TestEnvSelectionFollowsTheClick(t *testing.T) {
	a := envApp(t)
	tt := ui.NewTester(a.view, 1280, 800)
	if got := a.store.ActiveEnvironment(a.active).Name; got != "本地" {
		t.Fatalf("active environment %q, want 本地 to begin with", got)
	}

	if err := tt.Click("环境 测试"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := a.store.ActiveEnvironment(a.active).Name; got != "测试" {
		t.Errorf("active environment %q after choosing 测试, want it", got)
	}
	if got := a.envName; got != "测试" {
		t.Errorf("editor name %q, want the environment chosen", got)
	}

	// The tick is drawn on the row of the environment now active.
	row, ok := tt.Find("环境 测试")
	if !ok {
		t.Fatal("no row for the environment chosen")
	}
	accent := a.palette().Accent
	img := tt.Image()
	marked := false
	for y := int(row.Y); y < int(row.Y+row.H); y++ {
		for x := int(row.X); x < int(row.X+row.W); x++ {
			if c := img.RGBAAt(x, y); c.R == accent.R && c.G == accent.G && c.B == accent.B {
				marked = true
			}
		}
	}
	if !marked {
		t.Error("the environment chosen shows no accent, want its tick")
	}
}

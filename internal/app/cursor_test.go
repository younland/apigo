package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// pointerOver puts the pointer on the middle of the control named label and
// returns the cursor the window shows there.
func pointerOver(t *testing.T, tt *ui.Tester, label string) ui.Cursor {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("%s: not found in the window", label)
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	return tt.Cursor()
}

// wantPointer checks that every named control shows the pointer cursor: MyGo
// inherits the cursor down to the innermost clickable element and stops
// there, so each of them sets it itself.
func wantPointer(t *testing.T, tt *ui.Tester, labels ...string) {
	t.Helper()
	for _, label := range labels {
		if got := pointerOver(t, tt, label); got != ui.CursorPointer {
			t.Errorf("%s: cursor %v, want %v", label, got, ui.CursorPointer)
		}
	}
}

// TestWorkspaceControlsShowPointer covers the controls of the workspace in
// one window: the request line, the tab strip, the tree, the key/value list
// and the body panel.
func TestWorkspaceControlsShowPointer(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	wantPointer(t, tt, "发送", "请求方法", "标签页 Pets", "关闭标签页 Pets",
		"新建标签页", "启用参数第 1 行", "添加参数", "文件夹 根目录", "在根目录里新建接口",
		"筛选类型")

	// The body panel and its wand are built only once their tab is chosen.
	if err := tt.Click("请求体"); err != nil {
		t.Fatal(err)
	}
	wantPointer(t, tt, "美化请求体")
}

// TestMenuItemsShowPointer covers the rows of the menus the app draws itself,
// and the body format list, whose rows the widget of the toolkit builds.
func TestMenuItemsShowPointer(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("请求方法"); err != nil {
		t.Fatal(err)
	}
	wantPointer(t, tt, "PATCH")
	if err := tt.Click("请求方法"); err != nil {
		t.Fatal(err)
	}

	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	wantPointer(t, tt, "关闭设置", "更换目录", "强调色 Green", "跟随系统")
	if err := tt.Click("关闭设置"); err != nil {
		t.Fatal(err)
	}

	// The format row is always there: its pills take the pointer like any
	// other button.
	if err := tt.Click("请求体"); err != nil {
		t.Fatal(err)
	}
	wantPointer(t, tt, "请求体格式 JSON", "请求体格式 form-data")
	if err := tt.Click("请求体格式 Text"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().bodyType; got != "text" {
		t.Errorf("body type %q after choosing 请求体格式 Text, want text", got)
	}
}

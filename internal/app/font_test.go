package app

import (
	"testing"

	"apigo/internal/design"
	"apigo/internal/store"

	"github.com/egoist/mygo/ui"
)

func TestInterfaceFontDefaultsToInterWithSansFallback(t *testing.T) {
	a := testApp(t)
	a.settingsOpen = true
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	if a.font != design.FontFamily {
		t.Errorf("default font %q, want %q", a.font, design.FontFamily)
	}
	if a.theme.Font != design.FontWithFallback(design.FontFamily) {
		t.Errorf("theme font %q, want Inter Variable with sans-serif fallback", a.theme.Font)
	}
}

func TestInterfaceFontPickerAppliesAndRemembersChoice(t *testing.T) {
	a := testApp(t)
	a.settingsOpen = true
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	families := interfaceFontFamilies(a.font)
	if len(families) < 2 {
		t.Skip("this system has no alternate interface font")
	}
	choice := families[1]
	if err := tt.Click("界面字体 " + a.font); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("选择字体 " + choice); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.font != choice {
		t.Errorf("font %q after the pick, want %q", a.font, choice)
	}
	if a.theme.Font != design.FontWithFallback(choice) {
		t.Errorf("theme font %q after the pick, want %q", a.theme.Font, design.FontWithFallback(choice))
	}
	loaded := store.New(a.store.Dir)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if loaded.Settings.Font != choice {
		t.Errorf("saved font %q, want %q", loaded.Settings.Font, choice)
	}
}

func TestSettingsItemsUseOneColumn(t *testing.T) {
	a := testApp(t)
	a.settingsOpen = true
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	title, ok := tt.Find("通用")
	if !ok {
		t.Fatal("no section title")
	}
	area, ok := tt.Find("设置项布局")
	if !ok {
		t.Fatal("no settings item area")
	}
	language, ok := tt.Find("界面语言 简体中文")
	if !ok {
		t.Fatal("no language picker")
	}
	font, ok := tt.Find("界面字体 " + a.font)
	if !ok {
		t.Fatal("no font picker")
	}
	if title.X == language.X {
		t.Error("the section title and its controls share one column, want left and right areas")
	}
	if font.X != language.X || font.Y <= language.Y {
		t.Errorf("font at (%.1f, %.1f), want below language at (%.1f, %.1f) in one column",
			font.X, font.Y, language.X, language.Y)
	}
	response, ok := tt.Find("响应体上限（MB）")
	if !ok {
		t.Fatal("no response limit field")
	}
	timeout, ok := tt.Find("请求超时（秒）")
	if !ok {
		t.Fatal("no timeout field")
	}
	if timeout.X != response.X || timeout.Y <= response.Y {
		t.Errorf("timeout at (%.1f, %.1f), want below the response limit at (%.1f, %.1f)",
			timeout.X, timeout.Y, response.X, response.Y)
	}
	want := area.W * 0.9
	if language.W < want-1 || language.W > want+1 {
		t.Errorf("picker width %.1f, want 90%% of the right area %.1f", language.W, want)
	}
}

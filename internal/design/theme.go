package design

import "github.com/egoist/mygo/ui"

// Theme turns a palette into the theme MyGo's widgets draw with, so inputs,
// buttons and scroll bars match the colors of the app around them.
func Theme(m Mode) *ui.Theme {
	p := PaletteOf(m)
	t := ui.LightTheme()
	if p.Dark {
		t = ui.DarkTheme()
	}
	t.Background = p.Background
	t.Surface = p.Surface
	t.SurfaceHover = p.SurfaceHover
	t.SurfacePressed = p.DividerStrong
	t.Border = p.DividerStrong
	t.Text = p.Text
	t.TextMuted = p.TextMuted
	t.Accent = p.Accent
	t.AccentHover = p.AccentHover
	t.AccentPressed = p.AccentPressed
	t.AccentText = p.AccentText
	t.Danger = p.Danger
	t.Warning = p.StatusRedirect
	t.Success = p.StatusSuccess
	if p.Dark {
		t.Selection = ui.RGBA(16, 185, 129, 0.35)
		t.Focus = ui.RGBA(16, 185, 129, 0.60)
		t.Scrollbar = ui.RGBA(255, 255, 255, 0.30)
	} else {
		t.Selection = ui.RGBA(16, 185, 129, 0.22)
		t.Focus = ui.RGBA(16, 185, 129, 0.55)
		t.Scrollbar = ui.RGBA(0, 0, 0, 0.28)
	}
	t.Radius = Radius
	t.Spacing = 4
	t.FontSize = FontSizeBody
	return t
}

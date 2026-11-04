package app

import (
	"strings"

	"apigo/internal/design"

	"github.com/egoist/mygo/ui"
)

const fontPickerHeight float32 = 280

// interfaceFontFamilies lists the proportional fonts the desktop offers, with
// the current choice first so a font missing on this machine remains visible.
func interfaceFontFamilies(selected string) []string {
	families := []string{}
	seen := map[string]bool{}
	add := func(family string) {
		key := strings.ToLower(strings.TrimSpace(family))
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		families = append(families, family)
	}
	add(selected)
	for _, family := range ui.FontFamilies() {
		if !family.Monospaced {
			add(family.Name)
		}
	}
	return families
}

// fontPicker picks the interface font, keeping the generic sans-serif fallback
// for glyphs the chosen family does not cover.
func (a *app) fontPicker(c *ui.Context, p design.Palette) {
	open := a.menuFlag("settings-font")
	btn := selectTrigger(c, p, selectSpec{name: "界面字体 " + a.font, label: a.font, width: 200}).FillWidth()
	if btn.Clicked() {
		*open = !*open
	}
	ui.PopoverBase(c, btn, open, func(panel ui.Element) {
		menuPanel(panel, p, 260)
		ui.Scroll(c).Height(fontPickerHeight).FillWidth().Children(func() {
			for _, family := range interfaceFontFamilies(a.font) {
				selectChoiceRow(c, p, selectChoice{
					name: "选择字体 " + family, label: family, chosen: family == a.font,
					run: func() {
						a.font = family
						*open = false
						a.saveSettingsOrToast(c)
					},
				})
			}
		})
	})
}

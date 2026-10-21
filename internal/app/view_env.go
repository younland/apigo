package app

import (
	"fmt"

	"apigo/internal/design"
	"apigo/internal/icons"
	"apigo/internal/widgets"

	"github.com/egoist/mygo/ui"
)

// envModal manages the environments of the active collection: the list of them
// on the left, the variables of the one being edited on the right, and the
// buttons that keep or drop what it changed.
func (a *app) envModal(c *ui.Context, p design.Palette) {
	// Closed by Escape or its backdrop is a cancel.
	if a.envEdits != nil && !a.envOpen {
		a.discardEnvEdits()
	}
	if !a.envOpen || a.active == "" {
		return
	}
	ui.Modal(c, &a.envOpen, func() {
		ui.Column(c).Width(720).Height(560).Padding(design.PanelPadding).Gap(design.RowGap).
			Background(p.Background).Radius(8).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "环境").Bold().FontSize(design.FontSizeTitle).TextColor(p.Text)
				ui.Text(c, a.activeCollection().Name).FontSize(design.FontSizeTiny).
					TextColor(p.TextMuted)
			})
			a.divider(c, p)
			ui.Row(c).FillWidth().Grow(1).AlignItems(ui.Start).
				Gap(design.PanelPadding * 2).Children(func() {
				a.envList(c, p)
				a.envEditor(c, p)
			})
			a.divider(c, p)
			ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Grow(1)
				if ui.Button(c, "取消").Cursor(ui.CursorPointer).Clicked() {
					a.discardEnvEdits()
					a.envOpen = false
				}
				if saveButton(c, p).Clicked() {
					a.saveEnvEdits(c)
				}
			})
		})
	})
}

// saveButton is the button that keeps what a dialog changed: the accent of the
// app, as a primary action reads.
func saveButton(c *ui.Context, p design.Palette) ui.Element {
	btn := ui.ButtonBase(c).Height(32).Padding(0, 14).Radius(design.Radius).
		Label("保存环境").Background(p.Accent).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(p.AccentHover)
	}
	btn.Children(func() {
		ui.Text(c, "保存").FontSize(design.FontSizeBody).
			TextColor(p.AccentText).SingleLine()
	})
	return btn
}

// envList lists the environments, the one being edited marked, with the button
// that makes another.
func (a *app) envList(c *ui.Context, p design.Palette) {
	ui.Column(c).Width(200).Gap(2).Children(func() {
		for _, env := range a.collectionEnvironments() {
			a.envRow(c, p, env)
		}
		if ui.Button(c, "新建环境").Cursor(ui.CursorPointer).Clicked() {
			// Added in memory: saving the manager writes it, cancelling forgets it.
			env := a.store.NewEnvironment(a.active, "新环境")
			a.editEnvironment(env.ID)
		}
	})
}

// envRow is one environment of the list: the mark that activates it, its name,
// and the cross that deletes it.
func (a *app) envRow(c *ui.Context, p design.Palette, env environment) {
	// Selecting an environment edits it and makes it the one requests send
	// with: the tick follows the choice.
	color, bg := p.TextSecondary, ui.Color{}
	switch {
	case env.Active:
		color = p.Accent
	case env.ID == a.envID:
		color = p.Text
	}
	if env.ID == a.envID {
		bg = p.SurfaceHover
	}
	row := ui.Row(c).FillWidth().Height(30).AlignItems(ui.Center).Gap(6).Padding(0, 6).
		Radius(design.Radius).Background(bg)
	row.Children(func() {
		if ui.ButtonBase(c).Grow(1).Height(30).Label("环境 " + env.Name).
			Justify(ui.Start).Gap(8).Cursor(ui.CursorPointer).Children(func() {
			if env.Active {
				ui.Icon(c, icons.Check).FontSize(12).TextColor(p.Accent)
			} else {
				ui.Row(c).Width(12)
			}
			ui.Text(c, env.Name).FontSize(design.FontSizeBody).TextColor(color).SingleLine()
		}).Clicked() {
			// In memory: saving the manager makes it the active one.
			_ = a.store.SelectEnvironment(env.ID)
			a.editEnvironment(env.ID)
		}
		if ui.ButtonBase(c).Width(22).Height(22).Label("删除环境 " + env.Name).
			Cursor(ui.CursorPointer).Children(func() {
			ui.Icon(c, icons.Trash).FontSize(11).TextColor(p.Danger)
		}).Clicked() {
			a.removeEnvironment(env.ID)
		}
	})
	if env.ID != a.envID && row.Hovered() {
		row.Background(p.SurfaceHover)
	}
}

// envEditor edits the selected environment: its name, and its variables one to
// a line.
func (a *app) envEditor(c *ui.Context, p design.Palette) {
	env := a.store.Environment(a.envID)
	if env == nil {
		ui.Text(c, "选择一个环境来编辑变量").FontSize(design.FontSizeBody).
			TextColor(p.TextMuted)
		return
	}
	ui.Column(c).Grow(1).Height(480).Gap(design.RowGap).Children(func() {
		if ui.TextInput(c, &a.envName).Width(220).Label("环境名称").Changed() {
			// The file follows the name when the manager is saved.
			a.envName = a.store.NameEnvironment(a.envID, a.envName)
		}
		ui.Text(c, "变量可在 URL、参数、请求头与请求体里以 {{name}} 使用").
			FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		ui.Text(c, "环境变量").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		a.envVars.Flush()
		// The variables scroll inside the editor: the window stays the height
		// the settings and the list beside it read at.
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				// Built from a snapshot: a row's own cross deletes it while the
				// list is still being drawn.
				for i := range append([]widgets.Row(nil), a.envVars.Rows...) {
					if i >= len(a.envVars.Rows) {
						break
					}
					a.envVarRow(c, p, i)
				}
			})
		})
	})
}

// envVarRow is one variable on a line: its name, its value, and the cross that
// deletes it, which reads in the danger color. The rows are kept as they are
// typed and written when the manager is saved.
func (a *app) envVarRow(c *ui.Context, p design.Palette, i int) {
	row := &a.envVars.Rows[i]
	ui.Row(c).FillWidth().Height(30).AlignItems(ui.Center).Gap(design.RowGap).Children(func() {
		ui.TextInput(c, &row.Key).Placeholder("变量名").Grow(2).MinWidth(0).
			Label(fmt.Sprintf("变量名第 %d 行", i+1))
		ui.TextInput(c, &row.Value).Placeholder("值").Grow(3).MinWidth(0).
			Label(fmt.Sprintf("变量值第 %d 行", i+1))
		if ui.ButtonBase(c).Width(24).Height(24).
			Label(fmt.Sprintf("删除变量第 %d 行", i+1)).Cursor(ui.CursorPointer).Children(func() {
			ui.Icon(c, icons.Trash).FontSize(12).TextColor(p.Danger)
		}).Clicked() {
			a.envVars.RemoveAt(i)
		}
	})
}

// removeEnvironment deletes an environment, moving the editing and the
// active selection to another one of the collection.
func (a *app) removeEnvironment(id string) {
	env := a.store.Environment(id)
	scope := ""
	if env != nil {
		scope = env.Scope
	}
	// Out of memory only: saving the manager is what takes it off the disk.
	a.store.ForgetEnvironment(id)
	envs := a.store.EnvironmentsFor(scope)
	if a.envID == id {
		a.envID = ""
		if len(envs) > 0 {
			a.editEnvironment(envs[0].ID)
		} else {
			a.editEnvironment("")
		}
	}
	if scope != "" && a.store.ActiveEnvironment(scope) == nil && len(envs) > 0 {
		_ = a.store.SelectEnvironment(envs[0].ID)
	}
}

// languages are the interface languages the settings offer: the app is
// written in one for now.
var languages = []string{"简体中文"}

// appearances are the background choices of the settings, with the icon each
// shows.
var appearances = []struct {
	mode  design.Mode
	icon  *ui.SVG
	label string
}{
	{design.System, icons.Monitor, "跟随系统"},
	{design.Light, icons.Sun, "浅色"},
	{design.Dark, icons.Moon, "深色"},
	{design.Black, icons.MoonLit, "纯黑"},
}

// settingsModal shows the app's settings: the language, the appearance and
// the accent, and where the collections are kept.
func (a *app) settingsModal(c *ui.Context, p design.Palette) {
	if !a.settingsOpen {
		return
	}
	ui.Modal(c, &a.settingsOpen, func() {
		ui.Column(c).Width(720).Padding(24).Gap(20).
			Background(p.Background).Radius(8).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, "设置").Bold().FontSize(design.FontSizeDisplay).TextColor(p.Text)
				ui.Row(c).Grow(1)
				if ui.ButtonBase(c).Width(24).Height(24).Label("关闭设置").Cursor(ui.CursorPointer).
					Children(func() {
						ui.Icon(c, icons.Close).FontSize(12).TextColor(p.TextMuted)
					}).Clicked() {
					a.settingsOpen = false
				}
			})
			a.settingsSection(c, p, "通用", "应用程序中使用的通用设置", func() {
				a.settingsLabel(c, p, "语言")
				a.languagePicker(c, p)
			})
			a.divider(c, p)
			a.settingsSection(c, p, "主题", "自定义您的应用程序主题。", func() {
				a.settingsLabel(c, p, "背景")
				ui.Text(c, a.backgroundNote()).FontSize(design.FontSizeBody).
					TextColor(p.TextMuted)
				a.appearanceButtons(c, p)
				ui.Row(c).Height(design.RowGap)
				a.settingsLabel(c, p, "强调色")
				ui.Text(c, a.accentColor().Name).FontSize(design.FontSizeBody).
					TextColor(p.TextMuted)
				a.accentButtons(c, p)
			})
			a.divider(c, p)
			a.settingsSection(c, p, "数据", "集合、环境与历史存放的位置。", func() {
				a.settingsLabel(c, p, "数据目录")
				ui.Text(c, a.store.DirText()).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).Selectable()
				ui.Row(c).Gap(design.RowGap).Children(func() {
					if ui.Button(c, "更换目录").Cursor(ui.CursorPointer).Clicked() {
						a.chooseDataDir(c)
					}
					if ui.Button(c, "打开目录").Cursor(ui.CursorPointer).Clicked() {
						if err := a.openFolder(a.store.DirText()); err != nil {
							c.Toast("打开目录失败：" + err.Error())
						}
					}
				})
				a.settingsLabel(c, p, "版本")
				ui.Text(c, appVersion()).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary)
			})
		})
	})
}

// settingsSection draws a settings section: what it changes on the left, its
// controls on the right.
func (a *app) settingsSection(c *ui.Context, p design.Palette, title, note string, controls func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(24).Children(func() {
		ui.Column(c).Width(240).Gap(4).Children(func() {
			ui.Text(c, title).Bold().FontSize(design.FontSizeTitle).TextColor(p.Text)
			ui.Text(c, note).FontSize(design.FontSizeBody).TextColor(p.TextMuted)
		})
		ui.Column(c).Grow(1).Gap(6).Children(controls)
	})
}

// settingsLabel is the name of one setting.
func (a *app) settingsLabel(c *ui.Context, p design.Palette, name string) {
	ui.Text(c, name).Bold().FontSize(design.FontSizeBody).TextColor(p.Text)
}

// backgroundNote names the appearance in use, as the settings show it: the
// desktop's own appearance while following it.
func (a *app) backgroundNote() string {
	switch a.mode {
	case design.Light:
		return "浅色"
	case design.Dark:
		return "深色"
	case design.Black:
		return "纯黑"
	}
	if a.resolvedMode() == design.Dark {
		return "系统（暗色）"
	}
	return "系统（亮色）"
}

// languagePicker picks the interface language.
func (a *app) languagePicker(c *ui.Context, p design.Palette) {
	btn := ui.ButtonBase(c).Width(200).Height(36).Padding(0, 12).Gap(8).
		MinWidth(0).Background(p.Surface).Border(design.BorderWidth, p.Border).
		Label("界面语言 " + a.language).Justify(ui.Start).Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Translate).FontSize(14).TextColor(p.TextSecondary)
		ui.Text(c, a.language).FontSize(design.FontSizeBody).TextColor(p.Text).SingleLine()
		ui.Row(c).Grow(1)
		ui.Icon(c, icons.Chevron).FontSize(12).TextColor(p.TextMuted)
	})
	if btn.Clicked() {
		a.languageOpen = !a.languageOpen
	}
	ui.PopoverBase(c, btn, &a.languageOpen, func(panel ui.Element) {
		menuPanel(panel, p, 200)
		for _, language := range languages {
			row := menuRow(ui.ButtonBase(c).Height(28).Label(language).Justify(ui.Start).
				Padding(0, 8).Cursor(ui.CursorPointer).Children(func() {
				ui.Text(c, language).FontSize(design.FontSizeBody).
					TextColor(p.TextSecondary).SingleLine()
			}), p)
			if row.Clicked() {
				a.language, a.languageOpen = language, false
				a.saveSettingsOrToast(c)
			}
		}
	})
}

// appearanceButtons picks the background of the app.
func (a *app) appearanceButtons(c *ui.Context, p design.Palette) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		for _, appearance := range appearances {
			selected := a.mode == appearance.mode
			color := p.TextMuted
			if selected {
				color = p.Accent
			}
			btn := ui.ButtonBase(c).Width(36).Height(32).Label(appearance.label).
				Cursor(ui.CursorPointer).Children(func() {
				ui.Icon(c, appearance.icon).FontSize(16).TextColor(color)
			})
			if selected {
				btn.Background(p.SurfaceHover).Radius(design.Radius)
			}
			if btn.Clicked() {
				a.mode = appearance.mode
				a.saveSettingsOrToast(c)
			}
		}
	})
}

// accentButtons picks the accent color of the app.
func (a *app) accentButtons(c *ui.Context, p design.Palette) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		for _, accent := range design.Accents {
			selected := a.accentColor().Name == accent.Name
			btn := ui.ButtonBase(c).Width(30).Height(30).Label("强调色 " + accent.Name).
				Cursor(ui.CursorPointer).Children(func() {
				swatch := ui.Row(c).Size(18, 18).Radius(9).Background(accent.Base)
				if selected {
					swatch.Border(2, p.Text)
				}
			})
			if btn.Clicked() {
				a.accent = accent.Name
				a.saveSettingsOrToast(c)
			}
		}
	})
}

// saveSettingsOrToast keeps the settings for the next run, saying so when the
// file cannot be written.
func (a *app) saveSettingsOrToast(c *ui.Context) {
	if err := a.rememberSettings(); err != nil {
		c.Toast("保存设置失败：" + err.Error())
	}
}

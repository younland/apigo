package main

import (
	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// envModal manages the environments of the active collection: their names,
// which one is active, and their variables.
func (a *app) envModal(c *ui.Context, p design.Palette) {
	if !a.envOpen || a.active == "" {
		return
	}
	ui.Modal(c, &a.envOpen, func() {
		ui.Column(c).Width(560).Padding(design.PanelPadding).Gap(design.RowGap).
			Background(p.Background).Radius(8).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "环境").Bold().FontSize(design.FontSizeTitle).TextColor(p.Text)
				ui.Text(c, a.activeCollection().Name).FontSize(design.FontSizeTiny).
					TextColor(p.TextMuted)
			})
			for _, env := range a.collectionEnvironments() {
				a.envRow(c, p, env)
			}
			if ui.Button(c, "新建环境").Clicked() {
				if _, err := a.store.addEnvironment(a.active, "新环境"); err != nil {
					c.Toast("新建环境失败：" + err.Error())
					return
				}
				envs := a.collectionEnvironments()
				if len(envs) > 0 {
					a.editEnvironment(envs[len(envs)-1].ID)
				}
			}
			a.divider(c, p)
			a.envEditor(c, p)
		})
	})
}

// envRow is one environment: the mark that activates it, its name, and the
// cross that deletes it.
func (a *app) envRow(c *ui.Context, p design.Palette, env environment) {
	mark, color := "○", p.TextMuted
	if env.Active {
		mark, color = "●", p.Accent
	}
	nameColor := p.TextSecondary
	if env.ID == a.envID {
		nameColor = p.Text
	}
	ui.Row(c).Height(28).FillWidth().AlignItems(ui.Center).Gap(6).Children(func() {
		if ui.ButtonBase(c).Width(20).Height(20).Label("激活环境 " + env.Name).
			Children(func() {
				ui.Text(c, mark).FontSize(design.FontSizeBody).TextColor(color)
			}).Clicked() {
			if err := a.store.activateEnvironment(env.ID); err != nil {
				c.Toast("激活环境失败：" + err.Error())
			}
		}
		if ui.ButtonBase(c).Grow(1).Height(28).Label("编辑环境 " + env.Name).
			Justify(ui.Start).Children(func() {
			ui.Text(c, env.Name).FontSize(design.FontSizeBody).TextColor(nameColor).SingleLine()
		}).Clicked() {
			a.editEnvironment(env.ID)
		}
		if ui.ButtonBase(c).Width(22).Height(22).Label("删除环境 " + env.Name).
			Children(func() {
				ui.Icon(c, icons.Trash).FontSize(11).TextColor(p.TextMuted)
			}).Clicked() {
			a.removeEnvironment(env.ID)
		}
	})
}

// envEditor edits the selected environment's name and variables.
func (a *app) envEditor(c *ui.Context, p design.Palette) {
	env := a.store.environment(a.envID)
	if env == nil {
		ui.Text(c, "选择一个环境来编辑变量").FontSize(design.FontSizeBody).
			TextColor(p.TextMuted)
		return
	}
	ui.Text(c, "环境变量").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
	if ui.TextInput(c, &a.envName).Width(240).Label("环境名称").Changed() {
		name, err := a.store.renameEnvironment(a.envID, a.envName)
		if err != nil {
			c.Toast("重命名环境失败：" + err.Error())
		}
		a.envName = name
	}
	ui.Text(c, "变量可在 URL、参数、请求头与请求体里以 {{name}} 使用").
		FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
	a.envVars.View(c, p)
}

// removeEnvironment deletes an environment, moving the editing and the
// active selection to another one of the collection.
func (a *app) removeEnvironment(id string) {
	env := a.store.environment(id)
	scope := ""
	if env != nil {
		scope = env.Scope
	}
	if err := a.store.deleteEnvironment(id); err != nil {
		return
	}
	envs := a.store.environmentsFor(scope)
	if a.envID == id {
		a.envID = ""
		if len(envs) > 0 {
			a.editEnvironment(envs[0].ID)
		} else {
			a.editEnvironment("")
		}
	}
	if scope != "" && a.store.activeEnvironment(scope) == nil && len(envs) > 0 {
		_ = a.store.activateEnvironment(envs[0].ID)
	}
}

// settingsModal shows the appearance, where the collections are kept, and
// the version.
func (a *app) settingsModal(c *ui.Context, p design.Palette) {
	if !a.settingsOpen {
		return
	}
	ui.Modal(c, &a.settingsOpen, func() {
		ui.Column(c).Width(420).Padding(design.PanelPadding).Gap(design.RowGap).
			Background(p.Background).Radius(8).Children(func() {
			ui.Text(c, "设置").Bold().FontSize(design.FontSizeTitle).TextColor(p.Text)
			ui.Text(c, "外观").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
			if ui.Button(c, a.mode.String()).Clicked() {
				a.mode = a.mode.Next()
			}
			ui.Text(c, "数据目录").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
			ui.Text(c, a.store.dirText()).FontSize(design.FontSizeTiny).
				TextColor(p.TextSecondary).Selectable()
			ui.Text(c, "版本").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
			ui.Text(c, appVersion()).FontSize(design.FontSizeTiny).TextColor(p.TextSecondary)
		})
	})
}

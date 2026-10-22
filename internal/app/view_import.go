package app

import (
	"fmt"
	"path/filepath"

	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
)

// importModal draws the 导入项目 dialog: where the document comes from, what a
// document makes of a collection, and the button that writes it. It opens at the
// top of the window, as the other dialogs of the app do, so the list under it
// stays in view.
func (a *app) importModal(c *ui.Context, p design.Palette) {
	if !a.importDialog.open {
		return
	}
	ui.DialogBase(c, &a.importDialog.open, func(backdrop, panel ui.Element) {
		topBackdrop(backdrop)
		panel.Width(importWidth).Padding(importPadding).Gap(importGap).Radius(8).
			Background(p.Background).Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
		a.importTitle(c, p)
		a.importFormat(c, p)
		a.importHint(c, p)
		a.importSourceTabs(c, p)
		a.importBody(c, p)
		a.importNote(c, p)
		a.importActions(c, p)
	})
}

// importTitle is the head of the dialog: what it does, and the cross that puts
// it away.
func (a *app) importTitle(c *ui.Context, p design.Palette) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Text(c, "导入项目").Grow(1).Bold().FontSize(design.FontSizeDisplay).TextColor(p.Text)
		if ui.ButtonBase(c).Width(24).Height(24).Label("关闭导入").Cursor(ui.CursorPointer).
			Children(func() {
				ui.Icon(c, icons.Close).FontSize(12).TextColor(p.TextMuted)
			}).Clicked() {
			a.closeImportDialog()
		}
	})
}

// importFormat is the source format of the dialog: the one format it reads,
// drawn as the picker the other lists of the app are drawn with, so the
// formats it grows read in the same place.
func (a *app) importFormat(c *ui.Context, p design.Palette) {
	ui.Row(c).FillWidth().Height(importFormatHeight).Padding(0, 12).Gap(10).AlignItems(ui.Center).
		Radius(8).Border(design.BorderWidth, p.Border).Children(func() {
		ui.Row(c).Size(28, 28).Radius(6).Background(p.SurfaceHover).Center().Children(func() {
			ui.Icon(c, icons.Doc).FontSize(16).TextColor(p.Accent)
		})
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Text(c, "数据源格式").FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
			ui.Text(c, "OpenAPI/Swagger").FontSize(design.FontSizeBody).TextColor(p.Text)
		})
		ui.Icon(c, icons.Chevron).FontSize(12).TextColor(p.TextMuted)
	})
}

// importBody draws what the dialog reads from: the summary of a document that
// is already parsed, the URL field of the 导入方式 it is on, or the area a file
// is dropped on.
func (a *app) importBody(c *ui.Context, p design.Palette) {
	switch {
	case a.importDialog.doc != nil:
		a.importSummary(c, p)
	case a.importDialog.source == importFromURL:
		a.importURLField(c, p)
	default:
		a.importDropZone(c, p)
	}
}

// importHint says what the dialog reads, with the note that unfolds into what a
// document becomes.
func (a *app) importHint(c *ui.Context, p design.Palette) {
	ui.Column(c).FillWidth().Padding(10, 12).Gap(6).Radius(8).
		Border(design.BorderWidth, p.Border).Background(p.Surface).Children(func() {
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "支持导入 OpenAPI 3.0、3.1 或 Swagger 2.0 数据格式的 JSON 或 YAML 文件。").
				FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
			ui.Row(c).Grow(1)
			link := ui.ButtonBase(c).Height(20).Label("查看详细说明").Cursor(ui.CursorPointer).
				Children(func() {
					ui.Text(c, "查看详细说明").FontSize(design.FontSizeBody).
						TextColor(p.Accent).Underline()
				})
			if link.Clicked() {
				a.importDialog.details = !a.importDialog.details
			}
		})
		if !a.importDialog.details {
			return
		}
		ui.Text(c, "每个 path 的每个方法成为一个接口，名字取 summary，没有时取 operationId；"+
			"tags 的第一个分组成为文件夹，没有 tag 的接口放在根目录；"+
			"query 与 header 参数填进参数与请求头，requestBody 的 JSON 示例填进请求体；"+
			"文档同名的集合会被更新：同名接口覆盖，其余保留。").
			FontSize(design.FontSizeTiny).TextColor(p.TextMuted).Wrap()
	})
}

// importSourceTabs picks where the document comes from: a file of the desktop,
// or a URL.
func (a *app) importSourceTabs(c *ui.Context, p design.Palette) {
	ui.Row(c).FillWidth().Height(importTabHeight).Padding(3).Gap(3).Radius(8).
		Background(p.SurfaceHover).Children(func() {
		a.importSourceTab(c, p, importFromFile, "上传文件")
		a.importSourceTab(c, p, importFromURL, "URL 导入")
	})
}

// importSourceTab is one source of the strip: filled with the background while
// it is the one shown, as the chosen tab of a segmented control is.
func (a *app) importSourceTab(c *ui.Context, p design.Palette, source int, label string) {
	chosen := a.importDialog.source == source
	color := p.TextSecondary
	tab := ui.ButtonBase(c).Grow(1).Height(importTabHeight - 6).Radius(6).
		Label("导入方式 " + label).Cursor(ui.CursorPointer).Children(func() {
		ui.Text(c, label).FontSize(design.FontSizeBody).TextColor(color)
	})
	if chosen {
		tab.Background(p.Background).Border(design.BorderWidth, p.Border)
		color = p.Text
	}
	if tab.Clicked() && !chosen {
		a.importDialog.source = source
	}
}

// importDropZone is the area a document is chosen from: a click opens the
// desktop's file dialog, and a file dropped on it is read as it lands.
func (a *app) importDropZone(c *ui.Context, p design.Palette) {
	border := p.Border
	zone := ui.Column(c).FillWidth().Height(importDropHeight).Center().Gap(10).
		Radius(8).Border(design.BorderWidth, border).BorderStyle(ui.BorderDashed).
		Background(p.Surface).Label("选择文档文件").Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icons.Upload).FontSize(24).TextColor(p.TextMuted)
		ui.Text(c, "点击或拖拽文件到本区域导入").
			FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
	})
	// The zone takes the files dragged over it from another application, and
	// says so while one is over it.
	if zone.FileDragOver() {
		zone.Border(design.BorderWidth, p.Accent).Background(p.SurfaceHover)
	}
	if files := zone.DroppedFiles(); len(files) > 0 {
		a.readImportFile(files[0])
	}
	if zone.Clicked() {
		a.chooseImportDoc(c)
	}
}

// importURLField is the URL a document is read from.
func (a *app) importURLField(c *ui.Context, p design.Palette) {
	row := ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center)
	row.Children(func() {
		input := ui.TextInput(c, &a.importDialog.url).Grow(1).MinWidth(0).
			Placeholder("https://api.example.com/openapi.json").Label("文档 URL")
		submitted := input.Submitted()
		button := ui.PrimaryButton(c, "读取").Cursor(ui.CursorPointer).
			Disabled(a.importDialog.busy)
		if (button.Clicked() || submitted) && !a.importDialog.busy {
			a.fetchImportDoc()
		}
	})
	ui.Text(c, "支持 JSON 与 YAML，文档指向文档本身而不是页面").
		FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
}

// importSummary says what the parsed document makes: its title, the folders and
// interfaces it holds, and whether the collection it becomes is already there.
func (a *app) importSummary(c *ui.Context, p design.Palette) {
	doc := a.importDialog.doc
	if doc == nil {
		return
	}
	name, existing := a.importTarget()
	ui.Column(c).FillWidth().Padding(14).Gap(6).Radius(8).
		Border(design.BorderWidth, p.Border).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Icon(c, icons.Check).FontSize(14).TextColor(p.Accent)
			ui.Text(c, fmt.Sprintf("已读取 %d 个接口", doc.RequestCount())).
				Grow(1).FontSize(design.FontSizeTitle).TextColor(p.Text)
			if ui.Button(c, "重新选择").Cursor(ui.CursorPointer).Clicked() {
				a.importDialog.doc, a.importDialog.note = nil, ""
			}
		})
		ui.Textf(c, "%s · %d 个文件夹 · 集合「%s」", importSpecText(doc.Spec), doc.FolderCount(), name).
			FontSize(design.FontSizeBody).TextColor(p.TextSecondary)
		if source := a.importSource(); source != "" {
			ui.Text(c, source).FontSize(design.FontSizeTiny).TextColor(p.TextMuted).SingleLine()
		}
		if existing {
			ui.Text(c, "已有同名集合：导入会覆盖同名接口，其余接口保持不变").
				FontSize(design.FontSizeTiny).TextColor(p.StatusRedirect)
		}
	})
}

// importSpecText names the format a document is written in.
func importSpecText(spec string) string {
	switch {
	case spec == "":
		return "OpenAPI/Swagger"
	case len(spec) > 2 && spec[:2] == "2.":
		return "Swagger " + spec
	default:
		return "OpenAPI " + spec
	}
}

// importSource is where the parsed document came from: its file, or its URL.
func (a *app) importSource() string {
	if a.importDialog.file != "" {
		return filepath.Base(a.importDialog.file)
	}
	return a.importDialog.url
}

// importNote says what reading a document came to: what failed, or what the
// dialog is waiting for.
func (a *app) importNote(c *ui.Context, p design.Palette) {
	if a.importDialog.note == "" {
		return
	}
	color := p.TextMuted
	if a.importDialog.doc == nil {
		color = p.Danger
	}
	ui.Text(c, a.importDialog.note).FontSize(design.FontSizeTiny).TextColor(color).Wrap()
}

// importActions are the buttons of the dialog: the one that writes the
// document, and the one that puts the dialog away.
func (a *app) importActions(c *ui.Context, p design.Palette) {
	ui.Row(c).FillWidth().Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1)
		if ui.Button(c, "取消导入").Cursor(ui.CursorPointer).Clicked() {
			a.closeImportDialog()
		}
		ready := a.importDialog.doc != nil && !a.importDialog.busy
		if ui.PrimaryButton(c, "导入").Cursor(ui.CursorPointer).Disabled(!ready).Clicked() {
			a.applyImport(c)
		}
	})
}

package app

import (
	"apigo/internal/design"
	"apigo/internal/icons"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// responseBodyView draws a response the way its format asks for: code in the
// editor, a picture in the pane, a player for a sound, and a card with a way to
// keep a file.
func (a *app) responseBodyView(c *ui.Context, p design.Palette, t *tab) {
	switch t.view {
	case bodyImage:
		a.responsePicture(c, p, t)
	case bodyAudio:
		a.responsePlayer(c, p, t)
	case bodyVideo, bodyFile:
		a.responseFile(c, p, t)
	default:
		editor.View(c, &t.respEdit).Grow(1).FillWidth()
	}
}

// responsePicture shows an image a response holds, with what it is under it.
func (a *app) responsePicture(c *ui.Context, p design.Palette, t *tab) {
	ui.Column(c).Grow(1).FillWidth().Padding(design.PanelPadding).
		Gap(design.RowGap).Center().Children(func() {
		ui.Image(c, t.picture).Fit(ui.Contain).Label("响应图片").AspectRatio(16.0 / 9)
		ui.Text(c, t.pictureNote+" · "+mediaDetail(t.resp)).FontSize(design.FontSizeTiny).
			TextColor(p.TextMuted).SingleLine()
	})
}

// responsePlayer plays a sound a response holds: a button, how far it has
// played and how long it is, and a way to keep it.
func (a *app) responsePlayer(c *ui.Context, p design.Palette, t *tab) {
	// A player moves on its own: the pane asks for a frame now and then while
	// the sound runs, so the bar follows it.
	if t.audio != nil && t.audio.Playing() {
		c.After(resumeEvery)
	}
	ui.Column(c).Grow(1).FillWidth().Padding(design.PanelPadding).
		Gap(design.RowGap).Center().Children(func() {
		ui.Row(c).Gap(design.RowGap).AlignItems(ui.Center).Children(func() {
			a.playButton(c, p, t)
			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, downloadName(t.resp, t.url)).FontSize(design.FontSizeBody).
					TextColor(p.Text).SingleLine()
				a.playBar(c, p, t)
				ui.Text(c, playText(t)+" · "+mediaDetail(t.resp)).
					FontSize(design.FontSizeTiny).TextColor(p.TextMuted).SingleLine()
			})
		})
	})
}

// playButton starts or pauses the sound of the tab.
func (a *app) playButton(c *ui.Context, p design.Palette, t *tab) {
	icon, label := icons.Play, "播放"
	playing := t.audio != nil && t.audio.Playing()
	if playing {
		icon, label = icons.Pause, "暂停"
	}
	btn := ui.ButtonBase(c).Width(44).Height(44).Radius(22).Background(p.Accent).
		Label(label).Cursor(ui.CursorPointer).Children(func() {
		ui.Icon(c, icon).FontSize(16).TextColor(p.AccentText)
	})
	if !btn.Clicked() {
		return
	}
	if playing {
		t.audio.Pause()
		return
	}
	t.audio.Play()
}

// playBar draws how far the sound has played, as a thin line under its name.
func (a *app) playBar(c *ui.Context, p design.Palette, t *tab) {
	ui.Row(c).Width(240).Height(4).Radius(2).Background(p.Divider).
		Children(func() {
			ui.Row(c).Width(240 * playFraction(t)).Height(4).Radius(2).
				Background(p.Accent)
		})
}

// responseFile offers a body the pane cannot show as a file to keep: its name,
// what it is, and a button that saves it to the downloads directory.
func (a *app) responseFile(c *ui.Context, p design.Palette, t *tab) {
	icon := icons.File
	if t.view == bodyVideo {
		icon = icons.Video
	}
	ui.Column(c).Grow(1).FillWidth().Center().Gap(design.RowGap).Children(func() {
		ui.Icon(c, icon).FontSize(32).TextColor(p.TextMuted)
		ui.Text(c, downloadName(t.resp, t.url)).FontSize(design.FontSizeBody).Bold().
			TextColor(p.Text).SingleLine()
		ui.Text(c, mediaDetail(t.resp)).FontSize(design.FontSizeTiny).
			TextColor(p.TextMuted).SingleLine()
		if t.view == bodyVideo {
			ui.Text(c, "视频暂不支持内嵌播放，保存后可用系统播放器打开").
				FontSize(design.FontSizeTiny).TextColor(p.TextMuted)
		}
		if t.mediaErr != nil {
			ui.Text(c, "无法直接显示："+t.mediaErr.Error()).FontSize(design.FontSizeTiny).
				TextColor(p.TextMuted)
		}
	})
}

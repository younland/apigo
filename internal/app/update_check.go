package app

import (
	"context"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// checkUpdatesAtStart looks for a newer version when the app opens, for the
// readers who asked it to: a check reaches the network, so it waits for the
// window and says nothing until it has an answer.
func (a *app) checkUpdatesAtStart() {
	if !a.autoUpdate {
		return
	}
	a.checkUpdates(true)
}

// checkUpdates asks for the newest release of the app and says what it found.
// With ask, a newer version offers to download its verified installer.
func (a *app) checkUpdates(ask bool) {
	if a.checkingUpdate || a.downloadingUpdate {
		return
	}
	check := a.checkRelease
	if check == nil {
		check = latestRelease
	}
	if a.after == nil {
		// No window to hand work back to, as in a test: the check runs here.
		rel, err := check(context.Background())
		a.noteUpdate(rel, err, ask)
		return
	}
	a.checkingUpdate = true
	a.updateNote = "正在检查更新…"
	go func() {
		rel, err := check(context.Background())
		a.after(func() {
			a.checkingUpdate = false
			a.noteUpdate(rel, err, ask)
		})
	}()
}

// noteUpdate says what a check found, and asks before downloading an update.
func (a *app) noteUpdate(rel release, err error, ask bool) {
	switch {
	case err != nil:
		a.updateNote = "检查更新失败：" + err.Error()
	case !newerVersion(rel.Version, currentVersion()):
		a.updateNote = "已是最新版本 " + currentVersion()
	default:
		a.updateNote = "有新版本 " + rel.Version
		if !ask {
			return
		}
		a.ask(confirm{
			title:  "发现新版本 " + rel.Version,
			body:   "当前版本 " + currentVersion() + "，新版本已发布。将自动下载并校验更新包，然后启动安装程序。",
			action: "下载并更新",
			run: func(c *ui.Context) {
				a.startUpdateDownload(rel)
			},
		})
	}
}

// currentVersion is the version the app runs as, "0.0.0" for a build that
// names none, so that a release reads as newer than it.
func currentVersion() string {
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return "0.0.0"
}

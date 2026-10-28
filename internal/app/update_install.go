package app

import (
	"context"
	"fmt"
	"os"
)

func downloadRelease(ctx context.Context, rel release, progress updateProgress) (string, error) {
	dir, err := os.MkdirTemp("", "apigo-update-")
	if err != nil {
		return "", err
	}
	path, err := rel.download(ctx, dir, progress)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return path, nil
}

func (a *app) startUpdateDownload(rel release) {
	if a.downloadingUpdate {
		return
	}
	a.downloadingUpdate = true
	a.updateDownloaded, a.updateDownloadSize = 0, 0
	a.updateNote = "正在下载更新…"
	a.updateToast = "正在后台下载并校验更新包"
	fetch := a.fetchUpdate
	if fetch == nil {
		fetch = downloadRelease
	}
	run := func() (string, error) {
		return fetch(context.Background(), rel, a.reportUpdateProgress())
	}
	if a.after == nil {
		path, err := run()
		a.finishUpdateDownload(path, err)
		return
	}
	go func() {
		path, err := run()
		a.after(func() { a.finishUpdateDownload(path, err) })
	}()
}

func (a *app) reportUpdateProgress() updateProgress {
	lastPercent := -1
	return func(downloaded, total int64) {
		percent := 0
		if total > 0 {
			percent = int(downloaded * 100 / total)
		} else if downloaded > 0 {
			return // an unknown total has no useful percentage to repeat
		}
		if percent == lastPercent {
			return
		}
		lastPercent = percent
		report := func() {
			if !a.downloadingUpdate {
				return
			}
			a.updateDownloaded, a.updateDownloadSize = downloaded, total
			a.updateNote = updateDownloadText(downloaded, total)
		}
		if a.after == nil {
			report()
			return
		}
		a.after(report)
	}
}

func updateDownloadText(downloaded, total int64) string {
	if total > 0 {
		return fmt.Sprintf("正在下载更新… %d%%", downloaded*100/total)
	}
	return fmt.Sprintf("正在下载更新… %.1f MB", float64(downloaded)/(1<<20))
}

func (a *app) finishUpdateDownload(path string, err error) {
	a.downloadingUpdate = false
	if err != nil {
		a.updateNote = "下载更新失败：" + err.Error()
		a.updateToast = a.updateNote
		return
	}
	if !a.saveTabsForUpdate() {
		a.updateNote = "接口保存失败，更新未启动"
		a.updateToast = a.saveErr
		return
	}
	if a.opener == nil {
		a.updateNote = "当前环境不能启动安装程序"
		a.updateToast = a.updateNote
		return
	}
	a.updateNote = "更新包已下载，正在启动安装程序…"
	a.updateToast = a.updateNote
	if err := a.opener(path); err != nil {
		a.updateNote = "启动安装程序失败：" + err.Error()
		a.updateToast = a.updateNote
		return
	}
	if a.quit != nil {
		a.quit()
	}
}

func (a *app) saveTabsForUpdate() bool {
	a.saveErr = ""
	for _, ws := range a.spaces {
		for _, t := range ws.tabs {
			a.saveTab(t)
			if a.saveErr != "" {
				return false
			}
		}
	}
	return true
}

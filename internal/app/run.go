package app

import (
	"log"

	// The interface font is embedded beside its own package: importing it
	// registers the font the theme draws with.
	_ "apigo/internal/assets"
	"apigo/internal/store"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Run opens the window and runs the application: it returns once the desktop
// closes it.
func Run() {
	a := newApp()
	mygo.App.WhenReady(func() {
		// The collections live where the user put them, or in the app's own
		// data directory.
		a.baseDir = store.DefaultDir()
		a.store = store.New(store.DirAt(a.baseDir))
		if err := a.store.Load(); err != nil {
			log.Printf("读取本地集合失败：%v", err)
		}
		// The appearance, accent and language of the last run.
		a.applySettings()
		window := mygo.NewWindow(mygo.WindowOptions{
			Title:     "apigo",
			Width:     1280,
			Height:    800,
			MinWidth:  1024,
			MinHeight: 640,
			// Opens where the user left it last time.
			StateKey: "main",
			// The window shows the interface MyGo draws, not a web page.
			Content: ui.View(a.view),
		})
		// Responses land on the window's thread, never on the one that
		// waited for the network.
		a.after = window.Update
		// A collection's folder opens in the desktop's file manager, and the
		// settings ask it for another data directory.
		a.opener = mygo.Shell.OpenPath
		a.chooseDir = chooseDirectory
		a.chooseFile = chooseUpload
		// The tree follows the files another program changes in the open
		// collection's directory.
		a.watch = watchFiles
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// chooseDirectory asks the desktop's file manager for a folder, "" when the
// user cancels.
func chooseDirectory() (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title:             "选择数据目录",
		ButtonLabel:       "选择",
		Directory:         true,
		CreateDirectories: true,
	})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

// chooseUpload asks the desktop for a file to upload, "" when the user
// cancels.
func chooseUpload() (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title:       "选择要上传的文件",
		ButtonLabel: "选择",
	})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

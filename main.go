package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		a.store = newStore(defaultStoreDir())
		if err := a.store.load(); err != nil {
			log.Printf("读取本地集合失败：%v", err)
		}
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
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

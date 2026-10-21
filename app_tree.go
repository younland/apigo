package main

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// folderQuestion asks where something of the tree goes, and what it is called
// when the dialog names it: the folder offered first, the label of the button
// that goes ahead, and what runs with the answers.
type folderQuestion struct {
	open   bool
	title  string
	action string
	folder string   // the folder chosen, "" for the 根目录
	paths  []string // the folders offered, 根目录 first
	// names asks for a name too: the title of the field, and what it holds.
	names     bool
	nameTitle string
	name      string
	run       func(c *ui.Context, folder, name string) error
}

// askFolder asks a question about a folder of the active collection.
func (a *app) askFolder(q folderQuestion) {
	col := a.store.collection(a.active)
	if col == nil {
		return
	}
	q.open, q.paths = true, nil
	for _, f := range foldersOf(col.Root) {
		q.paths = append(q.paths, f.Path)
	}
	a.where = q
}

// askNewFolder asks for the name of a folder added to one of the tree's
// folders: a level of the tree holds one folder of a name, no more.
func (a *app) askNewFolder(folderPath string) {
	a.askFolder(folderQuestion{
		title:     "新建文件夹",
		action:    "新建",
		folder:    folderPath,
		names:     true,
		nameTitle: "文件夹名称",
		run: func(c *ui.Context, folder, name string) error {
			_, err := a.store.addFolder(a.active, folder, name)
			if err != nil {
				return err
			}
			c.Toast("已新建文件夹「" + name + "」")
			return nil
		},
	})
	a.where.paths = []string{folderPath}
}

// saveCurrent writes the request the window shows: a request that has no file
// yet asks where in the collection it goes, one that has a file is written back
// to it. Ctrl+S and the tab's menu both come here.
func (a *app) saveCurrent(c *ui.Context) {
	t := a.current()
	if t == nil || a.active == "" {
		return
	}
	if t.path == "" {
		a.askWhereToSave(t)
		return
	}
	if err := a.store.updateRequest(t.collectionID, t.path, savedFromTab(t)); err != nil {
		c.Toast("保存失败：" + err.Error())
		return
	}
	t.markSaved()
	c.Toast("已保存「" + t.title + "」")
}

// askWhereToSave asks for the folder of a request that has no file yet, and
// for the name it is saved under, offering the folder its + stood on.
func (a *app) askWhereToSave(t *tab) {
	col := a.active
	a.askFolder(folderQuestion{
		title:     "保存接口",
		action:    "保存",
		folder:    t.folder,
		names:     true,
		nameTitle: "接口名称",
		name:      t.title,
		run: func(c *ui.Context, folder, name string) error {
			req, err := a.store.addRequest(col, folder, name, savedFromTab(t))
			if err != nil {
				return err
			}
			t.collectionID, t.title, t.path, t.folder = col, req.Name, req.Path, folder
			t.markSaved()
			c.Toast("已保存「" + req.Name + "」到「" + pathName(folder) + "」")
			return nil
		},
	})
}

// askToMoveRequest asks where an interface goes, offering every folder of the
// collection but the one it is in.
func (a *app) askToMoveRequest(requestPath string) {
	a.askFolder(folderQuestion{
		title:  "移动接口",
		action: "移动",
		folder: parentPath(requestPath),
		run: func(c *ui.Context, folder, _ string) error {
			req, err := a.store.moveRequest(a.active, requestPath, folder)
			if err != nil {
				return err
			}
			a.retargetTabs(requestPath, req.Path)
			c.Toast("已移动到「" + pathName(folder) + "」")
			return nil
		},
	})
}

// askToMoveFolder asks where a folder goes, leaving out the folders inside it,
// which it would move into itself.
func (a *app) askToMoveFolder(folderPath string) {
	q := folderQuestion{
		title:  "移动文件夹",
		action: "移动",
		folder: parentPath(folderPath),
		run: func(c *ui.Context, folder, _ string) error {
			moved, err := a.store.moveFolder(a.active, folderPath, folder)
			if err != nil {
				return err
			}
			a.retargetTabs(folderPath, moved.Path)
			c.Toast("已移动到「" + pathName(folder) + "」")
			return nil
		},
	}
	a.askFolder(q)
	// A folder cannot move into itself, nor into what it holds.
	kept := a.where.paths[:0]
	for _, path := range a.where.paths {
		if path == folderPath || strings.HasPrefix(path, folderPath+"/") {
			continue
		}
		kept = append(kept, path)
	}
	a.where.paths = kept
}

// askToDeleteRequest asks before an interface is deleted for good.
func (a *app) askToDeleteRequest(requestPath string) {
	name := pathName(requestPath)
	a.ask(confirm{
		title:  "删除接口",
		body:   "接口「" + name + "」的文件将被删除，无法恢复。",
		action: "删除接口",
		danger: true,
		run: func(c *ui.Context) {
			if err := a.store.deleteRequest(a.active, requestPath); err != nil {
				c.Toast("删除失败：" + err.Error())
				return
			}
			a.forgetTabsUnder(requestPath)
			c.Toast("已删除「" + name + "」")
		},
	})
}

// askToDeleteFolder asks before a folder is deleted with everything it holds.
func (a *app) askToDeleteFolder(folderPath string) {
	name := pathName(folderPath)
	held := 0
	if col := a.store.collection(a.active); col != nil {
		if f := findFolder(&col.Root, folderPath); f != nil {
			held = countRequests(*f)
		}
	}
	a.ask(confirm{
		title:  "删除文件夹",
		body:   "文件夹「" + name + "」及其中的 " + strconv.Itoa(held) + " 个接口、子文件夹都将在磁盘上删除，无法恢复。",
		action: "删除文件夹",
		danger: true,
		run: func(c *ui.Context) {
			if err := a.store.deleteFolder(a.active, folderPath); err != nil {
				c.Toast("删除失败：" + err.Error())
				return
			}
			a.forgetTabsUnder(folderPath)
			c.Toast("已删除「" + name + "」")
		},
	})
}

// forgetTabsUnder closes the tabs of the interfaces at or inside a tree path,
// without writing them back: what they held is gone.
func (a *app) forgetTabsUnder(treePath string) {
	for _, id := range a.open {
		ws := a.spaces[id]
		if ws == nil {
			continue
		}
		kept := ws.tabs[:0]
		for _, t := range ws.tabs {
			if t.path == treePath || strings.HasPrefix(t.path, treePath+"/") {
				a.dropTabMenu(t)
				continue
			}
			kept = append(kept, t)
		}
		ws.tabs = kept
		if ws.empty() {
			ws.active = 0
			continue
		}
		if ws.active >= len(ws.tabs) {
			ws.active = len(ws.tabs) - 1
		}
	}
}

// retargetTabs moves the tabs of a tree path to where it went.
func (a *app) retargetTabs(from, to string) {
	for _, ws := range a.spaces {
		for _, t := range ws.tabs {
			if t.path == from {
				t.path = to
				continue
			}
			if strings.HasPrefix(t.path, from+"/") {
				t.path = to + strings.TrimPrefix(t.path, from)
			}
		}
	}
}

// accentTint is how much of the accent shows under an interface a tab has
// open: enough to pick it out of the tree, light enough to read the name on.
const accentTint float32 = 0.12

// activeShows reports whether the tab the workspace shows holds the interface
// at a tree path: the tree tints the request being worked on, not every request
// that has a tab. A draft has no file of its own, so it tints nothing.
func (a *app) activeShows(treePath string) bool {
	t := a.current()
	if t == nil || treePath == "" || t.path == "" {
		return false
	}
	return t.path == treePath
}

// tabOfPath returns where the tab of an interface is kept, or -1 when no tab
// shows it.
func (a *app) tabOfPath(ws *workspace, treePath string) int {
	for i, t := range ws.tabs {
		if t.path != "" && t.path == treePath {
			return i
		}
	}
	return -1
}

// askRenameRequest asks for the new name of an interface, keeping it in the
// folder it is in.
func (a *app) askRenameRequest(requestPath string) {
	a.askFolder(folderQuestion{
		title:     "重命名接口",
		action:    "重命名",
		folder:    parentPath(requestPath),
		names:     true,
		nameTitle: "接口名称",
		name:      pathName(requestPath),
		run: func(c *ui.Context, _, name string) error {
			req, err := a.store.renameRequest(a.active, requestPath, name)
			if err != nil {
				return err
			}
			a.retargetTabs(requestPath, req.Path)
			c.Toast("已重命名为「" + req.Name + "」")
			return nil
		},
	})
	a.where.paths = []string{parentPath(requestPath)}
}

// askRenameFolder asks for the new name of a folder, which keeps its place in
// the tree.
func (a *app) askRenameFolder(folderPath string) {
	a.askFolder(folderQuestion{
		title:     "重命名文件夹",
		action:    "重命名",
		folder:    parentPath(folderPath),
		names:     true,
		nameTitle: "文件夹名称",
		name:      pathName(folderPath),
		run: func(c *ui.Context, _, name string) error {
			moved, err := a.store.renameFolder(a.active, folderPath, name)
			if err != nil {
				return err
			}
			a.retargetTabs(folderPath, moved.Path)
			c.Toast("已重命名为「" + moved.Name + "」")
			return nil
		},
	})
	a.where.paths = []string{parentPath(folderPath)}
}

// duplicateRequest copies an interface beside itself, as the menu's 复制接口
// does, and says what the copy is called.
func (a *app) duplicateRequest(c *ui.Context, requestPath string) {
	copied, err := a.store.duplicateRequest(a.active, requestPath)
	if err != nil {
		c.Toast("复制失败：" + err.Error())
		return
	}
	c.Toast("已复制为「" + copied.Name + "」")
}

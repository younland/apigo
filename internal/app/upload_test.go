package app

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"apigo/internal/store"
	"apigo/internal/widgets"

	"github.com/egoist/mygo/ui"
)

// writeTempFile writes a file of the given content in the test's directory and
// returns its path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestImportStaticKeepsFileByDayAndHash checks an uploaded file lands under
// static/<day>/<sha256>, that the same file twice is kept once, and that
// deleting it removes it.
func TestImportStaticKeepsFileByDayAndHash(t *testing.T) {
	s := store.New(t.TempDir())
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	col, err := s.NewCollection("Demo")
	if err != nil {
		t.Fatal(err)
	}
	src := writeTempFile(t, "Vue-Go冲锋Logo.png", "logo bytes")

	rel, err := s.ImportStatic(col.ID, src)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("logo bytes"))
	want := filepath.ToSlash(filepath.Join("static", time.Now().Format("2006-01-02"), hex.EncodeToString(sum[:])))
	if rel != want {
		t.Errorf("kept at %q, want %q", rel, want)
	}
	kept := s.StaticPath(col.ID, rel)
	if data, err := os.ReadFile(kept); err != nil || string(data) != "logo bytes" {
		t.Errorf("file %q: %v, %q; want the upload kept", kept, err, data)
	}

	again, err := s.ImportStatic(col.ID, src)
	if err != nil || again != rel {
		t.Errorf("second upload kept at %q (%v), want the same file %q", again, err, rel)
	}

	if err := s.RemoveStatic(col.ID, rel); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(kept); !os.IsNotExist(err) {
		t.Errorf("file %q: %v, want it deleted with the field", kept, err)
	}
}

// TestRemoveStaticKeepsWhatItDoesNotOwn checks a path outside the collection's
// static directory is left alone.
func TestRemoveStaticKeepsWhatItDoesNotOwn(t *testing.T) {
	s := store.New(t.TempDir())
	outside := writeTempFile(t, "apigo.yaml", "not a stale upload")
	if err := s.RemoveStatic("Demo", outside); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("file %q: %v, want a path outside static left alone", outside, err)
	}
}

// TestFormFileUploadsTheChosenFile checks a form-data field takes a file the
// desktop hands it, keeps it in the collection, and sends it under the name the
// file had.
func TestFormFileUploadsTheChosenFile(t *testing.T) {
	server, got := bodyServer(t)
	a := testApp(t)
	src := writeTempFile(t, "Vue-Go冲锋Logo.png", "logo bytes")
	a.chooseFile = func() (string, error) { return src, nil }
	a.current().url = server.URL
	a.current().method = "POST"
	a.current().bodyType = "form-data"
	a.current().reqTab = 2
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("选择文件第 1 行"); err != nil {
		t.Fatal(err)
	}
	row := a.current().form.Rows[0]
	if row.FileName != "Vue-Go冲锋Logo.png" || !strings.HasPrefix(row.File, "static/") {
		t.Fatalf("row %+v, want the file kept under static/ with the name it had", row)
	}
	if _, err := os.Stat(a.store.StaticPath(a.active, row.File)); err != nil {
		t.Fatalf("uploaded file: %v, want it kept in the collection", err)
	}

	// Name the field the way a user does, so the send carries it.
	if err := tt.Click("键第 1 行"); err != nil {
		t.Fatal(err)
	}
	tt.Type("logo")
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	sent := <-got
	if got := sent.files["logo"]; got != "Vue-Go冲锋Logo.png" {
		t.Errorf("file field name %q, want the name the file had; fields %v", got, sent.files)
	}
	if got := sent.fileBytes["logo"]; got != "logo bytes" {
		t.Errorf("file bytes %q, want the upload's content", got)
	}
}

// TestFormFileReplacedDeletesTheOldOne checks another file on the field deletes
// the one it held before.
func TestFormFileReplacedDeletesTheOldOne(t *testing.T) {
	a := testApp(t)
	first := writeTempFile(t, "first.txt", "first")
	second := writeTempFile(t, "second.txt", "second")
	chosen := first
	a.chooseFile = func() (string, error) { return chosen, nil }
	a.current().bodyType = "form-data"
	a.current().reqTab = 2
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("选择文件第 1 行"); err != nil {
		t.Fatal(err)
	}
	old := a.current().form.Rows[0].File
	chosen = second
	if err := tt.Click("选择文件第 1 行"); err != nil {
		t.Fatal(err)
	}
	if kept := a.current().form.Rows[0].File; kept == old {
		t.Fatalf("row still holds %q, want the second file", kept)
	}
	if _, err := os.Stat(a.store.StaticPath(a.active, old)); !os.IsNotExist(err) {
		t.Errorf("file %q: %v, want the replaced upload deleted", old, err)
	}
}

// TestFormFileDeletedWithItsRow checks deleting a form-data row deletes the
// file it held.
func TestFormFileDeletedWithItsRow(t *testing.T) {
	a := testApp(t)
	src := writeTempFile(t, "logo.txt", "logo")
	a.chooseFile = func() (string, error) { return src, nil }
	a.current().bodyType = "form-data"
	a.current().reqTab = 2
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("选择文件第 1 行"); err != nil {
		t.Fatal(err)
	}
	rel := a.current().form.Rows[0].File
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, err := os.Stat(a.store.StaticPath(a.active, rel)); !os.IsNotExist(err) {
		t.Errorf("file %q: %v, want it deleted with the row", rel, err)
	}
}

// TestFormFileRoundTrips checks an interface keeps the path and the name of an
// uploaded file, and brings them back when it is opened.
func TestFormFileRoundTrips(t *testing.T) {
	a := testApp(t)
	cur := a.current()
	cur.title = "上传"
	cur.bodyType = "form-data"
	cur.form.Rows = []widgets.Row{{
		Enabled: true, Key: "logo",
		File: "static/2025-10-20/abc123", FileName: "Vue-Go冲锋Logo.png",
	}}
	if err := a.saveCurrentToCollection(a.active); err != nil {
		t.Fatal(err)
	}

	reopened := store.New(a.store.Dir)
	if err := reopened.Load(); err != nil {
		t.Fatal(err)
	}
	col := reopened.Collection(a.active)
	if col == nil {
		t.Fatal("no collection after the reload")
	}
	req, ok := findRequest(col.Root, "上传")
	if !ok {
		t.Fatalf("no interface 上传 in %+v", col.Root)
	}
	if len(req.Form) != 1 {
		t.Fatalf("form %+v, want the uploaded field", req.Form)
	}
	if got := req.Form[0]; got.File != "static/2025-10-20/abc123" || got.FileName != "Vue-Go冲锋Logo.png" {
		t.Errorf("saved %+v, want the file's path and the name it had", got)
	}
}

// TestFormFileClearedDeletesIt checks clearing a field puts the row back to
// text and deletes the file it held.
func TestFormFileClearedDeletesIt(t *testing.T) {
	a := testApp(t)
	src := writeTempFile(t, "logo.txt", "logo")
	a.chooseFile = func() (string, error) { return src, nil }
	a.current().bodyType = "form-data"
	a.current().reqTab = 2
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("选择文件第 1 行"); err != nil {
		t.Fatal(err)
	}
	rel := a.current().form.Rows[0].File
	if err := tt.Click("清除文件第 1 行"); err != nil {
		t.Fatal(err)
	}
	if row := a.current().form.Rows[0]; row.File != "" || row.FileName != "" {
		t.Errorf("row %+v after clearing, want a text field again", row)
	}
	if _, err := os.Stat(a.store.StaticPath(a.active, rel)); !os.IsNotExist(err) {
		t.Errorf("file %q: %v, want it deleted with the clear", rel, err)
	}
}

// uploadTo keeps a file in a collection as an upload does, and returns the path
// the interface keeps it at.
func uploadTo(t *testing.T, a *app, collectionID, name string, data []byte) string {
	t.Helper()
	src := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}
	rel, err := a.store.ImportStatic(collectionID, src)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

// TestFormDataShowsAPictureOfAnImage checks a form field holding a picture shows
// it beside the file's name, and a field holding anything else keeps the name
// alone.
func TestFormDataShowsAPictureOfAnImage(t *testing.T) {
	a := testApp(t)
	picture := uploadTo(t, a, "Demo", "封面.png", shotPicture())
	report := uploadTo(t, a, "Demo", "报告.pdf", []byte("%PDF-1.7 a report"))

	t2 := a.current()
	t2.reqTab = 2
	t2.bodyType = "form-data"
	t2.form.Rows = []widgets.Row{
		{Enabled: true, Key: "封面", File: picture, FileName: "封面.png"},
		{Enabled: true, Key: "报告", File: report, FileName: "报告.pdf"},
		{Enabled: true},
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if _, ok := tt.Find("图片预览第 1 行"); !ok {
		t.Errorf("texts %q, want the picture previewed beside its name", tt.Texts())
	}
	// A file that is no picture keeps the name it came with, and no preview.
	if _, ok := tt.Find("图片预览第 2 行"); ok {
		t.Error("a preview is drawn for a file that is no picture")
	}
	if !tt.HasText("报告.pdf") || !tt.HasText("封面.png") {
		t.Errorf("texts %q, want both files named", tt.Texts())
	}

	// The preview stands between the file's name and the cross that clears it.
	name, _ := tt.Find("封面.png")
	thumb, _ := tt.Find("图片预览第 1 行")
	clear, _ := tt.Find("清除文件第 1 行")
	if name.X >= thumb.X || thumb.X >= clear.X {
		t.Errorf("name at %v, preview at %v, cross at %v; want the preview after the name",
			name, thumb, clear)
	}
	// The picture covers the square it draws in: one shrunk into it would leave
	// the row showing above and below.
	for _, y := range []int{int(thumb.Y) + 3, int(thumb.Y+thumb.H) - 4} {
		r, g, b, _ := tt.Image().RGBAAt(int(thumb.X+thumb.W/2), y).RGBA()
		if r>>8 > 240 && g>>8 > 240 && b>>8 > 240 {
			t.Errorf("the preview leaves the row showing at %d, want the picture covering it", y)
		}
	}
}

package app

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"apigo/internal/req"

	"github.com/egoist/mygo/ui"
)

// TestResponseBodyOf checks a response is shown by what it holds: code, a
// picture, a player, or a file to keep.
func TestResponseBodyOf(t *testing.T) {
	png := pngBytes(t)
	tests := []struct {
		name        string
		contentType string
		raw         []byte
		want        responseBody
	}{
		{"json", "application/json", []byte(`{"ok":true}`), bodyText},
		{"json without a type", "", []byte(`{"ok":true}`), bodyText},
		{"html", "text/html; charset=utf-8", []byte("<p>hi</p>"), bodyText},
		{"png", "image/png", png, bodyImage},
		{"png without a type", "", png, bodyImage},
		{"wav", "audio/wav", []byte("RIFF...."), bodyAudio},
		{"mp4", "video/mp4", []byte("\x00\x00\x00\x18ftypmp42"), bodyVideo},
		{"pdf", "application/pdf", []byte("%PDF-1.7"), bodyFile},
		{"zip", "application/zip", []byte("PK\x03\x04"), bodyFile},
	}
	for _, test := range tests {
		res := &req.Response{
			Headers: http.Header{"Content-Type": {test.contentType}},
			Body:    string(test.raw),
			Raw:     test.raw,
		}
		if got := responseBodyOf(res); got != test.want {
			t.Errorf("%s: responseBodyOf = %v, want %v", test.name, got, test.want)
		}
	}
}

// TestDownloadName checks the saved file is named as the server suggests, else
// as the address says, else by its type.
func TestDownloadName(t *testing.T) {
	tests := []struct {
		name        string
		disposition string
		contentType string
		url         string
		want        string
	}{
		{"suggested", `attachment; filename="report.pdf"`, "application/pdf", "https://x.test/get", "report.pdf"},
		{"encoded", `attachment; filename*=UTF-8''%E6%8A%A5%E5%91%8A.pdf`, "application/pdf", "https://x.test/get", "报告.pdf"},
		{"from the url", "", "application/pdf", "https://x.test/files/pets.csv?x=1", "pets.csv"},
		{"from the type", "", "application/pdf", "https://x.test/", "response.pdf"},
		{"no path of its own", `attachment; filename="../../etc/passwd"`, "", "https://x.test/", "passwd"},
	}
	for _, test := range tests {
		headers := http.Header{}
		if test.disposition != "" {
			headers.Set("Content-Disposition", test.disposition)
		}
		if test.contentType != "" {
			headers.Set("Content-Type", test.contentType)
		}
		res := &req.Response{Headers: headers, Raw: []byte("x")}
		if got := downloadName(res, test.url); got != test.want {
			t.Errorf("%s: downloadName = %q, want %q", test.name, got, test.want)
		}
	}
}

// TestResponseMetaHoldsTheDownloadIcon checks the download icon sits at the end
// of the meta bar, after the status and the time it took.
func TestResponseMetaHoldsTheDownloadIcon(t *testing.T) {
	server := serve(t, "application/json", "", []byte(`{"ok":true}`))
	a := sendTo(t, server.URL)
	tt := ui.NewTester(a.view, 1280, 800)

	status, ok := tt.Find("200 OK")
	if !ok {
		t.Fatalf("texts %q, want the status in the meta bar", tt.Texts())
	}
	icon, ok := tt.Find("下载响应")
	if !ok {
		t.Fatal("the meta bar has no download icon")
	}
	if icon.X < status.X+status.W {
		t.Errorf("download icon at x=%v, want it after the status at %v", icon.X, status.X)
	}
}

// TestResponsePictureIsShown checks an image response is drawn as a picture,
// with what it is and a way to keep it, rather than as text.
func TestResponsePictureIsShown(t *testing.T) {
	png := pngBytes(t)
	server := serve(t, "image/png", "", png)
	a := sendTo(t, server.URL)

	if got := a.current().view; got != bodyImage {
		t.Fatalf("view %v, want the response drawn as a picture", got)
	}
	tt := ui.NewTester(a.view, 1280, 800)
	if _, ok := tt.Find("响应图片"); !ok {
		t.Error("the pane shows no picture")
	}
	if want := "2 × 2 · image/png · " + formatSize(len(png)); !tt.HasText(want) {
		t.Errorf("texts %q, want %q under the picture", tt.Texts(), want)
	}
	if err := tt.Click("下载响应"); err != nil {
		t.Errorf("no way to keep the picture: %v", err)
	}
}

// TestResponsePlayerPlaysAndPauses checks a sound response gets a player that
// starts and stops, and offers the file as well.
func TestResponsePlayerPlaysAndPauses(t *testing.T) {
	server := serve(t, "audio/wav", "", wavBytes())
	a := sendTo(t, server.URL)
	t2 := a.current()
	if t2.view != bodyAudio || t2.audio == nil {
		t.Fatalf("view %v player %v, want the response drawn as a player", t2.view, t2.audio)
	}
	tt := ui.NewTester(a.view, 1280, 800)

	if err := tt.Click("播放"); err != nil {
		t.Fatalf("no play button: %v", err)
	}
	tt.Frame()
	if !t2.audio.Playing() {
		t.Error("the sound is not playing after 播放")
	}
	if err := tt.Click("暂停"); err != nil {
		t.Fatalf("no pause button while playing: %v", err)
	}
	tt.Frame()
	if t2.audio.Playing() {
		t.Error("the sound still plays after 暂停")
	}

	// A player asks for frames while it runs, and stops asking once it stops.
	if err := tt.Click("下载响应"); err != nil {
		t.Errorf("no way to keep the sound: %v", err)
	}
}

// TestResponseFileSavesToDownloads checks a body the pane cannot show is saved
// where the folder dialog says, numbered when the name is taken, and that
// cancelling writes nothing.
func TestResponseFileSavesToDownloads(t *testing.T) {
	body := []byte("%PDF-1.7 report")
	server := serve(t, "application/pdf", `attachment; filename="报告.pdf"`, body)
	a := sendTo(t, server.URL)
	tt := ui.NewTester(a.view, 1280, 800)

	dir := t.TempDir()
	a.pickDownloadDir = func() (string, error) { return dir, nil }
	if err := tt.Click("下载响应"); err != nil {
		t.Fatalf("no download button: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "报告.pdf"))
	if err != nil {
		t.Fatalf("saved file: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("saved %q, want the body as it came", got)
	}

	if err := tt.Click("下载响应"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "报告 (2).pdf")); err != nil {
		t.Errorf("second save: %v", err)
	}

	// A cancelled dialog leaves the downloads directory as it was.
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.pickDownloadDir = func() (string, error) { return "", nil }
	if err := tt.Click("下载响应"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Errorf("files %d after cancelling, want the %d that were there", len(after), len(before))
	}
}

// TestResponseVideoOffersSave checks a video is offered to keep, with a note
// that the pane does not play it.
func TestResponseVideoOffersSave(t *testing.T) {
	server := serve(t, "video/mp4", "", []byte("\x00\x00\x00\x18ftypmp42"))
	a := sendTo(t, server.URL)
	tt := ui.NewTester(a.view, 1280, 800)

	if got := a.current().view; got != bodyVideo {
		t.Fatalf("view %v, want the video kept as a file", got)
	}
	if !tt.HasText("视频暂不支持内嵌播放，保存后可用系统播放器打开") {
		t.Errorf("texts %q, want the note about the player", tt.Texts())
	}
	if err := tt.Click("下载响应"); err != nil {
		t.Errorf("no way to keep the video: %v", err)
	}
}

// serve starts a server that answers every request with a body.
func serve(t *testing.T, contentType, disposition string, body []byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		if disposition != "" {
			w.Header().Set("Content-Disposition", disposition)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server
}

// sendTo sends a request to the address and returns the app it landed in, with
// a directory of its own to save into.
func sendTo(t *testing.T, address string) *app {
	t.Helper()
	a := testApp(t)
	dir := t.TempDir()
	a.pickDownloadDir = func() (string, error) { return dir, nil }
	a.current().url = address
	tt := ui.NewTester(a.view, 1280, 800)
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if a.current().resp == nil {
		t.Fatal("no response recorded after sending")
	}
	return a
}

// pngBytes is a 2×2 picture, as a server would send one.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 0x2e, G: 0xa0, B: 0x6a, A: 0xff})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// wavBytes is a short sound: a header and a few frames of silence, which the
// player decodes as readily as a real one.
func wavBytes() []byte {
	const frames = 1920 // 40 ms at 48 kHz, mono
	var buf bytes.Buffer
	data := make([]byte, frames*2)
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+len(data)))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1) // PCM
	binary.LittleEndian.PutUint16(header[22:], 1) // mono
	binary.LittleEndian.PutUint32(header[24:], 48000)
	binary.LittleEndian.PutUint32(header[28:], 96000)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(len(data)))
	buf.Write(header)
	buf.Write(data)
	return buf.Bytes()
}

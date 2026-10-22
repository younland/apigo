package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	// The formats the response pane plays, registered as decoders.
	_ "github.com/egoist/mygo/audio/mp3"
	_ "github.com/egoist/mygo/audio/vorbis"
)

// resumeEvery is how often the pane asks for a frame while a sound plays, so
// the bar and the clock follow it.
const resumeEvery = 200 * time.Millisecond

// shown is what a response pane holds, worked out away from the window's
// thread: decoding a picture, opening a player or formatting a large body would
// stall a frame there.
type shown struct {
	res         *response
	view        responseBody
	picture     *ui.Bitmap
	pictureNote string
	audio       *audio.Player
	mediaErr    error
	text        string           // the body as the editor shows it
	lang        *editor.Language // the syntax it reads in, nil for plain text
}

// prepareResponse works out how a response is shown: the picture it holds, the
// player of a sound, or the text of a body in the syntax it reads in.
func prepareResponse(res *response) *shown {
	sh := &shown{res: res, view: bodyText}
	if res == nil || res.Err != nil {
		return sh
	}
	switch sh.view = responseBodyOf(res); sh.view {
	case bodyImage:
		picture, err := ui.DecodeBitmap(res.Raw)
		if err != nil {
			sh.view, sh.mediaErr = bodyFile, err
			return sh
		}
		sh.picture, sh.pictureNote = picture, pictureSize(picture)
	case bodyAudio:
		player, err := audio.OpenStream(bytes.NewReader(res.Raw))
		if err != nil {
			sh.view, sh.mediaErr = bodyFile, err
			return sh
		}
		sh.audio = player.SetVolume(0.8)
	case bodyText:
		// The syntax follows the content type, and a JSON body is indented as it
		// comes in: the server sends it over one line more often than not, and
		// the pane reads it as code.
		sh.lang = languageForResponse(responseContentType(res), res.Body)
		sh.text = res.Body
		if isJSON(responseContentType(res), res.Body) {
			sh.text = prettyResponse(res.Body)
		}
	}
	return sh
}

// dropResponse lets go of what the last response held: MyGo keeps a bitmap on
// the GPU as long as it is used, and a player holds its sound.
func (t *tab) dropResponse() {
	t.stopAudio()
	t.picture, t.pictureNote, t.mediaErr = nil, "", nil
	t.view = bodyText
}

// stopAudio stops the player of the tab, if it has one.
func (t *tab) stopAudio() {
	if t.audio == nil {
		return
	}
	t.audio.Stop()
	t.audio.Close()
	t.audio = nil
}

// pictureSize is how big a bitmap is, as the pane says it: the pixels a picture
// has rather than the room it takes.
func pictureSize(picture *ui.Bitmap) string {
	w, h := picture.Size()
	return fmt.Sprintf("%d × %d", w, h)
}

// rawBodyText is the body as the raw pane shows it: the bytes as text, or what
// a body the pane cannot read says of itself.
func rawBodyText(res *response) string {
	if responseBodyOf(res) != bodyText && len(res.Raw) > 0 {
		return fmt.Sprintf("（%s，%s）二进制内容，可在预览里查看或下载",
			mediaType(responseContentType(res)), formatSize(res.Size))
	}
	return res.Body
}

// mediaDetail says what a message body is, for the line under a picture or a
// player: the media type and the size.
func mediaDetail(res *response) string {
	kind := mediaType(responseContentType(res))
	if kind == "" {
		kind = "未知类型"
	}
	return kind + " · " + formatSize(res.Size)
}

// playText is how long a sound is, and where it is: "0:12 / 1:30".
func playText(t *tab) string {
	if t.audio == nil {
		return ""
	}
	return clockText(t.audio.Position()) + " / " + clockText(t.audio.Duration())
}

// clockText writes a length of sound as a player shows it.
func clockText(d time.Duration) string {
	if d <= 0 {
		return "0:00"
	}
	total := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

// playFraction is how much of the sound has played, from 0 to 1.
func playFraction(t *tab) float32 {
	if t.audio == nil {
		return 0
	}
	total := t.audio.Duration()
	if total <= 0 {
		return 0
	}
	played := float32(t.audio.Position()) / float32(total)
	return min(max(played, 0), 1)
}

// saveResponse keeps the body of a response where the reader says: the
// desktop's dialog opens at the downloads directory, and the file is numbered
// when the chosen directory has its name already.
func (a *app) saveResponse(c *ui.Context, t *tab) {
	if t.resp == nil || len(t.resp.Raw) == 0 {
		c.Toast("响应没有可下载的内容")
		return
	}
	if a.pickDownloadDir == nil {
		c.Toast("当前环境不能下载文件")
		return
	}
	dir, err := a.pickDownloadDir()
	if err != nil {
		c.Toast("选择目录失败：" + err.Error())
		return
	}
	if dir == "" {
		return // the reader cancelled: nothing to say
	}
	path, err := writeDownload(dir, downloadName(t.resp, t.url), t.resp.Raw)
	if err != nil {
		c.Toast("保存失败：" + err.Error())
		return
	}
	c.Toast("已保存到 " + path)
}

// writeDownload writes data into dir under name, numbered "name (2).ext" when
// that name is taken, and returns where it landed.
func writeDownload(dir, name string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := freeName(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// freeName is the first name of a file that is not taken: "name.ext", then
// "name (2).ext", and so on.
func freeName(dir, name string) string {
	path := filepath.Join(dir, name)
	if _, err := os.Lstat(path); err != nil {
		return path
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		path = filepath.Join(dir, base+" ("+strconv.Itoa(i)+")"+ext)
		if _, err := os.Lstat(path); err != nil {
			return path
		}
	}
}

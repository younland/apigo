package app

import (
	"image"
	"os"

	"github.com/egoist/mygo/ui"
)

// The box a form field's picture is shrunk into before it reaches the GPU: the
// square it draws in, at twice its size so it stays sharp on a dense screen. A
// photograph decoded whole would otherwise sit in the window's memory for a
// preview a few dozen pixels wide.
const thumbPixels = 56

// fileThumbnail is the picture an uploaded file holds, by the path the interface
// keeps it at, nil for a file that is no picture. It reads each file once: a
// form of many rows asks for the same thumbnail every frame.
func (a *app) fileThumbnail(scope string) func(string) *ui.Bitmap {
	return func(rel string) *ui.Bitmap {
		key := scope + "|" + rel
		if picture, read := a.thumbs[key]; read {
			return picture
		}
		// The files a collection keeps are content-addressed, so a thumbnail never
		// goes stale under the path it was made from.
		picture := readThumbnail(a.store.StaticPath(scope, rel))
		if a.thumbs == nil {
			a.thumbs = map[string]*ui.Bitmap{}
		}
		a.thumbs[key] = picture
		return picture
	}
}

// readThumbnail reads a file as a picture and shrinks it into the box a form
// field shows it in, nil when the file is not one.
func readThumbnail(path string) *ui.Bitmap {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	picture, _, err := image.Decode(file)
	if err != nil {
		return nil
	}
	return ui.NewBitmap(shrunk(picture, thumbPixels, thumbPixels))
}

// shrunk scales a picture down to fit a box, keeping its proportions, by taking
// the nearest pixel of the source: a thumbnail of a few dozen pixels reads the
// same as one filtered with care, and needs nothing beyond the standard library.
// A picture already inside the box is kept as it is.
func shrunk(picture image.Image, boxW, boxH int) image.Image {
	bounds := picture.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= boxW && h <= boxH {
		return picture
	}
	scale := min(float64(boxW)/float64(w), float64(boxH)/float64(h))
	outW, outH := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	out := image.NewRGBA(image.Rect(0, 0, outW, outH))
	for y := range outH {
		for x := range outW {
			out.Set(x, y, picture.At(
				bounds.Min.X+x*w/outW, bounds.Min.Y+y*h/outH))
		}
	}
	return out
}

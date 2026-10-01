// Package ui is what the device draws, as distinct from the panel it draws on.
package ui

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"sync"

	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// The mark, one for each kind of background. Two drawings rather than one and a rule: the dark one
// is not the light one inverted, the wordmark changes colour. Scaled from assets by `make logos`.
//
//go:embed logo_light.png
var lightPNG []byte

//go:embed logo_dark.png
var darkPNG []byte

var (
	lightOnce sync.Once
	light     image.Image

	darkOnce sync.Once
	dark     image.Image
)

// Logo is the mark for a light background, decoded once.
func Logo() image.Image { return decode(&lightOnce, &light, lightPNG) }

// Night is the mark for a dark background.
func Night() image.Image { return decode(&darkOnce, &dark, darkPNG) }

// Mark is whichever of the two reads on this background.
func Mark(onDark bool) image.Image {
	if onDark {
		return Night()
	}
	return Logo()
}

// MarkWidth is how wide the mark comes out at a height.
func MarkWidth(h int) int {
	img := Logo()
	if img == nil || img.Bounds().Dy() == 0 {
		return h
	}
	b := img.Bounds()
	return b.Dx() * h / b.Dy()
}

// DrawLogo paints the mark to fit inside a box, keeping its proportions and centred in whatever
// room is left over. on is the background it is composited against.
func DrawLogo(s Surface, r Rect, on theme.Color) {
	img := Mark(theme.Dark(on))
	if img == nil {
		return
	}

	b := img.Bounds()
	w, h := r.W, r.H
	if b.Dx()*r.H > b.Dy()*r.W {
		h = b.Dy() * r.W / b.Dx()
	} else {
		w = b.Dx() * r.H / b.Dy()
	}

	at := Rect{X: r.X + (r.W-w)/2, Y: r.Y + (r.H-h)/2, W: w, H: h}
	DrawImageScaled(s, img, at, on)
}

func decode(once *sync.Once, into *image.Image, raw []byte) image.Image {
	once.Do(func() {
		if img, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
			*into = img
		}
	})
	return *into
}

// Package ui is what the device draws, as distinct from the panel it draws on.
package ui

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"sync"
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

func decode(once *sync.Once, into *image.Image, raw []byte) image.Image {
	once.Do(func() {
		if img, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
			*into = img
		}
	})
	return *into
}

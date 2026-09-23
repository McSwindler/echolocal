package ui

import (
	"image"
	"image/color"
	"log/slog"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// One weight. A boot list and a clock are read across a room, and what makes them legible is size,
// not a second cut of the same letters — which is another 880 KB.
var (
	faceOnce sync.Once
	parsed   *opentype.Font

	facesMu sync.Mutex
	faces   = map[float64]font.Face{}
)

// Face is the lettering at a size in pixels, built once per size.
func Face(px float64) font.Face {
	faceOnce.Do(func() {
		f, err := opentype.Parse(goregular.TTF)
		if err != nil {
			slog.Error("the lettering would not parse", "err", err)
			return
		}
		parsed = f
	})
	if parsed == nil {
		return nil
	}

	facesMu.Lock()
	defer facesMu.Unlock()

	if f, ok := faces[px]; ok {
		return f
	}

	// DPI 72 makes Size a height in pixels rather than points.
	f, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		slog.Error("the lettering would not size", "px", px, "err", err)
		return nil
	}
	faces[px] = f
	return f
}

// Measure is how wide a line comes out, and how far its baseline sits below the top.
func Measure(s string, px float64) (w, baseline int) {
	f := Face(px)
	if f == nil {
		return 0, 0
	}
	m := f.Metrics()
	return font.MeasureString(f, s).Ceil(), m.Ascent.Ceil()
}

// Metrics is how far a line reaches above and below its baseline.
func Metrics(px float64) (ascent, descent int) {
	f := Face(px)
	if f == nil {
		return 0, 0
	}
	m := f.Metrics()
	return m.Ascent.Ceil(), m.Descent.Ceil()
}

// Line draws text onto an image with its baseline at at.
func Line(dst *image.RGBA, s string, px float64, at image.Point, c color.Color) {
	f := Face(px)
	if f == nil {
		return
	}

	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: f,
		Dot:  fixed.P(at.X, at.Y),
	}
	d.DrawString(s)
}

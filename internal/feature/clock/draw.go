package clock

import (
	"github.com/ygelfand/echolocal/internal/hardware/screen"
	"github.com/ygelfand/echolocal/internal/ui"
	uitheme "github.com/ygelfand/echolocal/internal/ui/theme"
)

// How much of the panel the time takes. Bounded by both, since height alone runs a tall panel's
// time off the sides.
const (
	timeHeightShare = 0.34
	timeWidthShare  = 0.86
)

// The rest is sized against the time rather than the panel, so the face keeps its proportions.
const (
	suffixOfTime = 0.24
	dateOfTime   = 0.20
	gapOfTime    = 0.22
	suffixGap    = 0.35
)

// drawFace paints the time with the date under it, the pair centred as one block.
func drawFace(p *screen.Panel, t uitheme.Theme, r Reading) {
	p.Fill(screen.Opaque(t.Background.R, t.Background.G, t.Background.B))

	s := ui.Of(p)
	w, h := s.Size()

	size := fit(ui.Bold, r.Time+r.Suffix, int(float64(w)*timeWidthShare), int(float64(h)*timeHeightShare))
	time := ui.MustLoad(ui.Bold, size)
	suffix := ui.MustLoad(ui.Medium, int(float64(size)*suffixOfTime))
	date := ui.MustLoad(ui.Regular, int(float64(size)*dateOfTime))

	timeW, timeH := time.Measure(r.Time)
	dateW, dateH := date.Measure(r.Date)

	suffixW := 0
	if r.Suffix != "" {
		suffixW, _ = suffix.Measure(r.Suffix)
		suffixW += int(float64(size) * suffixOfTime * suffixGap)
	}

	between := int(float64(timeH) * gapOfTime)
	top := (h - (timeH + between + dateH)) / 2
	timeX := (w - (timeW + suffixW)) / 2

	ui.DrawText(s, time, timeX, top, t.Text, t.Background, r.Time)
	if r.Suffix != "" {
		// Sat on the time's baseline rather than its top, which is where an eye expects it.
		ui.DrawText(s, suffix,
			timeX+timeW+int(float64(size)*suffixOfTime*suffixGap),
			top+time.Ascent()-suffix.Ascent(), t.Muted, t.Background, r.Suffix)
	}
	ui.DrawText(s, date, (w-dateW)/2, top+timeH+between, t.Muted, t.Background, r.Date)
}

// fit is the largest size at which the text is no wider than maxW, starting from maxH.
func fit(weight ui.Weight, text string, maxW, maxH int) int {
	size := max(maxH, 1)

	for range 8 {
		w, _ := ui.MustLoad(weight, size).Measure(text)
		if w <= maxW || size <= 1 {
			return size
		}

		next := size * maxW / w
		if next >= size {
			next = size - 1
		}
		size = max(next, 1)
	}
	return size
}

package privacy

import (
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// The wedge against the shorter side, and the glyph within the wedge.
const (
	markShare  = 1.0 / 11.0
	glyphShare = 0.30
)

// Fixed, not the palette.
var (
	red   = theme.Color{R: 0xe0, G: 0x3b, B: 0x2f}
	white = theme.Color{R: 0xff, G: 0xff, B: 0xff}
)

// Marks is what the physical controls are doing.
type Marks struct {
	MicMuted      bool
	CameraCovered bool
}

// Showing reports whether there is anything to draw.
func (m Marks) Showing() bool { return m.MicMuted || m.CameraCovered }

// Draw puts a wedge in a top corner for each control that is cutting something off.
func Draw(s ui.Surface, m Marks) {
	w, h := s.Size()

	size := int(float64(min(w, h)) * markShare)
	glyph := int(float64(size) * glyphShare)
	if size < 1 || glyph < 1 {
		return
	}
	at := size/3 - glyph/2

	right := w - 1

	if m.MicMuted {
		ui.FillPolygon(s, []ui.Point{{X: 0, Y: 0}, {X: size, Y: 0}, {X: 0, Y: size}}, red)
		ui.DrawIcon(s, icons.AVMicOff, ui.Rect{X: at, Y: at, W: glyph, H: glyph}, white, red)
	}
	if m.CameraCovered {
		ui.FillPolygon(s, []ui.Point{
			{X: right, Y: 0}, {X: right - size, Y: 0}, {X: right, Y: size},
		}, red)
		ui.DrawIcon(s, icons.AVVideocamOff,
			ui.Rect{X: right - at - glyph, Y: at, W: glyph, H: glyph}, white, red)
	}
}

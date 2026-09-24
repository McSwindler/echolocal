package drawer

import (
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// The rail, as fractions of the shorter side of the picture.
const (
	// thickShare is how far in from its edge the rail reaches. Wide enough for a mark and its
	// name, narrow enough that the clock beside it is still a clock — the longest name is well
	// inside this, and widening it for them crowds the hour.
	thickShare = 0.135

	// marginShare is the gap between the rail and the edge it hangs off, so it reads as something
	// laid over the screen rather than part of the bezel.
	marginShare = 0.025

	iconShare   = 0.058
	labelShare  = 0.019
	radiusShare = 0.05

	// pressedShare is how far a cell shifts towards the text color under a finger.
	pressedShare = 0.12
)

// Strip is where the rail sits for an edge and a screen size.
//
// It hangs off the edge of the picture rather than of the panel, so it follows the device being
// turned without anything here knowing that happened.
func Strip(edge config.Edge, w, h int) ui.Rect {
	unit := min(w, h)
	thick := int(float64(unit) * thickShare)
	margin := int(float64(unit) * marginShare)

	switch edge {
	case config.EdgeLeft:
		return ui.Rect{X: margin, Y: margin, W: thick, H: h - margin*2}
	case config.EdgeTop:
		return ui.Rect{X: margin, Y: margin, W: w - margin*2, H: thick}
	case config.EdgeBottom:
		return ui.Rect{X: margin, Y: h - thick - margin, W: w - margin*2, H: thick}
	}
	return ui.Rect{X: w - thick - margin, Y: margin, W: thick, H: h - margin*2}
}

// Cells is where each icon sits, spread along the rail and centered as a block, so a rail of three
// does not read as a rail of eight with five missing.
func Cells(strip ui.Rect, n int) []ui.Rect {
	if n <= 0 {
		return nil
	}

	along, across, down := strip.H, strip.W, true
	if strip.W > strip.H {
		along, across, down = strip.W, strip.H, false
	}

	cell := min(across, along/n)
	start := (along - cell*n) / 2

	out := make([]ui.Rect, 0, n)
	for i := range n {
		at := start + i*cell
		if down {
			out = append(out, ui.Rect{X: strip.X, Y: strip.Y + at, W: across, H: cell})
		} else {
			out = append(out, ui.Rect{X: strip.X + at, Y: strip.Y, W: cell, H: across})
		}
	}
	return out
}

// Draw paints the rail over whatever is underneath it.
//
// A mark with its name under it, so the rail reads as a set of places to go rather than a column
// of settings to nudge. Every one of them opens a screen, including the two whose mark also says
// what they are currently at.
func Draw(s ui.Surface, strip ui.Rect, entries []Entry, palette theme.Theme, pressed func(int) bool) {
	w, h := s.Size()
	unit := min(w, h)

	ui.FillRounded(s, strip, int(float64(unit)*radiusShare), palette.Surface)

	mark := int(float64(unit) * iconShare)
	font := ui.MustLoad(ui.Medium, int(float64(unit)*labelShare))

	for i, at := range Cells(strip, len(entries)) {
		e := entries[i]
		if e.Icon == nil {
			continue
		}

		on := palette.Surface
		if pressed != nil && pressed(i) {
			on = palette.Surface.Blend(palette.Text, pressedShare)
			ui.FillRounded(s, at.Inset(at.W/12), at.W/5, on)
		}

		name := e.Label()

		_, high := font.Measure(name)
		gap := mark / 5
		if name == "" {
			high, gap = 0, 0
		}

		top := at.Y + (at.H-(mark+gap+high))/2
		cx, _ := at.Center()

		ui.DrawIcon(s, e.Icon(),
			ui.Rect{X: cx - mark/2, Y: top, W: mark, H: mark}, palette.Text, on)

		if name != "" {
			ui.DrawTextIn(s, font,
				ui.Rect{X: at.X, Y: top + mark + gap, W: at.W, H: high},
				palette.Muted, on, name)
		}
	}
}

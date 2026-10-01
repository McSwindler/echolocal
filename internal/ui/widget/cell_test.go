package widget

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
)

// The panel, so the layout is exercised at the size it will run at.
func panel() Metrics { return New(1200, 1920) }

// Tiles fill the width they are given: a grid that left a margin on the right would look like a
// column was missing.
func TestAGridFillsTheWidth(t *testing.T) {
	m := panel()
	body := ui.Rect{X: 60, Y: 300, W: 1080, H: 1500}

	for _, across := range []int{2, 3, 4} {
		got := m.Cells(body, across, across)
		if len(got) != across {
			t.Fatalf("%d across: %d tiles", across, len(got))
		}

		first, last := got[0], got[across-1]
		if first.X != body.X {
			t.Errorf("%d across: starts at %d, want %d", across, first.X, body.X)
		}

		// Within a pixel or two: the width is shared out in whole pixels and the remainder has
		// nowhere to go.
		if off := body.X + body.W - (last.X + last.W); off < 0 || off > across {
			t.Errorf("%d across: ends %d short of the right edge", across, off)
		}
	}
}

// A row of tiles is a row: they line up, and the next line is below them.
func TestTilesRunAcrossThenDown(t *testing.T) {
	m := panel()
	got := m.Cells(ui.Rect{W: 1200, H: 1600}, 7, 3)

	if len(got) != 7 {
		t.Fatalf("%d tiles", len(got))
	}

	for i := 1; i < 3; i++ {
		if got[i].Y != got[0].Y {
			t.Errorf("tile %d sits at %d, not on the first line at %d", i, got[i].Y, got[0].Y)
		}
		if got[i].X <= got[i-1].X {
			t.Errorf("tile %d is at %d, not right of %d", i, got[i].X, got[i-1].X)
		}
	}

	if got[3].Y <= got[0].Y {
		t.Errorf("the fourth tile is at %d, not below the first line at %d", got[3].Y, got[0].Y)
	}
	if got[3].X != got[0].X {
		t.Errorf("the fourth tile starts at %d, not under the first at %d", got[3].X, got[0].X)
	}
}

// The space between lines is the space between columns. Tiles butted up against each other read as
// one block rather than as a grid of separate things, and nothing about them overlapping would say
// so — they would simply touch.
func TestTheGapIsTheSameBothWays(t *testing.T) {
	m := panel()
	got := m.Cells(ui.Rect{W: 1200, H: 1600}, 6, 3)

	across := got[1].X - (got[0].X + got[0].W)
	down := got[3].Y - (got[0].Y + got[0].H)

	if across <= 0 {
		t.Errorf("no space between columns")
	}
	if down != across {
		t.Errorf("%d between lines and %d between columns", down, across)
	}
}

// Tiles do not overlap. They are drawn in order, so an overlap would be the last one quietly
// covering the edge of the one before it rather than anything obvious.
func TestTilesDoNotOverlap(t *testing.T) {
	m := panel()
	got := m.Cells(ui.Rect{W: 1200, H: 1600}, 9, 3)

	for i := range got {
		for j := i + 1; j < len(got); j++ {
			if got[i].Overlaps(got[j]) {
				t.Errorf("tile %d at %+v overlaps tile %d at %+v", i, got[i], j, got[j])
			}
		}
	}
}

// Twelve themes have to fit without scrolling — that is the whole reason for a grid rather than a
// row each.
func TestTwelveTilesFitThePanel(t *testing.T) {
	m := panel()
	body := m.Body(ui.Rect{W: 1200, H: 1920})

	got := m.Cells(body, 12, 3)
	last := got[len(got)-1]

	if bottom := last.Y + last.H; bottom > body.Y+body.H {
		t.Errorf("twelve tiles reach %d, past the %d the page has", bottom, body.Y+body.H)
	}
}

func TestAShapedGridTakesItsShape(t *testing.T) {
	m := panel()
	body := ui.Rect{W: 1200, H: 1600}

	for _, shape := range []float64{0.625, 1.6} {
		got := m.CellsShaped(body, 4, 4, shape)
		if len(got) != 4 {
			t.Fatalf("shape %v: %d tiles, want 4", shape, len(got))
		}
		if tall := float64(got[0].H) / float64(got[0].W); tall < shape-0.02 || tall > shape+0.02 {
			t.Errorf("shape %v: tile %dx%d", shape, got[0].W, got[0].H)
		}
	}
	if got, want := m.CellsShaped(body, 1, 4, 0)[0], m.Cells(body, 1, 4)[0]; got != want {
		t.Errorf("unshaped tile %v, want %v", got, want)
	}
}

func TestAGridOfNothingIsNothing(t *testing.T) {
	m := panel()

	if got := m.Cells(ui.Rect{W: 1200, H: 1600}, 0, 3); got != nil {
		t.Errorf("no tiles laid out as %v", got)
	}
	if got := m.Cells(ui.Rect{W: 1200, H: 1600}, 4, 0); got != nil {
		t.Errorf("nothing across laid out as %v", got)
	}
}

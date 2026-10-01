package shell

import (
	"fmt"
	"testing"

	"github.com/ygelfand/echolocal/internal/hardware/touch"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

func longPage(n int, kind widget.Kind, tapped *int, levels *[]int) *Page {
	p := &Page{
		Title: "Long",
		Build: func() ([]widget.Row, []func(int)) {
			rows := make([]widget.Row, n)
			acts := make([]func(int), n)
			for i := range rows {
				rows[i] = widget.Row{Label: fmt.Sprintf("Row %d", i), Kind: kind, Level: 50}
				acts[i] = func(level int) {
					if tapped != nil {
						*tapped = i
					}
					if levels != nil {
						*levels = append(*levels, level)
					}
				}
			}
			return rows, acts
		},
	}
	p.Draw(ui.NewImage(1200, 1920, theme.Default().Background), theme.Default())
	return p
}

func TestAPageScrollsWithinItsOverflow(t *testing.T) {
	p := longPage(30, widget.Chevron, nil, nil)
	if p.overflow <= 0 {
		t.Fatal("thirty rows fit")
	}
	if p.Scroll(-10) {
		t.Error("scrolled above the top")
	}
	if !p.Scroll(1 << 20) {
		t.Fatal("did not scroll down")
	}
	if p.scroll != p.overflow {
		t.Errorf("scrolled to %d, want %d", p.scroll, p.overflow)
	}
	if p.Scroll(10) {
		t.Error("scrolled past the bottom")
	}
}

func TestATapAfterScrollingHitsTheRowUnderIt(t *testing.T) {
	tapped := -1
	p := longPage(30, widget.Chevron, &tapped, nil)
	p.Scroll(1 << 20)
	p.Draw(ui.NewImage(1200, 1920, theme.Default().Background), theme.Default())

	last := p.visible[len(p.visible)-1]
	p.Tap(last.X+last.W/2, last.Y+last.H/2)
	if tapped != 29 {
		t.Errorf("tapped row %d, want 29", tapped)
	}

	tapped = -1
	if _, ok := p.hit(600, p.body.Y-5); ok {
		t.Error("a row scrolled under the header was hit")
	}
}

func TestTheHeaderIsStillTheHeaderWhenScrolled(t *testing.T) {
	p := longPage(30, widget.Chevron, nil, nil)
	p.Scroll(1 << 20)
	p.Draw(ui.NewImage(1200, 1920, theme.Default().Background), theme.Default())

	if p.places[0].Y >= 0 {
		t.Fatal("the first row did not scroll off the top")
	}
	if !p.header(p.body.Y / 2) {
		t.Error("a tap on the header of a scrolled page is not a header tap")
	}
	if p.header(p.body.Y + 5) {
		t.Error("a tap on the rows reads as the header")
	}
}

func TestAnUpwardStrokeScrollsThePage(t *testing.T) {
	s := &Shell{}
	tapped := -1
	p := longPage(30, widget.Chevron, &tapped, nil)
	s.Push(p)

	stroke(s, p, 600, 1500, 604, 900)
	if p.scroll != 600 {
		t.Errorf("scrolled %d, want the 600 the finger moved", p.scroll)
	}
	if tapped != -1 || !s.Open() {
		t.Error("the stroke tapped or closed")
	}
}

func TestAVerticalStrokeOnASliderScrolls(t *testing.T) {
	s := &Shell{}
	var levels []int
	p := longPage(30, widget.Slider, nil, &levels)
	s.Push(p)
	row := p.visible[3]

	s.down(p, touch.Contact{X: row.X + row.W/2, Y: row.Y + row.H/2})
	s.move(p, touch.Contact{X: row.X + row.W/2 + 4, Y: row.Y + row.H/2 - 300})
	s.up(p, touch.Contact{X: row.X + row.W/2 + 4, Y: row.Y + row.H/2 - 300})

	if p.scroll != 300 {
		t.Errorf("scrolled %d, want 300", p.scroll)
	}
	if len(levels) != 0 {
		t.Errorf("the slider moved: %v", levels)
	}
}

func TestASidewaysStrokeOnASliderStillDragsIt(t *testing.T) {
	s := &Shell{}
	var levels []int
	p := longPage(30, widget.Slider, nil, &levels)
	s.Push(p)
	row := p.visible[3]

	stroke(s, p, row.X+row.W/2, row.Y+row.H/2, row.X+row.W-20, row.Y+row.H/2+5)
	if p.scroll != 0 {
		t.Errorf("scrolled %d", p.scroll)
	}
	if len(levels) == 0 {
		t.Error("the slider did not move")
	}
}

func TestASidewaysSwipeStillGoesBackOnALongPage(t *testing.T) {
	s := &Shell{}
	p := longPage(30, widget.Chevron, nil, nil)
	s.Push(p)

	stroke(s, p, rightEdge(), 800, rightEdge()-600, 810)
	if s.Open() {
		t.Error("a sideways swipe from the back edge left the page up")
	}
}

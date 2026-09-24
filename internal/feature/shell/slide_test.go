package shell

import (
	"testing"
	"time"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// mark is a view that paints one pixel at its own origin, so where it was drawn can be found by
// looking for it.
type mark struct {
	color  theme.Color
	covers bool
}

func (m mark) Draw(s ui.Surface, _ theme.Theme) { s.Set(0, 0, m.color) }
func (mark) Tap(int, int) bool                  { return true }
func (m mark) Covers() bool                     { return m.covers }

// found is where the view's pixel landed along the top row, or -1.
func found(img *ui.Image, c theme.Color) int {
	w, _ := img.Size()
	for x := range w {
		if img.At(x, 0) == c {
			return x
		}
	}
	return -1
}

func TestASlideStartsWholeAndEndsArrived(t *testing.T) {
	m := &slide{began: time.Now()}
	if got := m.traveled(); got > 0.2 {
		t.Errorf("a slide that has just begun is %.2f along", got)
	}

	m.began = time.Now().Add(-2 * slideFor)
	if got := m.traveled(); got != 1 {
		t.Errorf("a slide past its time is %.2f along, want 1", got)
	}
	if !m.done() {
		t.Error("a slide past its time does not report itself done")
	}
}

// The two pages are a screen apart the whole way across, so there is never a gap between them and
// never a strip of one over the other.
func TestTheTwoPagesStayAScreenApart(t *testing.T) {
	const w, h = 60, 10

	from := mark{color: theme.Color{R: 255}, covers: true}
	to := mark{color: theme.Color{B: 255}, covers: true}

	for _, back := range []bool{false, true} {
		for _, along := range []time.Duration{0, slideFor / 3, slideFor / 2, slideFor} {
			m := &slide{from: from, back: back, began: time.Now().Add(-along)}

			img := ui.NewImage(w, h, theme.Color{})
			m.draw(to, img, theme.Default())

			leaving, arriving := found(img, from.color), found(img, to.color)

			// One of the pair is off the surface at each end of the slide, and one is always on.
			if leaving < 0 && arriving < 0 {
				t.Errorf("back=%v at %v: neither page was drawn", back, along)
				continue
			}
			if leaving >= 0 && arriving >= 0 && abs(arriving-leaving) != w {
				t.Errorf("back=%v at %v: the pages are %d apart, want %d",
					back, along, abs(arriving-leaving), w)
			}
		}
	}
}

// Pushing brings the new page in from the right; going back sends the old one that way.
func TestTheSlideGoesTheWayItIsAsked(t *testing.T) {
	const w, h = 60, 10

	from := mark{color: theme.Color{R: 255}, covers: true}
	to := mark{color: theme.Color{B: 255}, covers: true}

	forward := &slide{from: from, began: time.Now().Add(-slideFor / 2)}
	img := ui.NewImage(w, h, theme.Color{})
	forward.draw(to, img, theme.Default())

	if at := found(img, to.color); at <= 0 {
		t.Errorf("the page being pushed is at %d, want somewhere to the right", at)
	}

	back := &slide{from: from, back: true, began: time.Now().Add(-slideFor / 2)}
	img = ui.NewImage(w, h, theme.Color{})
	back.draw(to, img, theme.Default())

	if at := found(img, from.color); at <= 0 {
		t.Errorf("the page being left is at %d, want somewhere to the right", at)
	}
}

func TestNothingSlidesOntoAnEmptyStack(t *testing.T) {
	if m := begin(nil, mark{covers: true}, false); m != nil {
		t.Error("the first page opened with a slide, with nothing to slide from")
	}
}

// An overlay has the dashboard showing through it, so sliding one would drag a hole across the
// picture rather than moving a page.
func TestAnOverlayDoesNotSlide(t *testing.T) {
	covering := mark{covers: true}
	over := mark{covers: false}

	if m := begin(covering, over, false); m != nil {
		t.Error("an overlay slid in over a page")
	}
	if m := begin(over, covering, false); m != nil {
		t.Error("a page slid in over an overlay")
	}
	if m := begin(covering, mark{color: theme.Color{R: 1}, covers: true}, false); m == nil {
		t.Error("two pages that both cover did not slide")
	}
}

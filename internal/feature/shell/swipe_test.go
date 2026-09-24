package shell

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/hardware/touch"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// counted is a view that says whether it was tapped, and where.
type counted struct {
	taps   int
	atX    int
	atY    int
	refuse bool
}

func (c *counted) Draw(ui.Surface, theme.Theme) {}
func (c *counted) Covers() bool                 { return true }

func (c *counted) Tap(x, y int) bool {
	c.taps++
	c.atX, c.atY = x, y
	return !c.refuse
}

// stroke puts a finger down, moves it there, and lifts.
func stroke(s *Shell, v View, fromX, fromY, toX, toY int) {
	s.down(v, touch.Contact{X: fromX, Y: fromY})
	s.move(v, touch.Contact{X: toX, Y: toY})
	s.up(v, touch.Contact{X: toX, Y: toY})
}

// A page is put away sideways. Dragging up or down it is not asking for that, and taking the screen
// away mid-stroke is the fault this holds.
func TestAVerticalSwipeDoesNotCloseThePage(t *testing.T) {
	for _, to := range []int{100, 900} {
		s := &Shell{}
		page := &counted{}
		s.Push(page)

		stroke(s, page, 600, 500, 604, to)

		if !s.Open() {
			t.Errorf("a swipe to y=%d closed the page", to)
		}
		if page.taps != 0 {
			t.Errorf("a swipe to y=%d tapped the page %d times", to, page.taps)
		}
	}
}

// Sideways still closes, either way: the gesture works from whichever hand is nearer.
func TestASidewaysSwipeClosesThePage(t *testing.T) {
	for _, to := range []int{100, 1100} {
		s := &Shell{}
		page := &counted{}
		s.Push(page)

		stroke(s, page, 600, 500, to, 505)

		if s.Open() {
			t.Errorf("a swipe to x=%d left the page up", to)
		}
	}
}

// A diagonal that is mostly down is not a sideways swipe, however far it went.
func TestADiagonalIsJudgedByItsLongerSide(t *testing.T) {
	s := &Shell{}
	page := &counted{}
	s.Push(page)

	// Past the sweep threshold on both axes, further down than across.
	stroke(s, page, 400, 300, 600, 700)

	if !s.Open() {
		t.Error("a mostly vertical diagonal closed the page")
	}
}

// A finger that went down on one row and lifted over another pressed neither. This is the case
// between the two thresholds: too far to be a tap, not sideways enough to be a swipe.
func TestDriftingOffARowTapsNothing(t *testing.T) {
	s := &Shell{}
	page := &counted{}
	s.Push(page)

	stroke(s, page, 600, 500, 620, 560)

	if page.taps != 0 {
		t.Errorf("a drift of 60px tapped the page %d times", page.taps)
	}
	if !s.Open() {
		t.Error("a drift of 60px closed the page")
	}
}

// A tap still taps, wobble and all. A fingertip is eighty pixels across and its center moves as the
// pressure changes, so this is the ordinary case rather than a careless one.
func TestATapWithAWobbleStillTaps(t *testing.T) {
	s := &Shell{}
	page := &counted{}
	s.Push(page)

	stroke(s, page, 600, 500, 612, 508)

	if page.taps != 1 {
		t.Fatalf("the page was tapped %d times, want once", page.taps)
	}
	if page.atX != 612 || page.atY != 508 {
		t.Errorf("the tap landed at %d,%d, want where the finger lifted", page.atX, page.atY)
	}
	if !s.Open() {
		t.Error("a tap closed the page")
	}
}

// A touch the view does not want is the rail being tapped beside rather than on, which closes.
func TestATapTheViewRefusesCloses(t *testing.T) {
	s := &Shell{}
	page := &counted{refuse: true}
	s.Push(page)

	stroke(s, page, 600, 500, 600, 500)

	if s.Open() {
		t.Error("a touch the view refused left it up")
	}
}

package shell

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
)

func spots() []ui.Rect {
	return []ui.Rect{
		{X: 0, Y: 0, W: 100, H: 100},
		{X: 200, Y: 0, W: 100, H: 100},
	}
}

// The zero value is a view nobody has touched, which every view starts as.
func TestNothingIsPressedToBeginWith(t *testing.T) {
	var touches Touches
	touches.Places(spots())

	for i := range spots() {
		if touches.Pressed(i) {
			t.Errorf("place %d reads as pressed before anything was touched", i)
		}
	}
	if touches.Damaged() != (ui.Rect{}) {
		t.Error("an untouched view reports damage")
	}
}

func TestPressingMarksThePlaceUnderTheFinger(t *testing.T) {
	var touches Touches
	touches.Places(spots())

	if !touches.Press(250, 50, true) {
		t.Fatal("pressing a place reports nothing changed")
	}
	if touches.Pressed(0) {
		t.Error("the place the finger missed reads as pressed")
	}
	if !touches.Pressed(1) {
		t.Error("the place the finger landed on does not read as pressed")
	}
	if got, want := touches.Damaged(), spots()[1]; got != want {
		t.Errorf("damage is %+v, want the place pressed %+v", got, want)
	}
}

// Lifting has to repaint what was pressed, so the mark stays until something else is touched.
func TestLiftingClearsThePressAndKeepsTheDamage(t *testing.T) {
	var touches Touches
	touches.Places(spots())
	touches.Press(50, 50, true)

	if !touches.Press(0, 0, false) {
		t.Fatal("lifting reports nothing changed")
	}
	if touches.Pressed(0) {
		t.Error("a place still reads as pressed after the finger lifted")
	}
	if got, want := touches.Damaged(), spots()[0]; got != want {
		t.Errorf("damage after lifting is %+v, want the place that was pressed %+v", got, want)
	}
}

// A finger going down on nothing is most of the screen, and must not be reported as a change.
func TestPressingNothingChangesNothing(t *testing.T) {
	var touches Touches
	touches.Places(spots())

	if touches.Press(150, 50, true) {
		t.Error("pressing between the places reports a change")
	}
}

// A pull outranks a press: the slider being moved is what changed.
func TestDraggingOutranksAPress(t *testing.T) {
	var touches Touches
	touches.Places(spots())
	touches.Press(50, 50, true)

	track := ui.Rect{X: 0, Y: 400, W: 300, H: 40}
	touches.Dragging(track)

	if got := touches.Damaged(); got != track {
		t.Errorf("damage while dragging is %+v, want the track %+v", got, track)
	}

	touches.Press(0, 0, false)
	if got := touches.Damaged(); got == track {
		t.Error("the track is still reported as damaged after the finger lifted")
	}
}

// Rows come and go as a page is rebuilt, and a press left pointing past the end would mark nothing
// or panic a view that trusted it.
func TestFewerPlacesDropAPressPastTheEnd(t *testing.T) {
	var touches Touches
	touches.Places(spots())
	touches.Press(250, 50, true)

	touches.Places(spots()[:1])

	if touches.Pressed(1) {
		t.Error("a press survives the place it was on going away")
	}
}

// Tapped runs what the place does, which is what keeps the rectangles and the actions from being
// two lists that can disagree.
func TestTappedRunsThePlacesAction(t *testing.T) {
	var touches Touches

	ran := ""
	touches.Spots([]Spot{
		{At: spots()[0], Do: func() { ran = "first" }},
		{At: spots()[1], Do: func() { ran = "second" }},
	})

	if !touches.Tapped(250, 50) {
		t.Fatal("tapping a place reports nothing there")
	}
	if ran != "second" {
		t.Errorf("the action that ran was %q, want second", ran)
	}

	ran = ""
	if touches.Tapped(150, 50) {
		t.Error("tapping between the places reports something there")
	}
	if ran != "" {
		t.Errorf("an action ran for a tap on nothing: %q", ran)
	}
}

// A place with nothing to do is a view that resolves taps itself.
func TestTappedIgnoresAPlaceWithNothingToDo(t *testing.T) {
	var touches Touches
	touches.Places(spots())

	if touches.Tapped(50, 50) {
		t.Error("a place with no action reports that it ran one")
	}
}

func TestHitFindsThePlace(t *testing.T) {
	var touches Touches
	touches.Places(spots())

	if at, ok := touches.Hit(250, 50); !ok || at != 1 {
		t.Errorf("hit is %d, %v, want 1, true", at, ok)
	}
	if _, ok := touches.Hit(150, 50); ok {
		t.Error("a point between the places hits one")
	}
}

package shell

import (
	"sync"

	"github.com/ygelfand/echolocal/internal/ui"
)

// Spot is somewhere on a view worth touching, and what touching it does. Do may be nil for a view
// that resolves taps itself.
type Spot struct {
	At ui.Rect
	Do func()
}

// Touches is the places on a view worth touching and which one a finger is on. A view embeds it to
// get press feedback without keeping the bookkeeping itself: say where the places are while drawing,
// and ask which is pressed.
//
// Locked throughout. Drawing runs on the render goroutine and touches arrive on another, and a press
// repaints, so a finger can be resolved against the places while they are being rebuilt.
type Touches struct {
	mu    sync.Mutex
	spots []Spot

	// on is one past the pressed place, so the zero value is a finger on nothing. mark is where
	// that place was, so lifting repaints it rather than the screen.
	on   int
	mark ui.Rect

	// drag is what is being pulled, which outranks a press: a slider being moved is what changed.
	drag ui.Rect
}

// Spots says where the touchable places are, in the order the view knows them. Said each time the
// view is drawn, since they move with the picture.
func (t *Touches) Spots(at []Spot) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.spots = at
	if t.on > len(at) {
		t.on = 0
	}
}

// Places is the shorthand for a view whose places do nothing by themselves.
func (t *Touches) Places(at []ui.Rect) {
	spots := make([]Spot, len(at))
	for i, r := range at {
		spots[i] = Spot{At: r}
	}
	t.Spots(spots)
}

// Pressed reports whether a finger is on the place at this index.
func (t *Touches) Pressed(i int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.on == i+1
}

// Dragging says what is being pulled. The zero rectangle clears it, which lifting does anyway.
func (t *Touches) Dragging(at ui.Rect) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.drag = at
}

// Press implements Presser.
func (t *Touches) Press(x, y int, down bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	was := t.on
	t.on = 0

	if !down {
		t.drag = ui.Rect{}
		return t.on != was
	}

	for i, s := range t.spots {
		if s.At.Contains(x, y) {
			t.on, t.mark = i+1, s.At
			break
		}
	}
	return t.on != was
}

// Damaged is what a touch changed: whatever is being pulled, or the place pressed or let go.
func (t *Touches) Damaged() ui.Rect {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.drag != (ui.Rect{}) {
		return t.drag
	}
	return t.mark
}

// Hit is the place under a point, for a view resolving a tap itself.
func (t *Touches) Hit(x, y int) (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, s := range t.spots {
		if s.At.Contains(x, y) {
			return i, true
		}
	}
	return 0, false
}

// Tapped runs what the place under a point does, and reports whether there was one to run.
func (t *Touches) Tapped(x, y int) bool {
	t.mu.Lock()
	var do func()
	for _, s := range t.spots {
		if s.At.Contains(x, y) {
			do = s.Do
			break
		}
	}
	t.mu.Unlock()

	// Outside the lock: what a button does may draw, which wants the places again.
	if do == nil {
		return false
	}
	do()
	return true
}

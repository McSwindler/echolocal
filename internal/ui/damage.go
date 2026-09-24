package ui

import "github.com/ygelfand/echolocal/internal/ui/theme"

// Within is a surface that only takes paint inside a rectangle.
//
// The same thing the driver does to the panel once damage has been declared, available over an
// image so a test can draw a narrowed repaint and check it against a full one. A declared rectangle
// is only right if those two agree.
//
// It reads back through whatever is underneath, which antialiasing needs, and reports the rectangle
// as its clip so drawing can skip work outside it rather than rasterising and discarding.
func Within(s Surface, to Rect) Surface { return within{under: s, to: to} }

type within struct {
	under Surface
	to    Rect
}

func (w within) Size() (int, int) { return w.under.Size() }
func (w within) Clipped() Rect    { return w.to }

func (w within) Set(x, y int, c theme.Color) {
	if !w.to.Contains(x, y) {
		return
	}
	w.under.Set(x, y, c)
}

func (w within) At(x, y int) theme.Color {
	if r, ok := w.under.(Reader); ok {
		return r.At(x, y)
	}
	return theme.Color{}
}

// Changed is the smallest rectangle holding every pixel where two pictures differ, and is empty
// when they do not differ at all.
//
// What it is for: checking a damage rectangle. Declaring damage is how a repaint is made cheap, and
// declaring one too small leaves a stale strip of screen that nothing catches until somebody looks
// at the panel. The check that does catch it is this — draw the state before, draw the state after,
// and see that everything which moved is inside what was declared.
//
// Pictures of different sizes have nothing meaningful to compare, so the whole of the larger one is
// the answer: a caller that has changed the size has changed everything.
func Changed(a, b *Image) Rect {
	aw, ah := a.Size()
	bw, bh := b.Size()

	if aw != bw || ah != bh {
		return Rect{W: max(aw, bw), H: max(ah, bh)}
	}

	left, top := aw, ah
	right, bottom := -1, -1

	for y := range ah {
		for x := range aw {
			if a.At(x, y) == b.At(x, y) {
				continue
			}
			if x < left {
				left = x
			}
			if x > right {
				right = x
			}
			if y < top {
				top = y
			}
			if y > bottom {
				bottom = y
			}
		}
	}

	if right < 0 {
		return Rect{}
	}
	return Rect{X: left, Y: top, W: right - left + 1, H: bottom - top + 1}
}

// Union is the smallest rectangle holding both, ignoring an empty one: a part that is not drawn
// contributes nothing rather than dragging the rectangle to the origin.
func (r Rect) Union(o Rect) Rect {
	if o.W <= 0 || o.H <= 0 {
		return r
	}
	if r.W <= 0 || r.H <= 0 {
		return o
	}

	left := min(r.X, o.X)
	top := min(r.Y, o.Y)
	right := max(r.X+r.W, o.X+o.W)
	bottom := max(r.Y+r.H, o.Y+o.H)

	return Rect{X: left, Y: top, W: right - left, H: bottom - top}
}

// Holds reports whether r covers the whole of inner, which is what a damage rectangle has to do for
// the pixels that changed.
//
// An empty inner is held by anything: nothing changed, so there is nothing to cover.
func (r Rect) Holds(inner Rect) bool {
	if inner.W <= 0 || inner.H <= 0 {
		return true
	}
	if r.W <= 0 || r.H <= 0 {
		return false
	}
	return inner.X >= r.X && inner.Y >= r.Y &&
		inner.X+inner.W <= r.X+r.W && inner.Y+inner.H <= r.Y+r.H
}

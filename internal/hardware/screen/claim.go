package screen

import (
	"image"
	"slices"
	"sync"
)

// Priority is how the panel resolves being asked for two things at once. Higher wins, and when a
// claim goes away whatever is under it comes back on its own.
type Priority int

const (
	// PriorityIdle is what the screen shows when nothing is happening.
	PriorityIdle Priority = iota

	// PrioritySetup is a device that cannot do its job yet and is asking to be set up.
	PrioritySetup

	// PriorityUI is a screen someone opened and is touching.
	PriorityUI

	// PriorityNotice is a brief acknowledgement, such as a volume change.
	PriorityNotice

	// PriorityAlert is something the user has to see.
	PriorityAlert

	// PriorityBoot is start-up. Nothing else may write while it holds.
	PriorityBoot
)

// Claim is one thing's hold on the panel. It is safe from any goroutine, and safe to keep after
// releasing: everything on a released claim does nothing.
type Claim struct {
	screen   *Screen
	priority Priority

	// covers says the claim paints the whole panel. One that does not is drawn on top of whatever
	// is under it, which has to be drawn first.
	covers bool

	mu       sync.Mutex
	draw     func(*Panel) error
	released bool

	// dirty is whether the claim has changed since it was drawn, damage which part of it. Nothing
	// changed and everything changed are different answers.
	dirty  bool
	damage image.Rectangle
}

// Claim asks for the panel at a priority, for something that covers it. Nothing is shown until the
// claim is given something to draw, and whatever it draws lasts until it is released or something
// higher takes over.
func (s *Screen) Claim(p Priority) *Claim { return s.claim(p, true) }

// Overlay is a claim that draws over what is beneath rather than replacing it. Everything under it
// is drawn first, so it composites onto what is really there.
func (s *Screen) Overlay(p Priority) *Claim { return s.claim(p, false) }

func (s *Screen) claim(p Priority, covers bool) *Claim {
	c := &Claim{screen: s, priority: p, covers: covers}

	s.mu.Lock()
	at := len(s.claims)
	for i, held := range s.claims {
		if held.priority > p {
			at = i
			break
		}
	}
	s.claims = slices.Insert(s.claims, at, c)
	s.forced = true
	s.mu.Unlock()

	s.wake()
	return c
}

// Repaint redraws every claim, for a change none of them reported: a new palette is the whole
// panel at once, whatever each claim thinks it is showing.
func (s *Screen) Repaint() {
	s.mu.Lock()
	s.forced = true
	s.mu.Unlock()

	s.wake()
}

// Show gives the claim something to draw. draw is called on the render goroutine, so it must not
// block on anything slow.
func (c *Claim) Show(draw func(*Panel) error) { c.show(image.Rectangle{}, draw) }

// ShowIn gives the claim something to draw and says which part of the panel it changes, so the
// repaint costs that rectangle rather than the screen.
//
// The rectangle has to hold every pixel that differs from the frame before, including the ones
// below an overlay, because everything is clipped to it. One too small leaves a stale strip that
// no later repaint corrects.
func (c *Claim) ShowIn(damage image.Rectangle, draw func(*Panel) error) { c.show(damage, draw) }

func (c *Claim) show(damage image.Rectangle, draw func(*Panel) error) {
	c.mu.Lock()
	c.draw = draw
	c.mark(damage)
	released := c.released
	c.mu.Unlock()

	if !released {
		c.screen.wake()
	}
}

// mark adds to what the claim has changed since it was last drawn. Several repaints can land
// between two frames, and one rectangle replacing another leaves the first unpainted for good.
//
// Called with the claim held.
func (c *Claim) mark(damage image.Rectangle) {
	switch {
	case !c.dirty:
		c.damage = damage
	case c.damage.Empty() || damage.Empty():
		c.damage = image.Rectangle{}
	default:
		c.damage = c.damage.Union(damage)
	}
	c.dirty = true
}

// Refresh redraws what the claim is already showing, limited to a region. For a view that changed
// itself rather than being handed something new to draw.
func (c *Claim) Refresh(damage image.Rectangle) {
	c.mu.Lock()
	drawing := c.draw != nil
	if drawing {
		c.mark(damage)
	}
	released := c.released
	c.mu.Unlock()

	if drawing && !released {
		c.screen.wake()
	}
}

// Clear gives up the surface without releasing the claim.
func (c *Claim) Clear() { c.Show(nil) }

// Release gives the panel back to whatever was underneath, which has to be drawn again.
func (c *Claim) Release() {
	c.mu.Lock()
	if c.released {
		c.mu.Unlock()
		return
	}
	c.released = true
	c.mu.Unlock()

	s := c.screen
	s.mu.Lock()
	for i, held := range s.claims {
		if held == c {
			s.claims = append(s.claims[:i], s.claims[i+1:]...)
			break
		}
	}
	s.forced = true
	s.mu.Unlock()

	s.wake()
}

// stack is the claims to draw, lowest first, and whether the panel has to be cleared under them.
//
// Everything from the topmost coverer upwards: what is below one is hidden, and an overlay with
// nothing under it needs the page cleared or it composites onto two frames ago.
func (s *Screen) stack() (draw []*Claim, clear bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	top := -1
	for i, c := range s.claims {
		c.mu.Lock()
		showing := c.draw != nil && c.covers
		c.mu.Unlock()

		if showing {
			top = i
		}
	}

	from := max(top, 0)
	for _, c := range s.claims[from:] {
		draw = append(draw, c)
	}
	return draw, top < 0
}

// damageOf is the region the stack changed, and whether anything did. An empty rectangle from any
// dirty claim means all of it.
//
// It takes the damage as it reads it. A repaint that landed between reading and clearing would
// otherwise be cleared without ever being drawn, and the wake it asked for finds nothing dirty.
func damageOf(stack []*Claim) (image.Rectangle, bool) {
	var region image.Rectangle
	var any, whole bool

	for _, c := range stack {
		c.mu.Lock()
		dirty, damage := c.dirty, c.damage
		c.damage, c.dirty = image.Rectangle{}, false
		c.mu.Unlock()

		if !dirty {
			continue
		}
		any = true
		if damage.Empty() {
			whole = true
			continue
		}
		region = region.Union(damage)
	}

	if whole {
		return image.Rectangle{}, true
	}
	return region, any
}

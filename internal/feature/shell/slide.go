package shell

import (
	"time"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// How long a page takes to arrive, and how often it is asked to be redrawn on the way.
//
// The step is what a smooth slide would want. It does not get it: two pages onto the panel take
// about fifty milliseconds, so two hundred is four frames. Ticks that arrive with the last frame
// still going are dropped, which is why asking for more costs nothing.
const (
	slideFor  = 200 * time.Millisecond
	slideStep = 16 * time.Millisecond
)

// slide is one page going away while another comes in.
type slide struct {
	from View

	// back is going out the way it came: the page that is leaving moves right, and the one
	// underneath comes back in from the left.
	back bool

	began time.Time
}

// begin is a slide between two views, or nil when there is nothing to animate. Both have to cover
// the screen: an overlay has the dashboard showing through it, and sliding one would drag a hole
// across the picture.
func begin(from, to View, back bool) *slide {
	if from == nil || to == nil || from == to {
		return nil
	}
	if !from.Covers() || !to.Covers() {
		return nil
	}
	return &slide{from: from, back: back, began: time.Now()}
}

// traveled is how far along the slide is, nought to one.
//
// Linear, not eased. A slide gets four frames on this panel, and easing out spends three of them
// in the last tenth of the travel: one jump and then a stop, where even steps read as a wipe.
func (s *slide) traveled() float64 {
	t := float64(time.Since(s.began)) / float64(slideFor)
	return max(0, min(t, 1))
}

func (s *slide) done() bool { return time.Since(s.began) >= slideFor }

// draw paints both pages where they have got to.
//
// Each is drawn onto the panel rather than composed in memory first. Copying a picture across is
// a store per pixel into a buffer the controller reads uncached, where drawing a page is mostly
// whole rows at a time — measured on the device, composing first was half the frame rate.
func (s *slide) draw(to View, surface ui.Surface, palette theme.Theme) {
	w, _ := surface.Size()
	at := int(s.traveled() * float64(w))

	leaving, arriving := -at, w-at
	if s.back {
		leaving, arriving = at, at-w
	}

	s.from.Draw(ui.Shift(surface, leaving, 0), palette)
	to.Draw(ui.Shift(surface, arriving, 0), palette)
}

// running plays the slide out, redrawing until it has arrived. It stops the moment something else
// takes over: a second push part way through replaces the slide, and this one is no longer it.
func (s *Shell) running(m *slide) {
	t := time.NewTicker(slideStep)
	defer t.Stop()

	for range t.C {
		s.mu.Lock()
		mine := s.sliding == m
		last := m.done()
		if mine && last {
			s.sliding = nil
		}
		s.mu.Unlock()

		if !mine {
			return
		}

		s.Redraw()

		if last {
			return
		}
	}
}

// start puts a slide up and plays it, which is what push and pop do once they have changed the
// stack.
func (s *Shell) start(m *slide) {
	s.Redraw()
	if m != nil {
		go s.running(m)
	}
}

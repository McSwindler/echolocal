package screen

import (
	"context"
	"fmt"
	"image"
	"log/slog"
)

// Run draws whatever holds the panel, until ctx is cancelled.
//
// One goroutine, because it is the only one that may touch the page being drawn into: a claim's
// draw is called from here rather than wherever it was handed over.
func (s *Screen) Run(ctx context.Context) error {
	if s.Panel() == nil {
		<-ctx.Done()
		return nil
	}

	for {
		if err := s.render(); err != nil {
			slog.Error("drawing the panel failed", "err", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-s.woken():
		}
	}
}

// wake asks for a frame. Never blocks: one pending is as good as several.
func (s *Screen) wake() {
	s.mu.Lock()
	ch := s.changed
	s.mu.Unlock()

	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (s *Screen) woken() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.changed == nil {
		s.changed = make(chan struct{}, 1)
	}
	return s.changed
}

// render draws the claims that are showing, lowest first.
func (s *Screen) render() error {
	p := s.Panel()
	if p == nil {
		return nil
	}

	stack, clear := s.stack()
	if len(stack) == 0 {
		return nil
	}

	now, any := damageOf(stack)

	s.mu.Lock()
	forced := s.forced
	s.mu.Unlock()

	// Flipping without drawing would put up the buffer from two frames ago.
	if !any && !forced {
		return nil
	}

	if forced {
		now = image.Rectangle{}

		s.mu.Lock()
		s.forced = false
		s.mu.Unlock()
	}

	painted := now
	if painted.Empty() {
		painted = image.Rect(0, 0, p.Width, p.Height)
	}

	s.mu.Lock()
	region := painted
	for _, was := range s.damaged {
		region = region.Union(was)
	}

	s.damaged = append(s.damaged, painted)
	if keep := max(p.Pages()-1, 0); len(s.damaged) > keep {
		s.damaged = s.damaged[len(s.damaged)-keep:]
	}
	s.mu.Unlock()

	p.Clip(region.Min.X, region.Min.Y, region.Dx(), region.Dy())
	defer p.Clip(0, 0, 0, 0)

	// An overlay with nothing under it would otherwise sit on whatever the page last held.
	if clear {
		p.Fill(Color{A: 0xFF})
	}

	for _, c := range stack {
		c.mu.Lock()
		draw := c.draw
		c.mu.Unlock()

		if draw == nil {
			continue
		}
		if err := draw(p); err != nil {
			return fmt.Errorf("screen: %w", err)
		}
	}
	return p.Flip()
}

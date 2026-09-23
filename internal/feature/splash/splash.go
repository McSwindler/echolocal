// Package splash is what the panel shows while the device is coming up: the mark, then what it is
// still waiting for.
package splash

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/theme"
	"github.com/ygelfand/echolocal/internal/hardware/screen"
	"github.com/ygelfand/echolocal/internal/ui"
	uitheme "github.com/ygelfand/echolocal/internal/ui/theme"
)

func init() {
	// First of the hardware, for the same reason the ring is: it is the only thing the device can
	// say before anything else works.
	component.Register(component.Hardware, Get, component.Order(5), component.Needs(board.Panel))
}

const (
	// logoFor is how long the mark is up before the screen starts saying what it is waiting for.
	// Long enough to be a greeting, short enough that a stuck device says so quickly.
	logoFor = 2 * time.Second

	// listFor is the least the list stays up once it has appeared, so a device that comes up in a
	// second does not flash it past unread.
	listFor = 3 * time.Second

	// look is how often the screen is reconsidered.
	look = 250 * time.Millisecond
)

type Splash struct{}

var (
	once   sync.Once
	shared *Splash
)

func Get() *Splash { once.Do(func() { shared = &Splash{} }); return shared }

func (s *Splash) Name() string { return "splash" }

// Start puts the mark up, full screen, before anything else has run.
func (s *Splash) Start(context.Context) error {
	p := screen.Get().Panel()
	if p == nil {
		return nil
	}

	t := theme.Get().Current()
	bg := colour(t.Background)

	p.Fill(bg)
	p.Draw(ui.Mark(t.Dark), p.Bounds(), bg)

	if err := p.Flip(); err != nil {
		slog.Error("drawing the splash failed", "err", err)
	}
	return nil
}

// Run shows the mark for a moment, then what the device is still waiting for, and finishes once
// everything is up.
func (s *Splash) Run(ctx context.Context) error {
	p := screen.Get().Panel()
	if p == nil {
		<-ctx.Done()
		return nil
	}

	if !wait(ctx, logoFor) {
		return nil
	}

	t := time.NewTicker(look)
	defer t.Stop()

	// said starts as something no summary can be, so the first pass always draws.
	said := "\x00"
	appeared := time.Now()

	draw := func() {
		progress := component.Default().Progress()

		// Redrawn only when something changed, since the panel is written pixel by pixel.
		now := summary(progress)
		if now == said {
			return
		}
		said = now

		drawBoot(p, theme.Get().Current(), progress)
		if err := p.Flip(); err != nil {
			slog.Error("drawing the boot screen failed", "err", err)
		}
		slog.Info("coming up", "waiting", now)
	}

	for !component.Default().Ready() || time.Since(appeared) < listFor {
		draw()

		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}

	// What settled last changed after the final pass, so the panel is still showing it waiting.
	draw()

	slog.Info("ready")
	return nil
}

// summary is what the screen currently says, for deciding whether to draw it again.
func summary(progress []component.Progress) string {
	var s string
	for _, p := range progress {
		if !p.Settled() {
			s += p.Name + ":" + p.Doing + " "
		}
	}
	return s
}

func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func colour(c uitheme.Color) screen.Color { return screen.Opaque(c.R, c.G, c.B) }

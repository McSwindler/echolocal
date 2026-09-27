// Package splash is what the panel shows while the device is coming up: the mark, then what it is
// still waiting for.
package splash

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/api"
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

type Splash struct{ hold *screen.Claim }

var (
	once   sync.Once
	shared *Splash
)

func Get() *Splash { once.Do(func() { shared = &Splash{} }); return shared }

func (s *Splash) Name() string { return "splash" }

// Start puts the mark up, full screen, before anything else has run.
func (s *Splash) Start(context.Context) error {
	if screen.Get().Panel() == nil {
		return nil
	}

	s.hold = screen.Get().Claim(screen.PriorityBoot)
	s.hold.Show(func(p *screen.Panel) error {
		t := theme.Get().Current()
		bg := colour(t.Background)

		p.Fill(bg)
		p.Draw(ui.Mark(t.Dark), p.Bounds(), bg)
		return nil
	})
	return nil
}

// Run shows the mark for a moment, then what the device is still waiting for, and finishes once
// everything is up.
func (s *Splash) Run(ctx context.Context) error {
	if s.hold == nil {
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

		s.hold.Show(func(p *screen.Panel) error {
			drawBoot(p, theme.Get().Current(), progress)
			return nil
		})
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
	return s.onboard(ctx)
}

// onboard holds the pairing code up until Home Assistant has the device. A device already adopted
// releases the panel to whatever is under it.
func (s *Splash) onboard(ctx context.Context) error {
	if api.Adopted() {
		s.hold.Release()
		return nil
	}

	// Setup rather than boot: the device is up, it is just not anybody's yet, so a notice or an
	// alert belongs over the top of this.
	s.hold.Release()
	s.hold = screen.Get().Claim(screen.PrioritySetup)

	slog.Info("waiting to be added to home assistant")

	t := time.NewTicker(look)
	defer t.Stop()

	said := "\x00"
	for !api.Adopted() {
		key := pairing()
		if !networked() {
			key = ""
		}

		if key != said {
			said = key
			s.hold.Show(func(p *screen.Panel) error {
				drawOnboard(p, theme.Get().Current(), key)
				return nil
			})
		}

		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}

	s.hold.Release()
	return nil
}

// summary is what the screen currently says, for deciding whether to draw it again.
func summary(progress []component.Progress) string {
	var s string
	for _, p := range progress {
		s += fmt.Sprintf("%s:%v:%s ", p.Name, p.Settled(), p.Doing)
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

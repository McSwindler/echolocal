// Package player is what is playing, on the screen: a card with the transport, and a bar along the
// bottom of the clock once the card has been left.
package player

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/drawer"
	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/hardware/touch"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
)

func init() {
	component.Register(component.Device, Get, component.Order(39), component.Needs(board.Panel))
}

const (
	look = 250 * time.Millisecond
	gap  = 5 * time.Second
)

type Player struct {
	mu      sync.Mutex
	held    *shell.Hold
	sounded time.Time
	begun   media.Source
}

var (
	once   sync.Once
	shared *Player
)

func Get() *Player {
	once.Do(func() {
		shared = &Player{}

		drawer.Get().Add(drawer.Entry{
			Name:  func() string { return say.T("rail.media") },
			Order: drawer.OrderPlayer,
			Icon:  func() ui.Icon { return icons.AVPlayCircleOutline },
			Open: func() {
				if now := media.Get().Now(); now.Playing || now.Paused {
					shared.Open()
					return
				}
				shell.Get().Push(Page())
			},
		})

		media.Get().Volume.Listen(func(int) { shared.refresh() })
		media.Get().Begun.Listen(func(s media.Source) {
			shared.mu.Lock()
			shared.begun = s
			shared.mu.Unlock()
		})
	})
	return shared
}

func (p *Player) Name() string { return "player" }

func (p *Player) Run(ctx context.Context) error {
	tick := time.NewTicker(look)
	defer tick.Stop()

	small := &mini{p: p}
	defer touch.Gestures.Listen(small.on)()
	defer small.hide()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
		p.follow()
		p.tick()
		small.follow()
	}
}

// Showing reports whether the card is the screen being looked at.
func Showing() bool { return shell.Get().Top() == Page() }

func (p *Player) refresh() {
	if Showing() {
		shell.Get().Redraw()
	}
}

func (p *Player) tick() {
	if !Showing() {
		return
	}
	now := media.Get().Now()
	if !now.Playing || now.Length <= 0 {
		return
	}
	if strip, ok := page.stale(now.Elapsed); ok {
		shell.Get().RedrawIn(strip)
	}
}

// follow opens the card when something starts, and lets it go once things have been quiet a while.
func (p *Player) follow() {
	now := media.Get().Now()
	playing := now.Playing || now.Hold
	src := media.Get().Holder()

	p.mu.Lock()
	begun := p.begun
	p.begun = nil
	if playing {
		p.sounded = time.Now()
	}
	held, quiet := p.held, time.Since(p.sounded)
	p.mu.Unlock()

	switch {
	case begun != nil && begun == src:
		slog.Info("the player opened", "by", fmt.Sprintf("%T", src))
		p.Open()

	case !playing && held.Held() && quiet > gap:
		slog.Info("the player is done", "quiet", quiet.Round(time.Second))
		held.Keep(false)
	}
}

// Open puts up whatever shows what is playing: the source's own screen where it has one, the card
// otherwise.
func (p *Player) Open() {
	if o, ok := media.Get().Holder().(media.Opener); ok && o.Open() {
		p.mu.Lock()
		card := p.held
		p.held = nil
		p.mu.Unlock()
		card.Release()
		return
	}
	p.mu.Lock()
	held := p.held
	p.mu.Unlock()
	if held.Held() {
		return
	}
	hold := shell.Get().Hold(Page())
	p.mu.Lock()
	p.held = hold
	p.mu.Unlock()
}

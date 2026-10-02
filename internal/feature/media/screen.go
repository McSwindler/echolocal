//go:build board_checkers || board_cronos || board_rook

package media

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/drawer"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
)

func init() {
	component.Register(component.Device, card, component.Order(39), component.Needs(board.Panel))
}

const (
	look = 250 * time.Millisecond
	gap  = 5 * time.Second
)

// Card is what is playing on the screen: it opens when something starts and goes once things have
// been quiet a while.
type Card struct {
	mu      sync.Mutex
	held    *shell.Hold
	sounded time.Time
	begun   Source
}

var (
	cardOnce sync.Once
	theCard  *Card
)

func card() *Card {
	cardOnce.Do(func() {
		theCard = &Card{}
		onRail()
		Get().Begun.Listen(func(s Source) {
			theCard.mu.Lock()
			theCard.begun = s
			theCard.mu.Unlock()
		})
	})
	return theCard
}

func (c *Card) Name() string { return "player" }

func (c *Card) Run(ctx context.Context) error {
	tick := time.NewTicker(look)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
		c.follow()
		c.tick()
	}
}

func (c *Card) tick() {
	if !Showing() {
		return
	}
	now := Get().Now()
	if !now.Playing || now.Length <= 0 {
		return
	}
	if Page().(*screen).moved(now.Elapsed) {
		shell.Get().Redraw()
	}
}

func (c *Card) follow() {
	now := Get().Now()
	playing := now.Playing || now.Hold
	src := Get().Holder()

	c.mu.Lock()
	begun := c.begun
	c.begun = nil
	if playing {
		c.sounded = time.Now()
	}
	held, quiet := c.held, time.Since(c.sounded)
	c.mu.Unlock()

	switch {
	case begun != nil && begun == src:
		slog.Info("the player opened", "by", fmt.Sprintf("%T", src))
		c.open()

	case !playing && held.Held() && quiet > gap:
		slog.Info("the player is done", "quiet", quiet.Round(time.Second))
		held.Keep(false)
	}
}

// Open shows the player for whatever is playing: the source's own if it has one, the card if not.
func (p *Player) Open() { card().open() }

func (c *Card) open() {
	if o, ok := Get().Holder().(Opener); ok && o.Open() {
		c.mu.Lock()
		held := c.held
		c.held = nil
		c.mu.Unlock()
		held.Release()
		return
	}
	c.mu.Lock()
	held := c.held
	c.mu.Unlock()
	if held.Held() {
		return
	}
	hold := shell.Get().Hold(Page())
	c.mu.Lock()
	c.held = hold
	c.mu.Unlock()
}

// Page is what is playing, with the transport for it. One instance, so the shell can recognize it.
func Page() shell.View {
	pageOnce.Do(func() { page = &screen{} })
	return page
}

var (
	pageOnce sync.Once
	page     *screen
)

// Showing reports whether the player is the screen being looked at.
func Showing() bool { return shell.Get().Top() == Page() }

func onRail() {
	drawer.Get().Add(drawer.Entry{
		Name:  func() string { return say.T("rail.media") },
		Order: drawer.OrderPlayer,
		Glyph: func() string { return gogui.IconMusic },
		Open: func() {
			if now := Get().Now(); now.Playing || now.Paused {
				Get().Open()
				return
			}
			shell.Get().Push(Page())
		},
	})
}

type screen struct {
	mu    sync.Mutex
	shown time.Duration
}

func (v *screen) Covers() bool { return true }

func (v *screen) Shows(s config.Stream) bool { return s == config.StreamMedia }

func (v *screen) Timeout() time.Duration { return shell.SettingsTimeout }

func (v *screen) moved(elapsed time.Duration) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	second := elapsed / time.Second
	if second == v.shown {
		return false
	}
	v.shown = second
	return true
}

func Heading(now Now) string {
	switch {
	case now.Title != "":
		return now.Title
	case now.Paused:
		return "Paused"
	case now.Playing:
		return "Playing"
	}
	return "Nothing playing"
}

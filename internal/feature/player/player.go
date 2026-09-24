// Package player is what is playing, with the transport for it.
package player

import (
	"sync"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/drawer"
	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
)

func init() {
	component.Register(component.Device, Get, component.Order(39), component.Needs(board.Panel))
}

type Player struct{}

var (
	once   sync.Once
	shared *Player
)

func Get() *Player {
	once.Do(func() {
		shared = &Player{}

		drawer.Get().Add(drawer.Entry{
			Name:  func() string { return say.T("player.title") },
			Order: drawer.OrderPlayer,
			Icon: func() ui.Icon {
				if now := media.Get().Now(); now.Playing && !now.Paused {
					return icons.AVPause
				}
				return icons.AVPlayArrow
			},
			Open: Open,
		})

		media.Get().Volume.Listen(func(int) { shared.refresh() })
	})
	return shared
}

func (p *Player) Name() string { return "player" }

// Open puts the player up.
func Open() { shell.Get().Push(Page()) }

// Showing reports whether the player is the screen being looked at.
func Showing() bool { return shell.Get().Top() == Page() }

// refresh repaints the player while it is the screen being looked at.
func (p *Player) refresh() {
	if Showing() {
		shell.Get().Redraw()
	}
}

// Package clock is what the panel shows once the device is up.
package clock

import (
	"context"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/theme"
	"github.com/ygelfand/echolocal/internal/hardware/screen"
)

func init() {
	// Last of the device phase: it takes the panel over from the boot screen.
	component.Register(component.Device, Get, component.Order(90), component.Needs(board.Panel))
}

// look is how often the reading is taken. The panel is only written when it changed, which is once
// a minute.
const look = time.Second

// twentyFour is the format until there is somewhere to choose one.
const twentyFour = true

type Clock struct {
	mu    sync.Mutex
	asked *esphome.Conn
	tz    string
}

var (
	once   sync.Once
	shared *Clock
)

func Get() *Clock { once.Do(func() { shared = &Clock{} }); return shared }

func (c *Clock) Name() string { return "clock" }

// Run draws the time until ctx is cancelled, once the boot screen has finished with the panel.
func (c *Clock) Run(ctx context.Context) error {
	if screen.Get().Panel() == nil {
		<-ctx.Done()
		return nil
	}

	// Claimed at once rather than waiting for the boot screen to finish: it holds a higher
	// priority, so this sits underneath and appears when it lets go.
	hold := screen.Get().Claim(screen.PriorityIdle)
	defer hold.Release()

	t := time.NewTicker(look)
	defer t.Stop()

	var said string
	for {
		if r := Read(time.Now(), twentyFour); r.String() != said {
			said = r.String()

			hold.Show(func(p *screen.Panel) error {
				drawFace(p, theme.Get().Current(), r)
				return nil
			})
		}

		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

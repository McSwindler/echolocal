// Package brightness drives the panel's backlight, from the slider or from the room's light.
package brightness

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/screen"
	"github.com/ygelfand/echolocal/internal/hardware/alsps"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/lib/safe"
)

func init() {
	component.Register(component.Device, Get, component.Order(41), component.Needs(board.Panel))
}

const (
	// settle is how much of each reading moves the level the backlight follows.
	settle = 0.25
	tick   = 250 * time.Millisecond

	// change is how far a reading must move from the last one sent, as a fraction of it.
	change = 0.05

	// The driver reports whole lux.
	leastLux = 1

	// stale is the longest Home Assistant goes without hearing the light.
	stale = 5 * time.Minute
)

type Auto struct {
	ambient *esphome.Sensor

	mu     sync.Mutex
	sensor *alsps.Sensor

	following float64
	followed  bool
	held      float64
	shown     int
	applied   int
	toldLux   told
}

var (
	once   sync.Once
	shared *Auto
)

func Get() *Auto {
	once.Do(func() { shared = build() })
	return shared
}

func build() *Auto {
	return &Auto{
		ambient: &esphome.Sensor{
			Base: esphome.Base{
				ObjectID: "ambient_light", Name: "Ambient light", Icon: "mdi:brightness-5",
				DeviceID: component.DeviceScreen,
			},
			Unit: "lx", DeviceClass: "illuminance", StateClass: esphome.StateClassMeasurement,
		},
	}
}

func (a *Auto) Name() string { return "brightness" }

func (a *Auto) Entities() []esphome.Entity {
	return []esphome.Entity{a.ambient}
}

func (a *Auto) Start(context.Context) error {
	s, err := alsps.Open()
	if err != nil {
		slog.Warn("no ambient light sensor, brightness is manual only", "err", err)
		return nil
	}
	a.mu.Lock()
	a.sensor = s
	a.mu.Unlock()
	return nil
}

func (a *Auto) Run(ctx context.Context) error {
	a.mu.Lock()
	s := a.sensor
	a.mu.Unlock()
	if s == nil {
		<-ctx.Done()
		return nil
	}

	failed := make(chan error, 1)
	safe.Go("ambient light", func() { failed <- s.Run(ctx) })

	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-failed:
			return err
		case <-t.C:
		}
		if lux, ok := s.Lux(); ok {
			a.follow(lux)
		}
	}
}

func (a *Auto) Close() error {
	a.mu.Lock()
	s := a.sensor
	a.sensor = nil
	a.mu.Unlock()
	if s == nil {
		return nil
	}
	return s.Close()
}

// Ambient is the smoothed light the backlight follows, and false before there has been a reading.
func (a *Auto) Ambient() (float64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.following, a.followed
}

func (a *Auto) follow(lux float64) {
	a.mu.Lock()
	if !a.followed {
		a.following, a.followed, a.held = lux, true, lux
	}
	a.following += settle * (lux - a.following)
	following := a.following
	report := a.toldLux.worth(following, leastLux)

	cfg := config.Get()
	var want int
	set := false
	if cfg.Screen.Mode == config.ModeAuto {
		a.held = hold(a.held, a.following, lux)
		want = Brightness(a.held, cfg.Screen.Backlight)
		if a.shown == 0 {
			a.shown = want
		}
		a.shown = ramp(a.shown, want)
		if a.shown != a.applied {
			a.applied, set = a.shown, true
		}
		want = a.shown
	} else {
		a.applied = 0
	}
	a.mu.Unlock()

	if report {
		a.ambient.Set(float32(math.Round(following*10) / 10))
	}
	if set {
		a.apply(want)
	}
}

func (a *Auto) apply(level int) {
	if err := display.Get().Brightness(level); err != nil {
		slog.Warn("setting the backlight failed", "err", err)
		return
	}
	screen.Get().Lit(level)
}

// told is the last reading handed to Home Assistant, and when.
type told struct {
	value float64
	at    time.Time
	sent  bool
}

func (t *told) worth(v, least float64) bool {
	now := time.Now()

	switch {
	case !t.sent, now.Sub(t.at) >= stale:
	case math.Abs(v-t.value) >= max(least, math.Abs(t.value)*change):
	default:
		return false
	}

	t.value, t.at, t.sent = v, now, true
	return true
}

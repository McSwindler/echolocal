// Package clock is what the panel shows once the device is up.
package clock

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/clock/face"
	"github.com/ygelfand/echolocal/internal/feature/theme"
	"github.com/ygelfand/echolocal/internal/feature/timesync"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/ui"
	uitheme "github.com/ygelfand/echolocal/internal/ui/theme"
)

func init() {
	component.Register(component.Device, Get, component.Order(90), component.Needs(board.Panel))
}

const tick = time.Second

type Clock struct {
	mu sync.Mutex

	format *esphome.Select
	face   *esphome.Select
	place  *esphome.Select
	size   *esphome.Select
	ink    *esphome.Select
	date   *esphome.Switch
	logo   *esphome.Switch

	claim    *display.Claim
	shown    string
	reading  face.Reading
	settings string
}

var (
	once   sync.Once
	shared *Clock
)

func Get() *Clock {
	once.Do(func() {
		shared = &Clock{}
		shared.build()
		theme.Changed.Listen(func(uitheme.Theme) { shared.Redraw() })
		timesync.Get().Synced.Listen(func(s timesync.Sync) {
			if s.Stepped() {
				shared.Redraw()
			}
		})
	})
	return shared
}

func (c *Clock) Name() string { return "clock" }

func (c *Clock) Entities() []esphome.Entity {
	return []esphome.Entity{c.format, c.face, c.place, c.size, c.ink, c.date, c.logo}
}

func (c *Clock) Restore(cfg config.Config) {
	c.format.Set(cfg.Screen.Hours.Label())
	c.face.Set(cfg.Clock.Face.Label())
	c.place.Set(cfg.Clock.Position.Label())
	c.size.Set(cfg.Clock.Size.Label())
	c.ink.Set(cfg.Clock.Ink.Label())
	c.date.Set(cfg.Clock.Date)
	c.logo.Set(cfg.Screen.Logo)
}

// Run draws the time until ctx is cancelled.
func (c *Clock) Run(ctx context.Context) error {
	t := time.NewTicker(tick)
	defer t.Stop()

	for !timesync.Get().Valid() {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}

	claim := display.Get().Claim(display.PriorityDashboard)
	defer claim.Release()

	c.mu.Lock()
	c.claim = claim
	c.mu.Unlock()

	for {
		c.paint(time.Now())

		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func (c *Clock) paint(at time.Time) {
	w, h := display.Get().Size()
	if w == 0 {
		return
	}

	cfg := config.Get()
	reading := face.Read(at, cfg.Screen.Hours != config.TwelveHour)
	if !cfg.Clock.Date {
		reading = reading.Undated()
	}

	kind := cfg.Clock.Face
	mark := cfg.Screen.Logo
	palette := theme.Get().Current()
	box := Box(cfg.Clock.Position, cfg.Clock.Size, w, h)

	settings := fmt.Sprintf("face=%s at=%s size=%s ink=%s mark=%v theme=%v",
		kind, cfg.Clock.Position, cfg.Clock.Size, cfg.Clock.Ink, mark, palette)
	key := reading.String() + " " + settings
	if face.Ticks(kind) {
		key = fmt.Sprintf("%s second=%d", key, reading.Second)
	}

	c.mu.Lock()
	claim, same := c.claim, c.shown == key
	was, wasSettings := c.reading, c.settings
	if claim != nil {
		c.shown, c.reading, c.settings = key, reading, settings
	}
	c.mu.Unlock()

	if claim == nil || same {
		return
	}

	draw := func(p *display.Panel) error {
		Draw(ui.Of(p), cfg.Clock, kind, reading, palette, mark)
		return nil
	}

	if wasSettings == settings {
		if r, ok := face.Damage(kind, box, was, reading); ok {
			claim.ShowIn(display.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, draw)
			return
		}
	}
	claim.Show(draw)
}

// Draw paints a whole clock screen.
func Draw(s ui.Surface, cfg config.Clock, kind config.Face, r face.Reading, palette uitheme.Theme, mark bool) {
	w, h := s.Size()
	ui.Fill(s, palette.Background)
	if mark {
		ui.DrawLogo(s, Mark(w, h), palette.Background)
	}
	face.Of(kind).Draw(s, Box(cfg.Position, cfg.Size, w, h), r, cfg.Ink.Over(palette))
}

// Redraw paints the whole screen again on the next look.
func (c *Clock) Redraw() {
	c.mu.Lock()
	c.shown, c.settings = "", ""
	c.mu.Unlock()

	c.paint(time.Now())
}

func (c *Clock) SetHours(v config.HourFormat) {
	c.pick(c.format, v.Label(), config.Set().Screen().Hours(v))
}

func (c *Clock) SetFace(v config.Face) { c.pick(c.face, v.Label(), config.Set().Clock().Face(v)) }

func (c *Clock) SetPosition(v config.Position) {
	c.pick(c.place, v.Label(), config.Set().Clock().Position(v))
}

func (c *Clock) SetSize(v config.Size) { c.pick(c.size, v.Label(), config.Set().Clock().Size(v)) }

func (c *Clock) SetInk(v config.Ink) { c.pick(c.ink, v.Label(), config.Set().Clock().Ink(v)) }

func (c *Clock) SetDate(on bool) { c.flip(c.date, on, config.Set().Clock().Date(on)) }

func (c *Clock) SetLogo(on bool) { c.flip(c.logo, on, config.Set().Screen().Logo(on)) }

func (c *Clock) pick(s *esphome.Select, label string, err error) {
	if err != nil {
		slog.Error("saving a setting failed", "setting", s.ObjectID, "err", err)
		return
	}
	s.Set(label)
	c.Redraw()
}

func (c *Clock) flip(s *esphome.Switch, on bool, err error) {
	if err != nil {
		slog.Error("saving a setting failed", "setting", s.ObjectID, "err", err)
		return
	}
	s.Set(on)
	c.Redraw()
}

func (c *Clock) build() {
	c.format = choice("clock_format", "Clock format", "mdi:clock-outline", config.HourFormats(), c.SetHours)
	c.face = choice("clock_face", "Clock face", "mdi:clock-digital", config.Faces(), c.SetFace)
	c.place = choice("clock_position", "Clock position", "mdi:align-vertical-center", config.Positions(), c.SetPosition)
	c.size = choice("clock_size", "Clock size", "mdi:format-size", config.Sizes(), c.SetSize)
	c.ink = choice("clock_color", "Clock color", "mdi:palette-outline", config.Inks(), c.SetInk)
	c.date = toggle("clock_date", "Clock date", "mdi:calendar-blank-outline", c.SetDate)
	c.logo = toggle("dashboard_logo", "Dashboard logo", "mdi:home-assistant", c.SetLogo)
}

func choice[T config.Labelled](id, name, icon string, values []T, use func(T)) *esphome.Select {
	s := &esphome.Select{
		Base: esphome.Base{
			ObjectID: id,
			Name:     name,
			Icon:     icon,
			Category: esphome.CategoryConfig,
			DeviceID: component.DeviceScreen,
		},
		Options: config.Labels(values),
	}
	s.OnCommand = func(label string) {
		if v, ok := config.ByLabel(values, label); ok {
			use(v)
		}
	}
	return s
}

func toggle(id, name, icon string, use func(bool)) *esphome.Switch {
	s := &esphome.Switch{
		Base: esphome.Base{
			ObjectID: id,
			Name:     name,
			Icon:     icon,
			Category: esphome.CategoryConfig,
			DeviceID: component.DeviceScreen,
		},
	}
	s.OnCommand = use
	return s
}

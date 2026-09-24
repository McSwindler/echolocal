// Package volume is how loud each kind of sound is, and the control for it on the panel.
package volume

import (
	"log/slog"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/hardware/speaker"
)

func init() {
	component.Register(component.Device, Get, component.Order(38), component.Needs(board.Panel))
}

type Volume struct {
	card    card
	numbers map[config.Stream]*esphome.Number
}

var (
	once   sync.Once
	shared *Volume
)

func Get() *Volume {
	once.Do(func() {
		shared = &Volume{numbers: map[config.Stream]*esphome.Number{}}
		shared.build()

		media.Get().Volume.Listen(func(int) { shared.changed(config.StreamMain) })
		media.Get().SetTurns(shared.turn)
	})
	return shared
}

func (v *Volume) Name() string { return "volume" }

func (v *Volume) Entities() []esphome.Entity {
	out := make([]esphome.Entity, 0, len(v.numbers))
	for _, s := range config.Streams() {
		if s == config.StreamMain {
			continue
		}
		out = append(out, v.numbers[s])
	}
	return out
}

// Restore puts the levels back where they were left.
func (v *Volume) Restore(c config.Config) {
	for _, s := range config.Streams() {
		if s == config.StreamMain {
			continue
		}
		v.hold(s, c.Volume.Level(s))
		slog.Info("restored", "what", v.numbers[s].ObjectID, "using", c.Volume.Level(s))
	}
}

// Level is how loud one kind of sound is, as a percentage.
func (v *Volume) Level(s config.Stream) int {
	if s == config.StreamMain {
		return media.Get().Step() * 100 / media.VolumeSteps
	}
	return config.Get().Volume.Level(s)
}

// Set changes a level and remembers it.
func (v *Volume) Set(s config.Stream, level int) {
	level = clamp(level)
	if v.Level(s) == level {
		return
	}

	if s == config.StreamMain {
		media.Get().Set((level*media.VolumeSteps + 50) / 100)
		return
	}

	v.hold(s, level)
	if err := config.Set().Volume().Level(s, level); err != nil {
		slog.Error("saving a volume failed", "stream", s, "err", err)
	}
	v.changed(s)
}

// Step is how far one press of a volume button moves a level.
const Step = 5

// turn moves whichever level is being shown, and leaves the device's own to the player.
func (v *Volume) turn(delta int) bool {
	s, ok := v.card.selected()
	if !ok {
		s, ok = wanted()
	}
	if !ok || s == config.StreamMain {
		return false
	}

	v.Set(s, v.Level(s)+delta*Step)
	return true
}

// wants is a view that says which level the buttons move while it is the screen being looked at.
type wants interface{ Wants() (config.Stream, bool) }

func wanted() (config.Stream, bool) {
	v, ok := shell.Get().Top().(wants)
	if !ok {
		return "", false
	}
	return v.Wants()
}

// hold applies a level without writing it back to the file it may have come from.
func (v *Volume) hold(s config.Stream, level int) {
	speaker.Get().SetLevel(s, clamp(level))
	v.numbers[s].Set(float32(clamp(level)))
}

// changed says what the level is now, except over a screen already showing it.
func (v *Volume) changed(s config.Stream) {
	if showing(s) {
		shell.Get().Redraw()
		return
	}
	v.card.show(s)
}

func (v *Volume) build() {
	for _, s := range config.Streams() {
		if s == config.StreamMain {
			continue
		}
		stream := s

		n := &esphome.Number{
			Base: esphome.Base{
				ObjectID: "volume_" + string(stream),
				DeviceID: component.DevicePlayback,
				Name:     stream.Label() + " volume",
				Icon:     "mdi:volume-high",
				Category: esphome.CategoryConfig,
			},
			Min: 0, Max: 100, Step: 1, Unit: "%",
			Mode: esphome.NumberSlider,
		}
		n.OnCommand = func(level float32) { shared.Set(stream, int(level)) }

		v.numbers[stream] = n
	}
}

// shows is a view that already has a level on it.
type shows interface{ Shows(config.Stream) bool }

func showing(s config.Stream) bool {
	v, ok := shell.Get().Top().(shows)
	return ok && v.Shows(s)
}

// Package privacy marks the panel while the microphones are cut or the camera is covered.
package privacy

import (
	"context"
	"log/slog"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/mute"
	"github.com/ygelfand/echolocal/internal/hardware/buttons"
	"github.com/ygelfand/echolocal/internal/lib/hook"
)

func init() {
	// After mute, whose line this shows.
	component.Register(component.Device, Get, component.Order(35), component.Needs(board.Panel))
}

type Privacy struct {
	Changed hook.Hook[Marks]

	camera *esphome.BinarySensor
	sw     *esphome.Switch

	mu    sync.Mutex
	marks Marks
}

var (
	once   sync.Once
	shared *Privacy
)

func Get() *Privacy {
	once.Do(func() {
		shared = &Privacy{
			camera: &esphome.BinarySensor{
				Base: esphome.Base{
					ObjectID: "camera_covered",
					Name:     "Camera covered",
					Icon:     "mdi:camera-off",
				},
			},
			sw: &esphome.Switch{
				Base: esphome.Base{
					ObjectID: "privacy_marks",
					Name:     "Privacy marks",
					Icon:     "mdi:shield-account-outline",
					Category: esphome.CategoryConfig,
				},
			},
		}
		shared.sw.OnCommand = shared.SetMarks

		buttons.Get().Shutter.Listen(shared.covered)
		mute.Get().Changed.Listen(shared.muted)
	})
	return shared
}

func (p *Privacy) Name() string { return "privacy" }

func (p *Privacy) Entities() []esphome.Entity {
	if !component.Board().Has(board.Shutter) {
		return []esphome.Entity{p.sw}
	}
	return []esphome.Entity{p.camera, p.sw}
}

// Restore puts the marks back to what was chosen.
func (p *Privacy) Restore(c config.Config) {
	p.sw.Set(c.Screen.Marks)
	p.show()
	slog.Info("restored", "what", p.sw.ObjectID, "on", c.Screen.Marks)
}

// SetMarks turns the corner marks on or off, and applies it at once.
func (p *Privacy) SetMarks(on bool) {
	if err := config.Set().Screen().Marks(on); err != nil {
		slog.Error("saving the privacy marks setting failed", "err", err)
		return
	}
	p.sw.Set(on)
	p.show()
}

// Start reads where both controls already are. Neither says anything until it is next touched.
func (p *Privacy) Start(context.Context) error {
	if component.Board().Has(board.Shutter) {
		p.covered(buttons.Get().Covered())
	}

	muted, err := mute.Get().Muted()
	if err != nil {
		slog.Warn("cannot tell whether the microphones are cut", "err", err)
	}
	p.muted(muted)
	return nil
}

func (p *Privacy) covered(on bool) {
	p.camera.Set(on)

	p.mu.Lock()
	p.marks.CameraBlocked = on
	p.mu.Unlock()

	p.show()
}

func (p *Privacy) muted(on bool) {
	p.mu.Lock()
	p.marks.MicMuted = on
	p.mu.Unlock()

	p.show()
}

// Current is the marks to show, none while they are turned off.
func (p *Privacy) Current() Marks {
	if !config.Get().Screen.Marks {
		return Marks{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.marks
}

func (p *Privacy) show() { p.Changed.Emit(p.Current()) }

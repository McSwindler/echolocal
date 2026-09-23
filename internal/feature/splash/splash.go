// Package splash is what the panel shows while the device is coming up.
package splash

import (
	"context"
	"image"
	"log/slog"
	"sync"

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

type Splash struct {
	panel *screen.Panel
}

var (
	once   sync.Once
	shared *Splash
)

func Get() *Splash { once.Do(func() { shared = &Splash{} }); return shared }

func (s *Splash) Name() string { return "splash" }

func (s *Splash) Start(context.Context) error {
	p, err := screen.Open(screen.DefaultPath, screen.Orientation(component.Board().PanelRotation))
	if err != nil {
		// A device that cannot draw still answers, so this is said rather than fatal.
		slog.Error("the panel would not open", "err", err)
		return nil
	}
	s.panel = p
	slog.Info("panel", "info", p.Info())

	if err := s.show(); err != nil {
		slog.Error("drawing the splash failed", "err", err)
	}
	return nil
}

// Panel is the screen, or nil where there is none. Whatever draws next takes it from here: the
// mapping and the page being drawn into are per handle, and two handles would fight.
func (s *Splash) Panel() *screen.Panel { return s.panel }

func (s *Splash) Close() error {
	if s.panel == nil {
		return nil
	}
	return s.panel.Close()
}

func (s *Splash) show() error {
	t := theme.Get().Current()
	bg := colour(t.Background)

	s.panel.Fill(bg)
	draw(s.panel, ui.Mark(t.Dark), bg)
	return s.panel.Flip()
}

func colour(c uitheme.Color) screen.Color { return screen.Opaque(c.R, c.G, c.B) }

// draw puts the artwork in the middle of the panel, as large as fits whole. Nearest neighbour: the
// artwork is already close to the size it is shown at.
func draw(p *screen.Panel, img image.Image, bg screen.Color) {
	b := img.Bounds()
	if b.Empty() {
		return
	}

	scale := min(float64(p.Width)/float64(b.Dx()), float64(p.Height)/float64(b.Dy()))
	w := int(float64(b.Dx()) * scale)
	h := int(float64(b.Dy()) * scale)

	ox := (p.Width - w) / 2
	oy := (p.Height - h) / 2

	for y := range h {
		sy := b.Min.Y + y*b.Dy()/h
		for x := range w {
			sx := b.Min.X + x*b.Dx()/w

			c := screen.From(img.At(sx, sy))
			if c.A == 0 {
				continue
			}
			if c.A < 0xFF {
				c = blend(c, bg)
			}
			p.Set(ox+x, oy+y, c)
		}
	}
}

// blend puts a partly transparent pixel over the background. src is premultiplied, as image.RGBA
// stores it, so the source term is already scaled.
func blend(src, dst screen.Color) screen.Color {
	inv := 255 - int(src.A)

	return screen.Color{
		R: byte(int(src.R) + int(dst.R)*inv/255),
		G: byte(int(src.G) + int(dst.G)*inv/255),
		B: byte(int(src.B) + int(dst.B)*inv/255),
		A: 0xFF,
	}
}

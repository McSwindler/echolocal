package ui

import (
	"github.com/ygelfand/echolocal/internal/hardware/screen"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// panel is a screen.Panel as something to draw on. The panel takes colour channels and works in
// viewed coordinates, which is what a surface is.
type panel struct{ p *screen.Panel }

// Of is the panel as a surface.
func Of(p *screen.Panel) Surface { return panel{p: p} }

func (s panel) Size() (w, h int) { return s.p.Width, s.p.Height }

func (s panel) Set(x, y int, c theme.Color) { s.p.Set(x, y, screen.Opaque(c.R, c.G, c.B)) }

func (s panel) FillRect(r Rect, c theme.Color) {
	s.p.FillRect(r.X, r.Y, r.W, r.H, screen.Opaque(c.R, c.G, c.B))
}

func (s panel) At(x, y int) theme.Color {
	c := s.p.At(x, y)
	return theme.Color{R: c.R, G: c.G, B: c.B}
}

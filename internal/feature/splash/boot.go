package splash

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/hardware/screen"
	"github.com/ygelfand/echolocal/internal/ui"
	uitheme "github.com/ygelfand/echolocal/internal/ui/theme"
)

// How the boot screen is laid out, as fractions so it holds on any panel.
const (
	logoShare  = 0.42
	rowShare   = 0.16
	nameShare  = 0.42
	doingShare = 0.30
	dotShare   = 0.22
)

// drawBoot paints the mark beside what the device is still waiting for.
func drawBoot(p *screen.Panel, t uitheme.Theme, progress []component.Progress) {
	bg := colour(t.Background)
	p.Fill(bg)

	logo, list := split(p.Bounds())
	p.Draw(ui.Mark(t.Dark), inset(logo, min(logo.Dx(), logo.Dy())/8), bg)

	if len(progress) == 0 {
		return
	}
	p.Blit(rows(list, t, progress), list.Min, bg)
}

// split is where the mark goes and where the list goes.
func split(in image.Rectangle) (logo, list image.Rectangle) {
	if in.Dx() >= in.Dy() {
		at := in.Min.X + int(float64(in.Dx())*logoShare)
		return image.Rect(in.Min.X, in.Min.Y, at, in.Max.Y), image.Rect(at, in.Min.Y, in.Max.X, in.Max.Y)
	}

	at := in.Min.Y + int(float64(in.Dy())*logoShare)
	return image.Rect(in.Min.X, in.Min.Y, in.Max.X, at), image.Rect(in.Min.X, at, in.Max.X, in.Max.Y)
}

func inset(r image.Rectangle, by int) image.Rectangle {
	return image.Rect(r.Min.X+by, r.Min.Y+by, r.Max.X-by, r.Max.Y-by)
}

// rows renders the list into an image of its own, so the text is drawn once and put on the panel in
// one pass rather than a glyph at a time through the rotation.
func rows(in image.Rectangle, t uitheme.Theme, progress []component.Progress) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, in.Dx(), in.Dy()))
	draw.Draw(out, out.Bounds(), image.NewUniform(rgba(t.Background)), image.Point{}, draw.Src)

	row := int(float64(min(in.Dx(), in.Dy())) * rowShare)
	namePx := float64(row) * nameShare
	doingPx := float64(row) * doingShare

	dot := int(float64(row) * dotShare)
	left := row / 2
	top := (in.Dy() - len(progress)*row) / 2

	gap := row / 12
	nameUp, nameDown := ui.Metrics(namePx)
	doingUp, doingDown := ui.Metrics(doingPx)

	for i, at := range progress {
		y := top + i*row

		mark := t.Muted
		switch {
		case at.Failed:
			mark = t.Warning
		case at.Done:
			mark = t.Accent
		}
		disc(out, left+dot/2, y+row/2, dot/2, rgba(mark))

		x := left + dot + row/3
		saying := at.Doing != "" && !at.Done

		// Both lines are centred as one block, so the second never lands on the first's descenders.
		tall := nameUp + nameDown
		if saying {
			tall += gap + doingUp + doingDown
		}
		base := y + (row-tall)/2 + nameUp

		ui.Line(out, at.Name, namePx, image.Pt(x, base), rgba(t.Text))
		if saying {
			ui.Line(out, at.Doing, doingPx, image.Pt(x, base+nameDown+gap+doingUp), rgba(t.Muted))
		}
	}
	return out
}

// disc is the state mark beside a row.
func disc(dst *image.RGBA, cx, cy, r int, c color.Color) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				dst.Set(cx+x, cy+y, c)
			}
		}
	}
}

func rgba(c uitheme.Color) color.RGBA { return color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xFF} }

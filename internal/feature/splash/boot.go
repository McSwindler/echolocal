package splash

import (
	"image"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/reveal"
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

// drawBoot leaves the mark's half clear for the reveal under it, and lists what the device is still
// waiting for beside it.
func drawBoot(p *display.Panel, t uitheme.Theme, progress []component.Progress) {
	s := ui.Of(p)
	w, h := s.Size()

	logo, list := reveal.Split(w, h)
	ui.Clear(s, logo)
	ui.FillRect(s, list, t.Background)

	if len(progress) == 0 {
		return
	}
	rows(s, list, t, progress)
}

func bounds(p *display.Panel) image.Rectangle { return image.Rect(0, 0, p.Width, p.Height) }

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

// box is a rectangle as the canvas wants it.
func box(r image.Rectangle) ui.Rect {
	return ui.Rect{X: r.Min.X, Y: r.Min.Y, W: r.Dx(), H: r.Dy()}
}

// rows draws the list, each name with what it is waiting for under it.
func rows(s ui.Surface, in ui.Rect, t uitheme.Theme, progress []component.Progress) {
	row := int(float64(min(in.W, in.H)) * rowShare)

	name := ui.MustLoad(ui.Medium, int(float64(row)*nameShare))
	doing := ui.MustLoad(ui.Regular, int(float64(row)*doingShare))

	dot := int(float64(row) * dotShare)
	left := in.X + row/2
	top := in.Y + (in.H-len(progress)*row)/2
	gap := row / 12

	for i, at := range progress {
		y := top + i*row

		mark := t.Muted
		switch {
		case at.Failed:
			mark = t.Warning
		case at.Done:
			mark = t.Accent
		}
		ui.FillRounded(s, ui.Rect{X: left, Y: y + (row-dot)/2, W: dot, H: dot}, dot/2, mark)

		x := left + dot + row/3
		saying := at.Doing != ""

		// Both lines are placed as one block, so the second never lands on the first's descenders.
		_, nameH := name.Measure(at.Name)
		_, doingH := doing.Measure(at.Doing)

		tall := nameH
		if saying {
			tall += gap + doingH
		}
		up := y + (row-tall)/2

		ui.DrawText(s, name, x, up, t.Text, t.Background, at.Name)
		if saying {
			ui.DrawText(s, doing, x, up+nameH+gap, t.Muted, t.Background, at.Doing)
		}
	}
}

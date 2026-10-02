package splash

import (
	"image"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/reveal"
	uitheme "github.com/ygelfand/echolocal/internal/ui/theme"
)

// How the boot screen is laid out, as fractions so it holds on any panel.
const (
	logoShare  = 0.42
	rowShare   = 0.11
	nameShare  = 0.42
	doingShare = 0.30
	dotShare   = 0.22
	skipShare  = 0.09
	padShare   = 0.048
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
	if !settled(progress) {
		drawSkip(s, t)
	}
}

func settled(progress []component.Progress) bool {
	for _, p := range progress {
		if !p.Settled() {
			return false
		}
	}
	return true
}

func skipBox(w, h int) ui.Rect {
	short := min(w, h)
	high := int(float64(short) * skipShare)
	wide := high * 3
	pad := int(float64(short) * padShare)
	logo, _ := reveal.Split(w, h)
	return ui.Rect{X: logo.X + (logo.W-wide)/2, Y: logo.Y + logo.H - high - pad, W: wide, H: high}
}

func drawSkip(s ui.Surface, t uitheme.Theme) {
	w, h := s.Size()
	at := skipBox(w, h)
	ui.FillRounded(s, at, at.H/2, t.Surface)
	font := ui.MustLoad(ui.Medium, at.H*2/5)
	label := say.T("boot.skip")
	tw, th := font.Measure(label)
	ui.DrawText(s, font, at.X+(at.W-tw)/2, at.Y+(at.H-th)/2, t.Text, t.Surface, label)
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
	cols := 1
	if len(progress)*row > in.H {
		cols = 2
	}
	per := (len(progress) + cols - 1) / cols
	row = min(row, in.H/per)
	colW := in.W / cols

	name := ui.MustLoad(ui.Medium, int(float64(row)*nameShare))
	doing := ui.MustLoad(ui.Regular, int(float64(row)*doingShare))

	dot := int(float64(row) * dotShare)
	top := in.Y + (in.H-per*row)/2
	gap := row / 12

	for i, at := range progress {
		left := in.X + (i/per)*colW + row/2
		y := top + (i%per)*row

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

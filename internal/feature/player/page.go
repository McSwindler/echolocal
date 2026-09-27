package player

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/feature/volume"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

// Page is what is playing. One instance, so the shell can recognise it.
func Page() shell.View {
	pageOnce.Do(func() { page = &screen{} })
	return page
}

var (
	pageOnce sync.Once
	page     *screen
)

// Fractions of the shorter side.
const (
	coverShare = 0.62

	buttonShare  = 0.115
	secondShare  = 0.66
	betweenShare = 0.06

	// pressedShare is how far a button shifts towards the text colour under a finger.
	pressedShare = 0.28
)

// titleLines and creditLines are how far a title and the line under it run before they are cut.
const (
	titleLines  = 2
	creditLines = 2
)

type screen struct {
	shell.Touches

	metrics widget.Metrics

	// Where things landed when this was last drawn, so a touch resolves against the picture that
	// was touched.
	level ui.Rect
	back  ui.Rect

	// taps is what the drawing put down, handed to Touches once at the end.
	taps []shell.Spot

	holding bool
}

func (v *screen) Covers() bool { return true }

// Shows keeps the volume card off this screen, which carries the music level on a slider of its own.
func (v *screen) Shows(s config.Stream) bool { return s == config.StreamMedia }

// Wants puts the volume buttons on the music while this is up.
func (v *screen) Wants() (config.Stream, bool) { return config.StreamMedia, true }

// spot records something touchable and answers where it sits, which is the index Pressed takes.
func (v *screen) spot(at ui.Rect, do func()) int {
	v.taps = append(v.taps, shell.Spot{At: at, Do: do})
	return len(v.taps) - 1
}

func (v *screen) Draw(s ui.Surface, palette theme.Theme) {
	w, h := s.Size()
	m := widget.New(w, h)
	v.metrics = m
	v.taps = nil

	ui.Fill(s, palette.Background)

	whole := ui.Rect{W: w, H: h}
	v.back = m.Header(whole)

	arrow := m.Arrow(whole)
	back := ui.Rect{X: arrow.X + (arrow.W-m.Icon)/2, Y: arrow.Y + (arrow.H-m.Icon)/2, W: m.Icon, H: m.Icon}
	ui.DrawIcon(s, icons.NavigationArrowBack, back, palette.Muted, palette.Background)
	v.drawChip(s, arrow, whole, m, palette)

	now := media.Get().Now()
	body := v.drawFoot(s, m.Body(whole).Inset(m.Pad), m, now, palette)

	// The artwork beside the words, which is what a wide short panel has room for.
	side := min(body.H, int(float64(m.Unit)*coverShare))
	cover := ui.Rect{X: body.X, Y: body.Y + (body.H-side)/2, W: side, H: side}
	v.drawCover(s, cover, now, palette)

	words := ui.Rect{X: cover.X + side + m.Pad*2, Y: body.Y, W: body.X + body.W - cover.X - side - m.Pad*2}
	said := says(m, now, words.W)
	words.Y, words.H = body.Y+(body.H-said.height())/2, said.height()
	said.draw(s, words, palette)

	v.Spots(v.taps)
}

// drawChip says where the audio came from, opposite the way back.
func (v *screen) drawChip(s ui.Surface, arrow, whole ui.Rect, m widget.Metrics, palette theme.Theme) {
	side := arrow.H * 5 / 8
	right := whole.W - arrow.W - (arrow.X - whole.X) + (arrow.W+side)/2

	at := ui.Rect{X: right - side, Y: arrow.Y + (arrow.H-side)/2, W: side, H: side}

	named := ui.Fit(m.Hint, media.Get().Named(), whole.W/3)
	if named != "" {
		w, _ := m.Hint.Measure(named)
		at.X -= w + m.Pad/2
		at.W += w + m.Pad/2
	}

	ui.FillRounded(s, at, side/2, palette.Surface)

	mark := ui.Rect{X: at.X, Y: at.Y, W: side, H: side}
	drawFrom(s, media.Get().From(), mark.Inset(side/4), palette)

	if named != "" {
		_, h := m.Hint.Measure(named)
		ui.DrawText(s, m.Hint, mark.X+side, at.Y+(side-h)/2, palette.Text, palette.Surface, named)
	}
}

// drawFrom is the mark for where the audio came from.
func drawFrom(s ui.Surface, from media.Kind, at ui.Rect, palette theme.Theme) {
	ui.DrawIcon(s, fromIcon(from), at, palette.Text, palette.Surface)
}

func fromIcon(from media.Kind) ui.Icon {
	switch from {
	case media.FromBluetooth:
		return icons.DeviceBluetooth
	case media.FromGroup:
		return icons.HardwareSpeakerGroup
	}
	return icons.ActionHome
}

// drawFoot is the progress, the transport and the level, and answers what is left above them.
func (v *screen) drawFoot(s ui.Surface, body ui.Rect, m widget.Metrics, now media.Now, palette theme.Theme) ui.Rect {
	level := ui.Rect{X: body.X, Y: body.Y + body.H - m.Row, W: body.W, H: m.Row}
	v.level = level

	music := volume.Get().Level(config.StreamMedia)
	m.Draw(s, level, widget.Row{Kind: widget.Slider, Level: music, Icon: volume.Mark(config.StreamMedia)},
		palette, palette.Background)

	button := int(float64(m.Unit) * buttonShare)
	transport := ui.Rect{X: body.X, Y: level.Y - m.Pad - button, W: body.W, H: button}
	v.drawTransport(s, transport, now, palette)

	top := transport.Y
	if now.Length > 0 {
		bar := ui.Rect{X: body.X, Y: transport.Y - m.Pad - marked(m), W: body.W, H: marked(m)}
		v.drawProgress(s, bar, m, now, palette)
		top = bar.Y
	}

	return ui.Rect{X: body.X, Y: body.Y, W: body.W, H: top - m.Pad - body.Y}
}

// marked is how tall the bar and the two times under it are together.
func marked(m widget.Metrics) int {
	_, h := m.Hint.Measure("0:00")
	return thick(m) + m.Pad/2 + h
}

// thick is how heavy the progress bar is drawn.
func thick(m widget.Metrics) int { return max(2, m.Pad/3) }

// drawProgress is how far through the track is, with the times under either end.
func (v *screen) drawProgress(s ui.Surface, at ui.Rect, m widget.Metrics, now media.Now, palette theme.Theme) {
	if at.W <= 0 || now.Length <= 0 {
		return
	}

	t := thick(m)
	ui.FillRounded(s, ui.Rect{X: at.X, Y: at.Y, W: at.W, H: t}, t/2,
		palette.Surface.Blend(palette.Text, 0.18))

	done := min(max(int(int64(at.W)*int64(now.Elapsed)/int64(now.Length)), 0), at.W)
	if done > 0 {
		ui.FillRounded(s, ui.Rect{X: at.X, Y: at.Y, W: done, H: t}, t/2, palette.Accent)
	}

	knob := t * 3
	ui.FillRounded(s, ui.Rect{
		X: at.X + min(max(done-knob/2, 0), at.W-knob), Y: at.Y + t/2 - knob/2, W: knob, H: knob,
	}, knob/2, palette.Accent)

	f := m.Hint
	_, h := f.Measure("0:00")
	line := ui.Rect{X: at.X, Y: at.Y + t + m.Pad/2, W: at.W, H: h}

	ui.DrawText(s, f, line.X, line.Y, palette.Muted, palette.Background, clock(now.Elapsed))
	ui.DrawTextRight(s, f, line, palette.Muted, palette.Background, clock(now.Length))
}

// clock is a duration the way a player shows one.
func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	whole := int(d / time.Second)
	if h := whole / 3600; h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, whole/60%60, whole%60)
	}
	return fmt.Sprintf("%d:%02d", whole/60, whole%60)
}

// drawCover is the album art, or a mark holding its place where none came.
func (v *screen) drawCover(s ui.Surface, at ui.Rect, now media.Now, palette theme.Theme) {
	if at.W <= 0 || at.H <= 0 {
		return
	}

	if art := cover(now); art != nil {
		ui.DrawScaled(s, art, at)
		return
	}

	ui.FillRounded(s, at, at.W/12, palette.Surface)
	ui.DrawIcon(s, icons.ImageMusicNote, at.Inset(at.W/3), palette.Muted, palette.Surface)
}

type control struct {
	icon    ui.Icon
	do      func()
	need    media.Controls
	primary bool
}

// drawTransport offers what the source answers to, grouped in the middle.
func (v *screen) drawTransport(s ui.Surface, at ui.Rect, now media.Now, palette theme.Theme) {
	if at.H <= 0 {
		return
	}

	p := media.Get()

	// One button for both: what it shows is what pressing it would do.
	middle := control{icon: icons.AVPlayArrow, do: p.Unpause, primary: true}
	if now.Playing && !now.Paused {
		middle = control{icon: icons.AVPause, do: p.Pause, need: media.CanPause, primary: true}
	}

	shown := make([]control, 0, 4)
	for _, c := range []control{
		{icon: icons.AVSkipPrevious, do: p.Previous, need: media.CanPrevious},
		middle,
		{icon: icons.AVStop, do: p.Stop, need: media.CanStop},
		{icon: icons.AVSkipNext, do: p.Next, need: media.CanNext},
	} {
		if c.need == 0 || now.Can.Has(c.need) {
			shown = append(shown, c)
		}
	}
	if len(shown) == 0 {
		return
	}

	second := int(float64(at.H) * secondShare)
	between := int(float64(at.H) * betweenShare)

	width := -between
	for _, c := range shown {
		width += side(c.primary, at.H, second) + between
	}

	x := at.X + (at.W-width)/2
	for _, c := range shown {
		w := side(c.primary, at.H, second)
		box := ui.Rect{X: x, Y: at.Y + (at.H-w)/2, W: w, H: w}

		fill, mark := palette.Surface, palette.Text
		if c.primary {
			fill, mark = palette.Accent, palette.Background
		}
		if v.Pressed(v.spot(box, c.do)) {
			fill = fill.Blend(palette.Text, pressedShare)
		}

		ui.FillRounded(s, box, box.W/2, fill)
		ui.DrawIcon(s, c.icon, box.Inset(box.W/4), mark, fill)

		x += w + between
	}
}

func side(primary bool, full, second int) int {
	if primary {
		return full
	}
	return second
}

// Tap. The header goes back, and a touch that landed on nothing stays here.
func (v *screen) Tap(x, y int) bool {
	if v.Tapped(x, y) {
		return true
	}
	if v.back.Contains(x, y) {
		shell.Get().Pop()
		return true
	}
	return true
}

// Grab takes a finger going down on the level, which is the one thing here that is pulled.
func (v *screen) Grab(x, y int) bool {
	v.holding = v.metrics.Track(v.level).Contains(x, y) || v.level.Contains(x, y)
	if !v.holding {
		return false
	}

	v.Dragging(v.level)
	v.Drag(x, y)
	return true
}

// Drag sets the music level the finger is at.
func (v *screen) Drag(x, _ int) bool {
	if !v.holding {
		return false
	}

	want := v.metrics.Level(v.level, x)
	if want == volume.Get().Level(config.StreamMedia) {
		return false
	}

	volume.Get().Set(config.StreamMedia, want)
	return true
}

// Damaged is the level's row while it is being pulled.
func (v *screen) Damaged() ui.Rect {
	if v.holding {
		return v.level
	}
	return v.Touches.Damaged()
}

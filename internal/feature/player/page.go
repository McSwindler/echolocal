package player

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/feature/volume"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

// Page is what is playing, with the transport for it. One instance, so the shell can recognize it.
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
	coverShare  = 0.58
	listedCover = 0.34

	buttonShare  = 0.105
	secondShare  = 0.66
	betweenShare = 0.055

	pressedShare = 0.28
)

type screen struct {
	shell.Touches

	metrics widget.Metrics

	volume ui.Rect
	back   ui.Rect

	seekBar   ui.Rect
	length    time.Duration
	scrubbed  bool
	scrubAt   time.Duration
	landUntil time.Time

	taps []shell.Spot

	holding bool

	clockMu sync.Mutex
	bar     ui.Rect
	shown   time.Duration
}

func (v *screen) clocked(bar ui.Rect, elapsed time.Duration) {
	v.clockMu.Lock()
	v.bar, v.shown = bar, elapsed.Truncate(time.Second)
	v.clockMu.Unlock()
}

func (v *screen) stale(elapsed time.Duration) (ui.Rect, bool) {
	v.clockMu.Lock()
	defer v.clockMu.Unlock()
	if v.bar.W <= 0 || elapsed.Truncate(time.Second) == v.shown {
		return ui.Rect{}, false
	}
	return v.bar, true
}

func (v *screen) spot(at ui.Rect, do func()) int {
	v.taps = append(v.taps, shell.Spot{At: at, Do: do})
	return len(v.taps) - 1
}

func (v *screen) Covers() bool { return true }

// Shows keeps the volume card off this screen, which carries the same level on a slider of its own.
func (v *screen) Shows(s config.Stream) bool { return s == config.StreamMedia }

// Wants puts the volume buttons on the music while this is up.
func (v *screen) Wants() (config.Stream, bool) { return config.StreamMedia, true }

func (v *screen) Draw(s ui.Surface, palette theme.Theme) {
	w, h := s.Size()
	m := widget.New(w, h)
	v.metrics = m
	v.taps = nil

	ui.Fill(s, palette.Background)

	whole := ui.Rect{W: w, H: h}
	v.back = m.Header(whole)

	arrow := m.Arrow(whole)
	m.Back(s, arrow, palette.Text)
	v.drawChip(s, arrow, whole, m, palette)

	now := media.Get().Now()
	body := v.drawFoot(s, m.Body(whole).Inset(m.Pad*2), m, now, palette)

	list := len(now.Queue) > 0
	left := body
	if list {
		left.W = body.W/2 - m.Pad
	}

	said := says(m, now, left.W)

	if !list {
		cover, at := stack(left, m, said.height(), coverShare)
		v.drawCover(s, cover, now, palette)
		said.draw(s, at, palette)
	} else {
		cover, at := stack(left, m, said.height(), listedCover)
		v.drawCover(s, cover, now, palette)
		said.draw(s, at, palette)

		queue := ui.Rect{X: left.X + left.W + m.Pad*2, Y: body.Y}
		queue.W, queue.H = body.X+body.W-queue.X, body.H
		v.drawQueue(s, queue, m, now, palette)
	}

	v.Spots(v.taps)
}

// drawChip says where the audio came from, opposite the way back.
func (v *screen) drawChip(s ui.Surface, arrow, whole ui.Rect, m widget.Metrics, palette theme.Theme) {
	side := arrow.H * 5 / 8
	right := whole.W - arrow.W - (arrow.X - whole.X) + (arrow.W+side)/2

	at := ui.Rect{X: right - side, Y: arrow.Y + (arrow.H-side)/2, W: side, H: side}

	named := fit(m.Hint, media.Get().Named(), whole.W/3)
	if named != "" {
		w, _ := m.Hint.Measure(named)
		at.X -= w + m.Pad/2
		at.W += w + m.Pad/2
	}

	ui.FillRounded(s, at, side/2, palette.Surface)

	mark := ui.Rect{X: at.X, Y: at.Y, W: side, H: side}
	ui.DrawIcon(s, kindIcon(media.Get().From()), mark.Inset(side/4), palette.Text, palette.Surface)

	if named != "" {
		_, h := m.Hint.Measure(named)
		ui.DrawText(s, m.Hint, mark.X+side, at.Y+(side-h)/2, palette.Text, palette.Surface, named)
	}
}

// stack is the artwork with the words under it, no larger than the share of the panel it is allowed.
func stack(in ui.Rect, m widget.Metrics, h int, most float64) (cover, words ui.Rect) {
	side := min(in.W, in.H-h-m.Pad, int(float64(m.Unit)*most))

	cover = ui.Rect{X: in.X + (in.W-side)/2, Y: in.Y + (in.H-h-m.Pad-side)/2, W: side, H: side}
	words = ui.Rect{X: in.X, Y: cover.Y + side + m.Pad, W: in.W, H: h}
	return cover, words
}

// drawFoot is the progress, the transport and the level, and answers what is left above them.
func (v *screen) drawFoot(s ui.Surface, body ui.Rect, m widget.Metrics, now media.Now, palette theme.Theme) ui.Rect {
	level := ui.Rect{X: body.X, Y: body.Y + body.H - m.Row, W: body.W, H: m.Row}
	v.volume = level
	m.Draw(s, level, widget.Row{
		Kind:  widget.Slider,
		Level: volume.Get().Level(config.StreamMedia),
		Icon:  volume.Mark(config.StreamMedia),
	}, palette, palette.Background)

	button := int(float64(m.Unit) * buttonShare)
	transport := ui.Rect{X: body.X, Y: level.Y - m.Pad - button, W: body.W, H: button}
	v.drawTransport(s, transport, now, palette)

	top := transport.Y
	v.clocked(ui.Rect{}, 0)
	if now.Length > 0 {
		bar := ui.Rect{X: body.X, Y: transport.Y - m.Pad - marked(m), W: body.W, H: marked(m)}
		v.drawProgress(s, bar, m, now, palette)
		top = bar.Y
	}

	return ui.Rect{X: body.X, Y: body.Y, W: body.W, H: top - m.Pad*2 - body.Y}
}

func marked(m widget.Metrics) int {
	_, h := m.Hint.Measure("0:00")
	return thick(m) + m.Pad/2 + h
}

func thick(m widget.Metrics) int { return max(2, m.Pad/3) }

func (v *screen) drawProgress(s ui.Surface, at ui.Rect, m widget.Metrics, now media.Now, palette theme.Theme) {
	if at.W <= 0 || now.Length <= 0 {
		v.length = 0
		return
	}

	t := thick(m)
	v.seekBar, v.length = ui.Rect{X: at.X, Y: at.Y - t*3, W: at.W, H: t * 7}, now.Length
	switch {
	case v.scrubbed:
		now.Elapsed = v.scrubAt
	case time.Now().Before(v.landUntil):
		if d := now.Elapsed - v.scrubAt; d > landed || d < -landed {
			now.Elapsed = v.scrubAt
		} else {
			v.landUntil = time.Time{}
		}
	}
	ui.FillRounded(s, ui.Rect{X: at.X, Y: at.Y, W: at.W, H: t}, t/2,
		palette.Surface.Blend(palette.Text, 0.18))

	done := min(max(int(int64(at.W)*int64(now.Elapsed)/int64(now.Length)), 0), at.W)
	if done > 0 {
		ui.FillRounded(s, ui.Rect{X: at.X, Y: at.Y, W: done, H: t}, t/2, palette.Accent)
	}

	knob := t * 3
	v.clocked(ui.Rect{X: at.X, Y: at.Y + t/2 - knob/2, W: at.W, H: at.H - t/2 + knob/2}, now.Elapsed)
	ui.FillRounded(s, ui.Rect{
		X: at.X + min(max(done-knob/2, 0), at.W-knob), Y: at.Y + t/2 - knob/2, W: knob, H: knob,
	}, knob/2, palette.Accent)

	f := m.Hint
	_, h := f.Measure("0:00")
	line := ui.Rect{X: at.X, Y: at.Y + t + m.Pad/2, W: at.W, H: h}

	if now.LiveWithin > 0 {
		v.drawLive(s, f, line, m, now, palette)
		return
	}
	ui.DrawText(s, f, line.X, line.Y, palette.Muted, palette.Background, clock(now.Elapsed))
	ui.DrawTextRight(s, f, line, palette.Muted, palette.Background, clock(now.Length))
}

func (v *screen) drawLive(s ui.Surface, f *ui.Font, line ui.Rect, m widget.Metrics, now media.Now, palette theme.Theme) {
	label, fill := say.T("player.live"), palette.Danger
	if behind := now.Length - now.Elapsed; behind > now.LiveWithin {
		fill = palette.Muted
		ui.DrawText(s, f, line.X, line.Y, palette.Muted, palette.Background, "-"+clock(behind))
	}
	pill := livePill(f, line, m.Pad/2)
	ui.FillRounded(s, pill, pill.H/2, fill)
	ui.DrawText(s, f, pill.X+pill.H/2, line.Y, theme.Color{R: 0xff, G: 0xff, B: 0xff}, fill, label)
	v.spot(pill.Inset(-m.Pad/2), v.goLive)
}

func livePill(f *ui.Font, line ui.Rect, pad int) ui.Rect {
	w, h := f.Measure(say.T("player.live"))
	ph := h + 2*pad
	return ui.Rect{X: line.X + line.W - w - ph, Y: line.Y - pad, W: w + ph, H: ph}
}

func (v *screen) goLive() {
	if sk := seeker(); sk != nil && v.length > 0 {
		v.scrubAt, v.landUntil = v.length, time.Now().Add(landWait)
		sk.Seek(v.length)
	}
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

type lines struct {
	head, under *ui.Font
	gap         int

	title   []string
	credits []string
}

const (
	titleLines  = 2
	creditLines = 2
)

func says(m widget.Metrics, now media.Now, width int) lines {
	return lines{
		head:    m.Title,
		under:   m.Value,
		gap:     m.Pad / 2,
		title:   wrap(m.Title, heading(now), width, titleLines),
		credits: credits(m.Value, now, width),
	}
}

func (l lines) height() int {
	_, th := l.head.Measure("Ag")
	_, uh := l.under.Measure("Ag")
	return len(l.title)*th + l.gap + len(l.credits)*uh
}

func (l lines) draw(s ui.Surface, at ui.Rect, palette theme.Theme) {
	if at.W <= 0 || at.H <= 0 {
		return
	}

	_, th := l.head.Measure("Ag")
	_, uh := l.under.Measure("Ag")

	y := at.Y
	for _, said := range l.title {
		ui.DrawTextIn(s, l.head, ui.Rect{X: at.X, Y: y, W: at.W, H: th},
			palette.Text, palette.Background, said)
		y += th
	}
	y += l.gap

	for _, said := range l.credits {
		ui.DrawTextIn(s, l.under, ui.Rect{X: at.X, Y: y, W: at.W, H: uh},
			palette.Text.Blend(palette.Muted, 0.4), palette.Background, said)
		y += uh
	}
}

func credits(f *ui.Font, now media.Now, width int) []string {
	switch {
	case now.Artist == "" && now.Album == "":
		return nil
	case now.Artist == "":
		return wrap(f, now.Album, width, creditLines)
	case now.Album == "":
		return wrap(f, now.Artist, width, creditLines)
	}

	both := now.Artist + " · " + now.Album
	if w, _ := f.Measure(both); w <= width {
		return []string{both}
	}
	return []string{fit(f, now.Artist, width), fit(f, now.Album, width)}
}

// drawQueue is what is playing and what follows it, each under a rule of its own.
func (v *screen) drawQueue(s ui.Surface, at ui.Rect, m widget.Metrics, now media.Now, palette theme.Theme) {
	if at.W <= 0 || at.H <= 0 {
		return
	}

	head, title, under := m.Hint, m.Label, m.Hint

	_, hh := head.Measure("Ag")
	_, th := title.Measure("Ag")
	_, uh := under.Measure("Ag")

	row := th + uh
	gap := m.Pad / 2
	bottom := at.Y + at.H

	y := at.Y
	if y+hh+gap+row > bottom {
		return
	}

	thumbs := len(now.Art) > 0
	for _, t := range now.Queue {
		thumbs = thumbs || len(t.Art) > 0
	}

	y = rule(s, ui.Rect{X: at.X, Y: y, W: at.W, H: hh}, head, m, say.T("player.now"), "", palette) + gap
	y = entry(s, ui.Rect{X: at.X, Y: y, W: at.W, H: row}, m, title, under,
		media.Track{Title: heading(now), Artist: now.Artist, Length: now.Length, Art: now.Art, ArtID: now.ArtID},
		thumbs, palette.Text, palette) + m.Pad

	if y+hh+gap+row > bottom {
		return
	}
	y = rule(s, ui.Rect{X: at.X, Y: y, W: at.W, H: hh}, head, m,
		say.T("player.next"), strconv.Itoa(len(now.Queue)), palette) + gap

	fits := (bottom - y + gap) / (row + gap)
	for i, t := range now.Queue {
		if i >= fits {
			return
		}

		fg := palette.Text
		if i == fits-1 && len(now.Queue) > fits {
			fg = palette.Text.Blend(palette.Background, 0.6)
		}

		y = entry(s, ui.Rect{X: at.X, Y: y, W: at.W, H: row}, m, title, under, t, thumbs, fg, palette) +
			gap
	}
}

func rule(s ui.Surface, at ui.Rect, f *ui.Font, m widget.Metrics, says, count string, palette theme.Theme) int {
	label := strings.ToUpper(says)

	w, _ := f.Measure(label)
	ui.DrawText(s, f, at.X, at.Y, palette.Muted, palette.Background, label)
	x := at.X + w + m.Pad/2

	if count != "" {
		cw, _ := f.Measure(count)
		ui.DrawText(s, f, x, at.Y, palette.Muted.Blend(palette.Background, 0.3), palette.Background, count)
		x += cw + m.Pad/2
	}

	if hair := at.X + at.W - x; hair > 0 {
		ui.FillRect(s, ui.Rect{X: x, Y: at.Y + at.H/2, W: hair, H: 1},
			palette.Muted.Blend(palette.Background, 0.6))
	}
	return at.Y + at.H
}

func entry(s ui.Surface, at ui.Rect, m widget.Metrics, title, under *ui.Font, t media.Track, thumbs bool, fg theme.Color, palette theme.Theme) int {
	quiet := fg.Blend(palette.Background, 0.4)

	if thumbs {
		side := at.H
		if art := ui.Picture(t.Art, t.ArtID); art != nil {
			ui.DrawScaled(s, art, ui.Rect{X: at.X, Y: at.Y, W: side, H: side})
		}

		at.X += side + m.Pad/2
		at.W -= side + m.Pad/2
	}

	width := at.W
	if t.Length > 0 {
		said := clock(t.Length)
		w, h := under.Measure(said)

		ui.DrawTextRight(s, under, ui.Rect{X: at.X, Y: at.Y, W: at.W, H: h},
			quiet, palette.Background, said)
		width = at.W - w - m.Pad
	}

	_, th := title.Measure("Ag")
	ui.DrawText(s, title, at.X, at.Y, fg, palette.Background, fit(title, t.Title, width))

	if t.Artist != "" {
		ui.DrawText(s, under, at.X, at.Y+th, quiet, palette.Background, fit(under, t.Artist, width))
	}
	return at.Y + at.H
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

	press := func(do func(transporter)) func() {
		return func() { do(transport()) }
	}

	middle := control{icon: icons.AVPlayArrow, do: press(transporter.Play), primary: true}
	if now.Playing && !now.Paused {
		middle = control{icon: icons.AVPause, do: press(transporter.Pause), need: media.CanPause, primary: true}
	}

	shown := make([]control, 0, 4)
	for _, c := range []control{
		{icon: icons.AVSkipPrevious, do: press(transporter.Previous), need: media.CanPrevious},
		middle,
		{icon: icons.AVStop, do: press(transporter.Stop), need: media.CanStop},
		{icon: icons.AVSkipNext, do: press(transporter.Next), need: media.CanNext},
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

// wrap breaks a line at spaces to the width it has, into at most so many lines.
func wrap(f *ui.Font, says string, width, most int) []string {
	if says == "" {
		return nil
	}
	if w, _ := f.Measure(says); w <= width {
		return []string{says}
	}

	out := make([]string, 0, most)
	rest := says

	for len(out) < most-1 {
		cut := breaks(f, rest, width)
		if cut <= 0 {
			break
		}

		out = append(out, rest[:cut])
		rest = strings.TrimLeft(rest[cut:], " ")

		if w, _ := f.Measure(rest); w <= width {
			break
		}
	}
	return append(out, fit(f, rest, width))
}

func breaks(f *ui.Font, says string, width int) int {
	last := 0
	for i, r := range says {
		if r != ' ' {
			continue
		}
		if w, _ := f.Measure(says[:i]); w > width {
			break
		}
		last = i
	}
	return last
}

// fit trims a line to the width it has, ending in an ellipsis.
func fit(f *ui.Font, says string, width int) string {
	if w, _ := f.Measure(says); w <= width {
		return says
	}

	runes := []rune(says)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		if w, _ := f.Measure(string(runes) + "…"); w <= width {
			return string(runes) + "…"
		}
	}
	return "…"
}

func heading(now media.Now) string {
	switch {
	case now.Title != "":
		return now.Title
	case now.Paused:
		return say.T("player.paused")
	case now.Playing:
		return say.T("player.playing")
	}
	return say.T("player.idle")
}

func (v *screen) Tap(x, y int) bool {
	if v.Tapped(x, y) {
		shell.Get().Redraw()
		return true
	}

	if v.back.Contains(x, y) {
		shell.Get().Pop()
	}
	return true
}

func (v *screen) Grab(x, y int) bool {
	if sk := seeker(); sk != nil && v.length > 0 && v.seekBar.Contains(x, y) {
		v.scrubbed, v.scrubAt = true, v.seekAt(x)
		return true
	}
	v.holding = v.volume.Contains(x, y)
	if v.holding {
		v.Dragging(v.volume)
	}
	return v.holding
}

func (v *screen) seekAt(x int) time.Duration {
	return time.Duration(int64(v.length) * int64(min(max(x-v.seekBar.X, 0), v.seekBar.W)) / int64(max(v.seekBar.W, 1)))
}

func (v *screen) Let(x, _ int) bool {
	if !v.scrubbed {
		return false
	}
	v.scrubbed = false
	v.scrubAt, v.landUntil = v.seekAt(x), time.Now().Add(landWait)
	if sk := seeker(); sk != nil {
		sk.Seek(v.scrubAt)
	}
	return true
}

const (
	landed   = 1500 * time.Millisecond
	landWait = 3 * time.Second
)

func (v *screen) Drag(x, _ int) bool {
	if v.scrubbed {
		at := v.seekAt(x)
		moved := at/time.Second != v.scrubAt/time.Second
		v.scrubAt = at
		return moved
	}
	if !v.holding {
		return false
	}

	want := v.metrics.Level(v.volume, x)
	if want == volume.Get().Level(config.StreamMedia) {
		return false
	}
	volume.Get().Set(config.StreamMedia, want)
	return true
}

// transporter is what the buttons reach.
type transporter interface {
	Play()
	Pause()
	Stop()
	Next()
	Previous()
}

// transport is whoever holds the card, or the local queue before anything else has played.
func transport() transporter {
	h := media.Get().Holder()
	if h == nil {
		return local{}
	}
	if t, ok := h.(transporter); ok {
		return t
	}
	return plain{h}
}

func seeker() media.Seeker {
	if sk, ok := media.Get().Holder().(media.Seeker); ok && sk.CanSeek() {
		return sk
	}
	return nil
}

type plain struct{ media.Source }

func (plain) Next()     {}
func (plain) Previous() {}

// local is a url Home Assistant played at this device.
type local struct{}

func (local) Play()     { media.Get().Unpause() }
func (local) Pause()    { media.Get().Pause() }
func (local) Stop()     { media.Get().Stop() }
func (local) Next()     {}
func (local) Previous() {}

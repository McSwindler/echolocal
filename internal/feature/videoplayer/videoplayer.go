package videoplayer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	apptheme "github.com/ygelfand/echolocal/internal/feature/theme"
	"github.com/ygelfand/echolocal/internal/feature/volume"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/hardware/touch"
	"github.com/ygelfand/echolocal/internal/hardware/video"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

type Controls interface {
	media.Source
	media.Details
	Seek(to time.Duration)
	CanSeek() bool
}

type Mark struct {
	From, To time.Duration
	Color    theme.Color
}

type Marked interface {
	Marks() []Mark
}

const (
	linger   = 4 * time.Second
	skip     = 10 * time.Second
	refresh  = 250 * time.Millisecond
	bandPart = 36
)

type hit struct {
	at    ui.Rect
	do    func(x int)
	seek  func(x int) time.Duration
	level func(x int) int
}

type Overlay struct {
	c Controls

	mu      sync.Mutex
	shown   bool
	paused  bool
	until   time.Time
	hits    []hit
	drawn   string
	band    video.Band
	have    bool
	bufs    [2][]byte
	next    int
	changed chan struct{}

	orientation func() display.Orientation
	palette     func() theme.Theme
	native      func() (w, h int)
}

func New(c Controls) *Overlay {
	return &Overlay{
		c:           c,
		changed:     make(chan struct{}, 1),
		orientation: func() display.Orientation { return display.Get().Orientation() },
		palette:     func() theme.Theme { return apptheme.Get().Current() },
		native:      func() (int, int) { return display.Get().Native() },
	}
}

func (o *Overlay) Changed() <-chan struct{} { return o.changed }

func (o *Overlay) Band(display.Orientation, int, int) (video.Band, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.band, o.shown && o.have
}

func (o *Overlay) Run(ctx context.Context) {
	stop := touch.Gestures.Listen(func(g touch.Gesture) {
		if g.Kind == touch.Tap {
			o.Tap(g.EndX, g.EndY)
		}
	})
	defer stop()
	defer o.release()

	t := time.NewTicker(refresh)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			o.tick()
		}
	}
}

func (o *Overlay) Tap(x, y int) {
	o.mu.Lock()
	if !o.shown {
		o.shown, o.until = true, time.Now().Add(linger)
		o.mu.Unlock()
		o.render(true)
		return
	}
	var do func(int)
	for _, h := range o.hits {
		if h.at.Contains(x, y) {
			do = h.do
			break
		}
	}
	if do == nil {
		o.shown = false
		o.mu.Unlock()
		o.signal()
		return
	}
	o.until = time.Now().Add(linger)
	o.mu.Unlock()
	do(x)
	o.render(true)
}

func (o *Overlay) tick() {
	now := o.c.Now()
	o.mu.Lock()
	shown := o.shown
	if shown && now.Playing && !now.Paused && time.Now().After(o.until) {
		o.shown, shown = false, false
		o.mu.Unlock()
		o.signal()
		return
	}
	appear := !shown && now.Paused && !o.paused
	o.paused = now.Paused
	if appear {
		o.shown, o.until = true, time.Now().Add(linger)
	}
	o.mu.Unlock()
	if shown || appear {
		o.render(appear)
	}
}

func (o *Overlay) signal() {
	select {
	case o.changed <- struct{}{}:
	default:
	}
}

func (o *Overlay) release() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.bufs = [2][]byte{}
	o.have = false
}

func (o *Overlay) render(force bool) {
	now := o.c.Now()
	orient := o.orientation()
	key := fmt.Sprintf("%v|%d|%v|%v|%s|%s|%s|%d|%d", orient, now.Elapsed/time.Second, now.Paused, now.Playing, now.Title, now.Artist, now.Album, now.MarkID, now.Can)
	o.mu.Lock()
	if !force && key == o.drawn {
		o.mu.Unlock()
		return
	}
	o.mu.Unlock()

	fw, fh := o.native()
	vw, vh := orient.Size(fw, fh)
	img, band, hits := Draw(o.c, now, vw, vh, o.palette())

	at := fbRect(orient, fw, fh, band)
	stride := (at.W*4 + 63) &^ 63

	o.mu.Lock()
	defer o.mu.Unlock()
	o.hits, o.drawn = hits, key
	if len(o.bufs[o.next]) < stride*at.H {
		o.bufs[o.next] = make([]byte, max(stride*at.H, (fh*4+63)&^63*vbandMost(fw, fh)))
	}
	buf := o.bufs[o.next]
	o.next ^= 1
	turn(buf, stride, at, img, band, orient, fw, fh)

	o.band = video.Band{Pix: buf, Stride: stride, At: at}
	o.have = true
	o.signal()
}

func vbandMost(fw, fh int) int { return min(fw, fh) * bandPart / 100 }

func fbRect(o display.Orientation, fw, fh int, v ui.Rect) display.Rect {
	x0, y0 := o.Project(fw, fh, v.X, v.Y)
	x1, y1 := o.Project(fw, fh, v.X+v.W-1, v.Y+v.H-1)
	return display.Rect{X: min(x0, x1), Y: min(y0, y1), W: abs(x1-x0) + 1, H: abs(y1-y0) + 1}
}

func turn(dst []byte, stride int, at display.Rect, img *ui.Image, band ui.Rect, o display.Orientation, fw, fh int) {
	for y := 0; y < at.H; y++ {
		row := dst[y*stride:]
		for x := 0; x < at.W; x++ {
			vx, vy := o.Unproject(fw, fh, at.X+x, at.Y+y)
			iy := vy - band.Y
			c := img.At(vx-band.X, iy)
			p := row[x*4 : x*4+4]
			if c == (theme.Color{}) {
				a := byte(40 + 170*iy/max(band.H, 1))
				p[0], p[1], p[2], p[3] = 0, 0, 0, a
				continue
			}
			p[0], p[1], p[2], p[3] = c.R, c.G, c.B, 0xff
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func Draw(c Controls, now media.Now, vw, vh int, palette theme.Theme) (*ui.Image, ui.Rect, []hit) {
	m := widget.New(vw, vh)
	sub := strings.Join(nonEmpty(now.Artist, now.Album), " · ")
	_, sh := m.Hint.Measure("Ag")
	bh := m.Unit*bandPart/100 + m.Row + m.Pad
	if sub != "" {
		bh += sh + m.Pad/2
	}
	band := ui.Rect{X: 0, Y: vh - bh, W: vw, H: bh}
	img := ui.NewImage(vw, bh, theme.Color{})
	ink := theme.Color{R: 0xf2, G: 0xf2, B: 0xf2}
	quiet := theme.Color{R: 0xb0, G: 0xb0, B: 0xb0}
	black := theme.Color{}

	pad := m.Pad * 2
	var hits []hit
	add := func(r ui.Rect, do func(x int)) {
		hits = append(hits, hit{at: ui.Rect{X: r.X + band.X, Y: r.Y + band.Y, W: r.W, H: r.H}, do: do})
	}

	title := now.Title
	if title == "" {
		title = c.Label()
	}
	_, th := m.Label.Measure("Ag")
	tx := pad
	if mark := ui.Picture(now.Mark, now.MarkID); mark != nil && vw > vh {
		iw, ih := mark.Size()
		side := th + sh
		mw := iw * side / max(ih, 1)
		ui.DrawScaled(img, mark, ui.Rect{X: pad, Y: pad, W: mw, H: side})
		tx += mw + pad
	}
	ui.DrawText(img, m.Label, tx, pad, ink, black, fit(m.Label, title, vw-tx-pad))
	barY := pad + th + pad
	if sub != "" {
		ui.DrawText(img, m.Hint, tx, pad+th+m.Pad/2, quiet, black, fit(m.Hint, sub, vw-tx-pad))
		barY += sh + m.Pad/2
	}
	thick := max(m.Unit/120, 4)
	bar := ui.Rect{X: pad, Y: barY, W: vw - pad*2, H: thick}
	if now.Length > 0 {
		ui.FillRounded(img, bar, thick/2, theme.Color{R: 0x60, G: 0x60, B: 0x60})
		done := min(max(int(int64(bar.W)*int64(now.Elapsed)/int64(now.Length)), 0), bar.W)
		if done > 0 {
			ui.FillRounded(img, ui.Rect{X: bar.X, Y: bar.Y, W: done, H: thick}, thick/2, palette.Accent)
		}
		if mk, ok := c.(Marked); ok {
			for _, r := range markRects(bar, now.Length, mk.Marks()) {
				ui.FillRect(img, r.at, r.color)
			}
		}
		knob := thick * 4
		ui.FillRounded(img, ui.Rect{X: bar.X + min(max(done-knob/2, 0), bar.W-knob), Y: bar.Y + thick/2 - knob/2, W: knob, H: knob}, knob/2, palette.Accent)
		_, hh := m.Hint.Measure("0:00")
		line := ui.Rect{X: bar.X, Y: bar.Y + thick + m.Pad, W: bar.W, H: hh}
		if now.LiveWithin > 0 {
			label, fill := say.T("player.live"), palette.Danger
			if behind := now.Length - now.Elapsed; behind > now.LiveWithin {
				fill = theme.Color{R: 0x60, G: 0x60, B: 0x60}
				ui.DrawText(img, m.Hint, line.X, line.Y, quiet, black, "-"+clock(behind))
			}
			w, h := m.Hint.Measure(label)
			pad := m.Pad / 2
			pill := ui.Rect{X: line.X + line.W - w - h - 2*pad, Y: line.Y - pad, W: w + h + 2*pad, H: h + 2*pad}
			ui.FillRounded(img, pill, pill.H/2, fill)
			ui.DrawText(img, m.Hint, pill.X+pill.H/2, line.Y, ink, fill, label)
			if c.CanSeek() {
				length := now.Length
				add(pill.Inset(-pad), func(int) { c.Seek(length) })
			}
		} else {
			ui.DrawText(img, m.Hint, line.X, line.Y, quiet, black, clock(now.Elapsed))
			ui.DrawTextRight(img, m.Hint, line, quiet, black, clock(now.Length))
		}
		if c.CanSeek() {
			length, left, wide := now.Length, bar.X+band.X, bar.W
			at := func(x int) time.Duration {
				return time.Duration(int64(length) * int64(min(max(x-left, 0), wide)) / int64(wide))
			}
			add(ui.Rect{X: bar.X, Y: bar.Y - knob, W: bar.W, H: knob*2 + thick}, func(x int) { c.Seek(at(x)) })
			hits[len(hits)-1].seek = at
		}
	}

	type control struct {
		icon     ui.Icon
		do       func()
		primary  bool
		disabled bool
	}
	atLive := now.LiveWithin > 0 && now.Length-now.Elapsed <= now.LiveWithin
	middle := control{icon: icons.AVPlayArrow, do: c.Play, primary: true}
	if now.Playing && !now.Paused {
		middle = control{icon: icons.AVPause, do: c.Pause, primary: true}
	}
	var shown []control
	if now.Can.Has(media.CanPrevious) {
		shown = append(shown, control{icon: icons.AVSkipPrevious, do: c.Previous})
	}
	if c.CanSeek() {
		shown = append(shown, control{icon: icons.AVReplay10, do: func() { c.Seek(c.Now().Elapsed - skip) }})
	}
	shown = append(shown, middle)
	if c.CanSeek() {
		shown = append(shown, control{icon: icons.AVForward10, do: func() { c.Seek(c.Now().Elapsed + skip) }, disabled: atLive})
	}
	if now.Can.Has(media.CanNext) {
		shown = append(shown, control{icon: icons.AVSkipNext, do: c.Next})
	}
	if now.Can.Has(media.CanStop) {
		shown = append(shown, control{icon: icons.AVStop, do: func() {
			c.Stop()
			shell.Get().Pop()
		}})
	}

	top := barY + thick + m.Pad*3
	if now.Length > 0 {
		_, hh := m.Hint.Measure("0:00")
		top += hh
	}
	row := ui.Rect{X: pad, Y: bh - pad - m.Row, W: vw - pad*2, H: m.Row}
	m.Draw(img, row, widget.Row{
		Kind:  widget.Slider,
		Level: volume.Get().Level(config.StreamMedia),
		Icon:  volume.Mark(config.StreamMedia),
	}, palette.On(true), black)
	onScreen := ui.Rect{X: row.X + band.X, Y: row.Y + band.Y, W: row.W, H: row.H}
	level := func(x int) int { return m.Level(onScreen, x) }
	add(row, func(x int) { volume.Get().Set(config.StreamMedia, level(x)) })
	hits[len(hits)-1].level = level

	big := min(row.Y-top-m.Pad, m.Row*2)
	small := big * 3 / 4
	gap := big / 2
	width := -gap
	for _, s := range shown {
		width += gap + map[bool]int{true: big, false: small}[s.primary]
	}
	x := (vw - width) / 2
	for _, s := range shown {
		side := small
		fill := theme.Color{R: 0x2a, G: 0x2a, B: 0x2a}
		mark := ink
		if s.primary {
			side, fill, mark = big, palette.Accent, palette.Background
		}
		if s.disabled {
			fill, mark = theme.Color{R: 0x1a, G: 0x1a, B: 0x1a}, theme.Color{R: 0x55, G: 0x55, B: 0x55}
		}
		box := ui.Rect{X: x, Y: top + (big-side)/2, W: side, H: side}
		ui.FillRounded(img, box, side/2, fill)
		ui.DrawIcon(img, s.icon, box.Inset(side/4), mark, fill)
		if !s.disabled {
			do := s.do
			add(box.Inset(-gap/3), func(int) { do() })
		}
		x += side + gap
	}
	return img, band, hits
}

type markRect struct {
	at    ui.Rect
	color theme.Color
}

func markRects(bar ui.Rect, length time.Duration, marks []Mark) []markRect {
	if length <= 0 {
		return nil
	}
	x := func(d time.Duration) int {
		return bar.X + min(max(int(int64(bar.W)*int64(d)/int64(length)), 0), bar.W)
	}
	var out []markRect
	for _, m := range marks {
		x0, x1 := x(m.From), x(m.To)
		if x1 <= x0 {
			x1 = x0 + 1
		}
		if x0 >= bar.X+bar.W {
			continue
		}
		out = append(out, markRect{at: ui.Rect{X: x0, Y: bar.Y, W: x1 - x0, H: bar.H}, color: m.Color})
	}
	return out
}

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

func nonEmpty(ss ...string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func fit(f *ui.Font, s string, width int) string {
	if w, _ := f.Measure(s); w <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 {
		r = r[:len(r)-1]
		if w, _ := f.Measure(string(r) + "…"); w <= width {
			return string(r) + "…"
		}
	}
	return ""
}

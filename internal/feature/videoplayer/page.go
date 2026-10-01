package videoplayer

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/feature/volume"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

const (
	spinDots  = 12
	spinEvery = 100 * time.Millisecond
	spinAfter = 300 * time.Millisecond
)

// Page is a video on the shell: the picture shows through it, and its controls, spinner, logo
// and errors are drawn like any other screen.
type Page struct {
	c Controls

	mu      sync.Mutex
	picture bool
	loading bool
	began   time.Time
	note    string
	logo    *ui.Image
	frameW  int
	frameH  int
	shown   bool
	paused  bool
	until   time.Time
	hits    []hit
	back    ui.Rect
	drawn   string
	hold    *shell.Hold

	scrub     func(x int) time.Duration
	scrubAt   time.Duration
	landing   time.Duration
	landUntil time.Time

	leveling func(x int) int
}

func (p *Page) Shows(s config.Stream) bool { return s == config.StreamMedia }

func (p *Page) reveal() {
	p.mu.Lock()
	p.shown, p.until = true, time.Now().Add(linger)
	p.mu.Unlock()
	p.redraw()
}

func NewPage(c Controls) *Page { return &Page{c: c, loading: true, began: time.Now()} }

func (p *Page) Covers() bool { return true }

// Follow points the controls at what plays now, when one track hands over to the next.
func (p *Page) Follow(c Controls) {
	p.mu.Lock()
	p.c = c
	p.mu.Unlock()
}

func (p *Page) controls() Controls {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.c
}

func (p *Page) SetPicture(on bool) {
	p.mu.Lock()
	p.picture = on
	if on {
		p.loading = false
	}
	up := p.hold.Held()
	p.mu.Unlock()
	p.redraw()
	if on && !up {
		p.Show()
	}
}

func (p *Page) Wakes() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.picture
}

func (p *Page) SetLoading(on bool) {
	p.mu.Lock()
	changed := on != p.loading
	if on && changed {
		p.began = time.Now()
	}
	p.loading = on
	p.mu.Unlock()
	if changed {
		p.redraw()
	}
}

func (p *Page) SetNote(note string) {
	p.mu.Lock()
	p.note = note
	if note != "" {
		p.loading = false
	}
	p.mu.Unlock()
	p.redraw()
}

func (p *Page) SetFrame(w, h int) {
	p.mu.Lock()
	p.frameW, p.frameH = w, h
	p.mu.Unlock()
	p.redraw()
}

func (p *Page) SetLogo(img *ui.Image) {
	p.mu.Lock()
	p.logo = img
	p.mu.Unlock()
	p.redraw()
}

func (p *Page) redraw() {
	if shell.Get().Top() == p {
		shell.Get().Redraw()
	}
}

// Run puts the page up and keeps it current until ctx ends.
func (p *Page) Run(ctx context.Context) {
	p.Show()
	quiet := volume.Get().Changed.Listen(func(c volume.Change) {
		if c.Stream == config.StreamMedia && shell.Get().Top() == shell.View(p) {
			p.reveal()
		}
	})
	defer quiet()
	defer func() {
		p.mu.Lock()
		hold := p.hold
		p.hold = nil
		p.mu.Unlock()
		hold.Release()
	}()
	t := time.NewTicker(spinEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.tick()
		}
	}
}

// Show puts the page back up, for coming back to it from the mini player.
func (p *Page) Show() {
	p.mu.Lock()
	held := p.hold.Held()
	p.mu.Unlock()
	if held {
		return
	}
	hold := shell.Get().Hold(p)
	p.mu.Lock()
	p.hold = hold
	p.mu.Unlock()
}

var watched atomic.Int64

func Watched() time.Time {
	at := watched.Load()
	if at == 0 {
		return time.Time{}
	}
	return time.Unix(0, at)
}

func (p *Page) tick() {
	now := p.controls().Now()
	p.mu.Lock()
	if p.picture && now.Playing && !now.Paused {
		watched.Store(time.Now().UnixNano())
	}
	if p.shown && now.Playing && !now.Paused && time.Now().After(p.until) {
		p.shown = false
	}
	if !p.shown && now.Paused && !p.paused {
		p.shown, p.until = true, time.Now().Add(linger)
	}
	p.paused = now.Paused
	spinning := p.loading && time.Since(p.began) > spinAfter
	key := fmt.Sprintf("%v|%v|%d|%v|%v|%s|%s|%s|%d|%d|%d|%d", p.shown, p.picture, now.Elapsed/time.Second, now.Paused, now.Playing, now.Title, now.Artist, now.Album, now.ArtID, now.MarkID, now.Can, volume.Get().Level(config.StreamMedia))
	changed := key != p.drawn
	p.drawn = key
	p.mu.Unlock()
	if spinning || changed {
		p.redraw()
	}
}

func (p *Page) Grab(x, y int) bool {
	p.mu.Lock()
	if !p.shown {
		p.mu.Unlock()
		return false
	}
	for _, h := range p.hits {
		if h.seek != nil && h.at.Contains(x, y) {
			p.scrub, p.scrubAt = h.seek, h.seek(x)
			p.until = time.Now().Add(linger)
			p.mu.Unlock()
			p.redraw()
			return true
		}
		if h.level != nil && h.at.Contains(x, y) {
			p.leveling = h.level
			p.until = time.Now().Add(linger)
			p.mu.Unlock()
			volume.Get().Set(config.StreamMedia, h.level(x))
			return true
		}
	}
	p.mu.Unlock()
	return false
}

func (p *Page) Drag(x, _ int) bool {
	p.mu.Lock()
	if lv := p.leveling; lv != nil {
		p.until = time.Now().Add(linger)
		p.mu.Unlock()
		if want := lv(x); want != volume.Get().Level(config.StreamMedia) {
			volume.Get().Set(config.StreamMedia, want)
		}
		return false
	}
	if p.scrub == nil {
		p.mu.Unlock()
		return false
	}
	at := p.scrub(x)
	moved := at/time.Second != p.scrubAt/time.Second
	p.scrubAt = at
	p.until = time.Now().Add(linger)
	p.mu.Unlock()
	if moved {
		p.redraw()
	}
	return false
}

func (p *Page) Let(x, _ int) bool {
	p.mu.Lock()
	p.leveling = nil
	scrub := p.scrub
	p.scrub = nil
	if scrub != nil {
		p.landing, p.landUntil = scrub(x), time.Now().Add(landWait)
	}
	at := p.landing
	p.mu.Unlock()
	if scrub == nil {
		return false
	}
	p.controls().Seek(at)
	p.redraw()
	return false
}

const (
	landed   = 1500 * time.Millisecond
	landWait = 3 * time.Second
)

func (p *Page) Tap(x, y int) bool {
	p.mu.Lock()
	if !p.shown {
		p.shown, p.until = true, time.Now().Add(linger)
		p.mu.Unlock()
		p.redraw()
		return true
	}
	if p.back.Contains(x, y) {
		p.mu.Unlock()
		shell.Get().Pop()
		return true
	}
	var do func(int)
	for _, h := range p.hits {
		if h.at.Contains(x, y) {
			do = h.do
			break
		}
	}
	if do == nil {
		p.shown = false
		p.mu.Unlock()
		p.redraw()
		return true
	}
	p.until = time.Now().Add(linger)
	p.mu.Unlock()
	do(x)
	p.redraw()
	return true
}

func (p *Page) Draw(s ui.Surface, palette theme.Theme) {
	w, h := s.Size()
	whole := ui.Rect{W: w, H: h}

	p.mu.Lock()
	picture, loading, began, note, logo, shown := p.picture, p.loading, p.began, p.note, p.logo, p.shown
	p.mu.Unlock()

	if picture {
		ui.Clear(s, whole)
	} else {
		ui.FillRect(s, whole, theme.Color{})
		drawBackdrop(s, w, h, logo, note)
	}
	if loading && time.Since(began) > spinAfter {
		drawSpinner(s, w, h, time.Since(began))
	}

	var hits []hit
	var back ui.Rect
	if shown {
		m := widget.New(w, h)
		back = m.Header(whole)
		arrow := m.Arrow(whole)
		ui.Shade(s, ui.Rect{W: w, H: back.H}, 170, 0)
		m.Back(s, arrow, theme.Color{R: 0xf2, G: 0xf2, B: 0xf2})
		c := p.controls()
		now := c.Now()
		p.mu.Lock()
		switch {
		case p.scrub != nil:
			now.Elapsed = p.scrubAt
		case time.Now().Before(p.landUntil):
			if d := now.Elapsed - p.landing; d > landed || d < -landed {
				now.Elapsed = p.landing
			} else {
				p.landUntil = time.Time{}
			}
		}
		p.mu.Unlock()
		img, band, got := Draw(c, now, w, h, palette)
		ui.Shade(s, band, 40, 210)
		bw, bh := img.Size()
		for y := range bh {
			for x := range bw {
				if c := img.At(x, y); c != (theme.Color{}) {
					s.Set(band.X+x, band.Y+y, c)
				}
			}
		}
		hits = got
	}
	p.mu.Lock()
	p.hits, p.back = hits, back
	p.mu.Unlock()
}

func drawBackdrop(s ui.Surface, w, h int, logo *ui.Image, note string) {
	side := min(w, h) * 2 / 5
	top := (h-side)/2 - h/12
	if logo != nil {
		lw, lh := logo.Size()
		iw, ih := side, side*lh/max(lw, 1)
		if ih > side {
			iw, ih = side*lw/max(lh, 1), side
		}
		ui.DrawScaled(s, logo, ui.Rect{X: (w - iw) / 2, Y: top + (side-ih)/2, W: iw, H: ih})
	}
	if note == "" {
		return
	}
	f := ui.MustLoad(ui.Regular, max(h/28, 12))
	_, th := f.Measure(note)
	width := w * 9 / 10
	lines := ui.Wrap(f, note, width, noteLines)
	for i, line := range lines {
		ui.DrawTextIn(s, f, ui.Rect{X: w / 20, Y: top + side + th + i*th*3/2, W: width, H: th * 2},
			theme.Color{R: 0xdd, G: 0xdd, B: 0xdd}, theme.Color{}, fit(f, line, width))
	}
}

const noteLines = 3

func drawSpinner(s ui.Surface, w, h int, since time.Duration) {
	r := min(w, h) / 14
	dot := max(r/5, 3)
	cx, cy := w/2, h/2
	lead := int(since/spinEvery) % spinDots
	for i := range spinDots {
		a := 2 * math.Pi * float64(i) / spinDots
		x := cx + int(float64(r)*math.Sin(a))
		y := cy - int(float64(r)*math.Cos(a))
		age := (lead - i + spinDots) % spinDots
		v := byte(0xf0 - age*0xb0/spinDots)
		ui.FillRounded(s, ui.Rect{X: x - dot, Y: y - dot, W: dot * 2, H: dot * 2}, dot, theme.Color{R: v, G: v, B: v})
	}
}

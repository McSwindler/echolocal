package idle

import (
	"fmt"
	"image"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/config"
	dashboard "github.com/ygelfand/echolocal/internal/feature/clock"
	"github.com/ygelfand/echolocal/internal/feature/clock/face"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/feature/visuals"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/visual"
)

const (
	frameEvery = time.Second / 20
	forever    = 100 * 365 * 24 * time.Hour
	sayEvery   = 30 * time.Second
)

type View struct {
	mu      sync.Mutex
	vis     [2]visual.Visual
	kinds   [2]string
	running bool
	shown   string
	stats   stats
	stamps  Stamps
	drawn   time.Time

	gmu    sync.Mutex
	gl     [2]*shading
	broken [2]visual.Visual
}

type stats struct {
	since          time.Time
	frames         int
	spent, slowest time.Duration
	speaker, mic   float32
}

func (s *stats) add(took time.Duration, x visual.Input) {
	if s.since.IsZero() {
		s.since = time.Now()
	}
	s.frames++
	s.spent += took
	s.slowest = max(s.slowest, took)
	s.speaker = max(s.speaker, x.Speaker.Level)
	s.mic = max(s.mic, x.Mic.Level)
}

func (s *stats) say(kinds []string) {
	if s.frames == 0 {
		return
	}
	held := time.Since(s.since)
	slog.Info("idle frames",
		"visuals", kinds,
		"painted", fmt.Sprintf("%.1f/s", float64(s.frames)/held.Seconds()),
		"draw", (s.spent / time.Duration(s.frames)).Round(100*time.Microsecond),
		"slowest", s.slowest.Round(100*time.Microsecond),
		"speaker", fmt.Sprintf("%.3f", s.speaker),
		"mic", fmt.Sprintf("%.3f", s.mic))
	*s = stats{}
}

func newView() *View { return &View{} }

func (v *View) Covers() bool           { return true }
func (v *View) Tap(int, int) bool      { return false }
func (v *View) Asleep() bool           { return true }
func (v *View) Timeout() time.Duration { return forever }
func (v *View) Draw(s ui.Surface, palette theme.Theme) {
	cfg := config.Get()
	slots := chosen(cfg.Idle)

	v.mu.Lock()
	if !v.running {
		v.running = true
		go v.run()
	}
	var vis []visual.Visual
	for n, slot := range slots {
		if v.vis[n] == nil || v.kinds[n] != slot.Kind {
			v.vis[n], v.kinds[n] = visual.New(visual.Kind(slot.Kind)), slot.Kind
		}
		vis = append(vis, v.vis[n])
	}
	v.shown = key(cfg, time.Now())
	v.mu.Unlock()

	var x visual.Input
	var backdrop *image.RGBA
	w, h := s.Size()
	if len(slots) > 0 {
		x = visuals.Get().Input()
		now := time.Now()
		v.mu.Lock()
		if !v.drawn.IsZero() {
			x.Dt = now.Sub(v.drawn)
		}
		v.drawn = now
		v.mu.Unlock()
		for n, area := range Areas(w, h, len(slots)) {
			v.shaded(n, vis[n], area, slots[n].Source)
		}
	}
	start := time.Now()
	PaintWith(s, cfg, slots, backdrop, palette, time.Now(), &v.stamps)
	took := time.Since(start)

	if len(slots) > 0 {
		v.mu.Lock()
		v.stats.add(took, x)
		if time.Since(v.stats.since) >= sayEvery {
			v.stats.say(kindsOf(slots))
		}
		v.mu.Unlock()
	}
}

func kindsOf(slots []config.IdleVisual) []string {
	out := make([]string, len(slots))
	for n, s := range slots {
		out[n] = s.Kind + "/" + string(s.Source)
	}
	return out
}

func chosen(c config.Idle) []config.IdleVisual {
	var out []config.IdleVisual
	for _, v := range []config.IdleVisual{c.First, c.Second} {
		if v.On() {
			out = append(out, v)
		}
	}
	return out
}

func Areas(w, h, n int) []ui.Rect {
	switch {
	case n <= 0:
		return nil
	case n == 1:
		return []ui.Rect{{W: w, H: h}}
	case w > h:
		return []ui.Rect{{W: w / 2, H: h}, {X: w / 2, W: w - w/2, H: h}}
	}
	return []ui.Rect{{W: w, H: h / 2}, {Y: h / 2, W: w, H: h - h/2}}
}

func reading(cfg config.Config, at time.Time) face.Reading {
	r := face.Read(at, cfg.Screen.Hours == config.TwentyFourHour)
	if !cfg.Clock.Date {
		r = r.Undated()
	}
	return r
}

func key(cfg config.Config, at time.Time) string {
	r := reading(cfg, at)
	k := fmt.Sprintf("%v %+v logo=%v ink=%s", r, cfg.Idle, cfg.Screen.Logo, cfg.Clock.Ink)
	if face.Ticks(cfg.Idle.Face) {
		k = fmt.Sprintf("%s second=%d", k, r.Second)
	}
	return k
}

func Paint(s ui.Surface, cfg config.Config, slots []config.IdleVisual, backdrop *image.RGBA, palette theme.Theme, at time.Time) {
	PaintWith(s, cfg, slots, backdrop, palette, at, nil)
}

func PaintWith(s ui.Surface, cfg config.Config, slots []config.IdleVisual, backdrop *image.RGBA, palette theme.Theme, at time.Time, stamps *Stamps) {
	w, h := s.Size()
	if backdrop != nil && backdrop.Bounds().Dx() == w && backdrop.Bounds().Dy() == h && len(slots) == 0 {
		ui.DrawRGBA(s, 0, 0, backdrop, 1, ui.Rect{})
	} else {
		ui.Fill(s, palette.Background)
	}

	for _, area := range Areas(w, h, len(slots)) {
		ui.Clear(s, area)
	}

	if cfg.Screen.Logo {
		ui.DrawLogo(s, dashboard.Mark(w, h), palette.Background)
	}

	if cfg.Idle.Face != config.FaceNone {
		box := dashboard.Place(cfg.Idle.Position, cfg.Idle.Align, cfg.Idle.Size, w, h)
		r, f := reading(cfg, at), face.Of(cfg.Idle.Face)
		tones := make([]visual.Traits, len(slots))
		for n, slot := range slots {
			tones[n] = visual.Kind(slot.Kind).Traits()
		}
		for _, p := range Pieces(palette, tones, Areas(w, h, len(slots)), box) {
			on := s
			if p.Clip.W > 0 {
				on = ui.Within(s, p.Clip)
			}
			ink := cfg.Clock.Ink.Over(p.Palette)
			if stamps == nil || len(slots) == 0 {
				f.Draw(on, box, r, ink)
				continue
			}
			said := fmt.Sprintf("%s s=%d", r, r.Second)
			paintStamp(on, stamps.get(stampKey(said, box, ink, f), func() []span { return makeStamp(f, box, r, ink) }))
		}
	}
}

type Piece struct {
	Clip    ui.Rect
	Palette theme.Theme
}

func Pieces(palette theme.Theme, tones []visual.Traits, areas []ui.Rect, box ui.Rect) []Piece {
	var out []Piece
	for n, area := range areas {
		if n >= len(tones) {
			break
		}
		if clip := intersect(area, box); clip.W > 0 {
			out = append(out, Piece{Clip: clip, Palette: tones[n].Under(palette)})
		}
	}
	if len(out) == 0 {
		return []Piece{{Palette: palette}}
	}
	if len(out) == 2 && out[0].Palette == out[1].Palette {
		return []Piece{{Palette: out[0].Palette}}
	}
	return out
}

func intersect(a, b ui.Rect) ui.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return ui.Rect{}
	}
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func (v *View) run() {
	var release func()
	defer func() {
		if release != nil {
			release()
		}
		v.unshadeAll()
	}()

	t := time.NewTicker(shadeEvery)
	defer t.Stop()
	var next, shaded time.Time
	for now := range t.C {
		if !shell.Get().Visible(v) {
			v.mu.Lock()
			v.stats.say(kindsOf(chosen(config.Get().Idle)))
			v.running = false
			v.drawn = time.Time{}
			v.mu.Unlock()
			return
		}
		if fps := config.Get().Visual.MaxFPS; fps <= 0 || now.Sub(shaded) >= time.Second/time.Duration(fps)-2*time.Millisecond {
			shaded = now
			v.shade()
		}
		if now.Before(next) {
			continue
		}
		next = now.Add(frameEvery)

		cfg := config.Get()
		if slots := chosen(cfg.Idle); len(slots) > 0 {
			if release == nil {
				release = visuals.Get().Hold()
			}
			if v.painting(len(slots)) {
				shell.Get().Redraw()
				continue
			}
		} else {
			v.unshadeAll()
			if release != nil {
				release()
				release = nil
			}
		}

		v.mu.Lock()
		stale := v.shown != key(cfg, time.Now())
		v.mu.Unlock()
		if stale {
			shell.Get().Redraw()
		}
	}
}

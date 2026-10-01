package videoplayer

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

type fake struct {
	now    media.Now
	seeks  []time.Duration
	paused int
}

func (f *fake) Play()                 { f.now.Paused = false }
func (f *fake) Pause()                { f.now.Paused = true; f.paused++ }
func (f *fake) Stop()                 {}
func (f *fake) Playing() (bool, bool) { return f.now.Playing && !f.now.Paused, f.now.Paused }
func (f *fake) Next()                 {}
func (f *fake) Previous()             {}
func (f *fake) Now() media.Now        { return f.now }
func (f *fake) Kind() media.Kind      { return media.FromCast }
func (f *fake) Label() string         { return "Yuri's phone" }
func (f *fake) Seek(to time.Duration) { f.seeks = append(f.seeks, to) }
func (f *fake) CanSeek() bool         { return true }

func playing() *fake {
	return &fake{now: media.Now{
		Playing: true, Title: "Is the AI Bubble About to Be Tested?",
		Elapsed: 95 * time.Second, Length: 14*time.Minute + 3*time.Second,
		Can: media.CanPause | media.CanStop | media.CanNext | media.CanPrevious,
	}}
}

func TestTheTurnedBandMatchesTheDrawingAtEveryOrientation(t *testing.T) {
	fw, fh := 480, 960
	for _, o := range []display.Orientation{display.Rotate0, display.Rotate90, display.Rotate180, display.Rotate270} {
		vw, vh := o.Size(fw, fh)
		img, band, _ := Draw(playing(), playing().now, vw, vh, theme.All[0])
		at := fbRect(o, fw, fh, band)
		if at.W*at.H != band.W*band.H {
			t.Fatalf("%v: fb rect %+v holds %d pixels, the band %d", o, at, at.W*at.H, band.W*band.H)
		}
		stride := (at.W*4 + 63) &^ 63
		buf := make([]byte, stride*at.H)
		turn(buf, stride, at, img, band, o, fw, fh)
		for _, p := range [][2]int{{band.W / 2, band.H - 5}, {10, 10}, {band.W - 3, band.H / 2}} {
			fx, fy := o.Project(fw, fh, band.X+p[0], band.Y+p[1])
			px := buf[(fy-at.Y)*stride+(fx-at.X)*4:]
			c := img.At(p[0], p[1])
			if c != (theme.Color{}) && (px[0] != c.R || px[1] != c.G || px[2] != c.B || px[3] != 0xff) {
				t.Errorf("%v: viewed %v drew %v, the buffer holds %v", o, p, c, px[:4])
			}
		}
	}
}

func TestTheScrubBarSeeksToWhereItWasTapped(t *testing.T) {
	f := playing()
	vw, vh := 1920, 1200
	_, band, hits := Draw(f, f.now, vw, vh, theme.All[0])
	var bar hit
	levels := 0
	for _, h := range hits {
		if h.seek != nil {
			bar = h
		}
		if h.level != nil && h.at.W > vw/2 {
			levels++
		}
	}
	if bar.do == nil {
		t.Fatal("no scrub bar among the hits")
	}
	if levels != 1 {
		t.Errorf("%d volume rows among the hits", levels)
	}
	bar.do(bar.at.X + bar.at.W/2)
	if len(f.seeks) != 1 || absDur(f.seeks[0]-f.now.Length/2) > time.Second {
		t.Errorf("tapping the middle of the bar sought %v, want about %v", f.seeks, f.now.Length/2)
	}
	if bar.at.Y < band.Y {
		t.Errorf("the bar's hit %+v is above the band %+v", bar.at, band)
	}
}

func TestATapShowsItAButtonActsAndAnEmptyTapHides(t *testing.T) {
	f := playing()
	o := New(f)
	o.orientation = func() display.Orientation { return display.Rotate0 }
	o.palette = func() theme.Theme { return theme.All[0] }
	o.native = func() (int, int) { return 960, 480 }
	o.Tap(600, 100)
	o.mu.Lock()
	shown, hits := o.shown, o.hits
	o.mu.Unlock()
	if !shown {
		t.Fatal("a tap did not show the controls")
	}
	var pause hit
	for _, h := range hits {
		if h.at.W < 400 && h.at.W > pause.at.W {
			pause = h
		}
	}
	x, y := pause.at.Center()
	o.Tap(x, y)
	if f.paused != 1 {
		t.Errorf("tapping the biggest button paused %d times, want once", f.paused)
	}
	o.Tap(600, 100)
	o.mu.Lock()
	shown = o.shown
	o.mu.Unlock()
	if shown {
		t.Error("a tap away from the controls did not hide them")
	}
}

func TestAPauseFromElsewhereShowsTheControlsOnce(t *testing.T) {
	f := playing()
	o := New(f)
	o.orientation = func() display.Orientation { return display.Rotate0 }
	o.palette = func() theme.Theme { return theme.All[0] }
	o.native = func() (int, int) { return 960, 480 }
	shown := func() bool {
		o.mu.Lock()
		defer o.mu.Unlock()
		return o.shown
	}

	o.tick()
	if shown() {
		t.Fatal("the controls came up while playing")
	}
	f.now.Paused = true
	o.tick()
	if !shown() {
		t.Fatal("a pause from elsewhere did not bring the controls up")
	}
	o.Tap(600, 100)
	o.tick()
	if shown() {
		t.Error("the controls came back after being dismissed while still paused")
	}
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

type marked struct {
	*fake
	marks []Mark
}

func (m marked) Marks() []Mark { return m.marks }

var green = theme.Color{G: 0xd4}

func sponsored() marked {
	return marked{playing(), []Mark{{From: 3 * time.Minute, To: 4*time.Minute + 30*time.Second, Color: green}}}
}

func TestMarksSitOnTheBarWhereTheirTimesAre(t *testing.T) {
	bar := ui.Rect{X: 100, Y: 50, W: 1000, H: 8}
	got := markRects(bar, 10*time.Minute, []Mark{
		{From: time.Minute, To: 2 * time.Minute, Color: green},
		{From: 9 * time.Minute, To: 12 * time.Minute, Color: green},
		{From: 5 * time.Minute, To: 5 * time.Minute, Color: green},
	})
	want := []ui.Rect{{X: 200, Y: 50, W: 100, H: 8}, {X: 1000, Y: 50, W: 100, H: 8}, {X: 600, Y: 50, W: 1, H: 8}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i].at != want[i] {
			t.Errorf("mark %d at %v, want %v", i, got[i].at, want[i])
		}
	}
	if markRects(bar, 0, []Mark{{To: time.Minute}}) != nil {
		t.Error("marks drawn on a bar with no length")
	}
}

func TestTheOverlayDrawsTheMarks(t *testing.T) {
	has := func(c Controls) bool {
		img, _, _ := Draw(c, c.Now(), 1920, 1200, theme.All[0])
		w, h := img.Size()
		for y := range h {
			for x := range w {
				if img.At(x, y) == green {
					return true
				}
			}
		}
		return false
	}
	if !has(sponsored()) {
		t.Error("no mark on the bar")
	}
	if has(playing()) {
		t.Error("a mark with none given")
	}
}

func TestRenderTheOverlay(t *testing.T) {
	dir := os.Getenv("LANOVO_RENDER")
	if dir == "" {
		t.Skip("set LANOVO_RENDER to a directory")
	}
	for name, size := range map[string][2]int{"landscape": {960, 480}} {
		vw, vh := size[0], size[1]
		img, band, _ := Draw(sponsored(), playing().now, vw, vh, theme.All[0])
		out := image.NewRGBA(image.Rect(0, 0, vw, vh))
		for y := range vh {
			for x := range vw {
				v := uint8(40 + 120*x/vw)
				out.Set(x, y, color.RGBA{v, uint8(60 + 80*y/vh), 150, 255})
			}
		}
		for y := range band.H {
			for x := range band.W {
				c := img.At(x, y)
				if c == (theme.Color{}) {
					a := float64(40+170*y/band.H) / 255
					bg := out.RGBAAt(x, band.Y+y)
					out.Set(x, band.Y+y, color.RGBA{uint8(float64(bg.R) * (1 - a)), uint8(float64(bg.G) * (1 - a)), uint8(float64(bg.B) * (1 - a)), 255})
					continue
				}
				out.Set(x, band.Y+y, color.RGBA{c.R, c.G, c.B, 255})
			}
		}
		f, err := os.Create(filepath.Join(dir, "video-overlay-"+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, out)
		f.Close()
	}
}

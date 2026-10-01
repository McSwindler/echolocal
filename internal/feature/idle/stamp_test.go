package idle

import (
	"testing"
	"time"

	"github.com/ygelfand/echolocal/internal/config"
	dashboard "github.com/ygelfand/echolocal/internal/feature/clock"
	"github.com/ygelfand/echolocal/internal/feature/clock/face"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/visual"
)

func busy(w, h int) *ui.Image {
	img := ui.NewImage(w, h, theme.Color{})
	for y := range h {
		for x := range w {
			img.Set(x, y, theme.Color{R: byte(x * 7), G: byte(y * 5), B: byte((x + y) * 3)})
		}
	}
	return img
}

func diff(a, b theme.Color) int {
	d := func(x, y byte) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return max(d(a.R, b.R), d(a.G, b.G), d(a.B, b.B))
}

func TestAStampedClockLooksLikeADrawnOne(t *testing.T) {
	const w, h = 1200, 800
	palette := theme.Default().On(true)
	box := dashboard.Place(config.PositionCenter, config.AlignCenter, config.SizeMedium, w, h)
	r := face.Read(noon.Add(17*time.Second), true)

	for _, kind := range config.Faces() {
		f := face.Of(kind)
		drawn, stamped := busy(w, h), busy(w, h)
		f.Draw(drawn, box, r, palette)
		paintStamp(stamped, makeStamp(f, box, r, palette))

		worst, wx, wy := 0, 0, 0
		for y := range h {
			for x := range w {
				if d := diff(drawn.At(x, y), stamped.At(x, y)); d > worst {
					worst, wx, wy = d, x, y
				}
			}
		}
		if worst > 3 {
			t.Errorf("%s: stamped differs from drawn by %d at %d,%d", kind, worst, wx, wy)
		}
	}
}

func TestTheSecondsFaceStampsEverySecond(t *testing.T) {
	var st Stamps
	cfg := idled(func(c *config.Idle) {
		c.Face, c.First = config.FaceAnalogSeconds, config.IdleVisual{Kind: string(visual.Orb)}
	})
	slots := []config.IdleVisual{cfg.Idle.First}
	a, b := ui.NewImage(1200, 1920, theme.Color{}), ui.NewImage(1200, 1920, theme.Color{})

	PaintWith(a, cfg, slots, nil, theme.Default(), noon, &st)
	PaintWith(b, cfg, slots, nil, theme.Default(), noon.Add(time.Second), &st)
	if len(st.made) != 2 {
		t.Errorf("%d stamps for two seconds", len(st.made))
	}
}

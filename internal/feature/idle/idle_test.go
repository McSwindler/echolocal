package idle

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ygelfand/echolocal/internal/config"
	dashboard "github.com/ygelfand/echolocal/internal/feature/clock"
	"github.com/ygelfand/echolocal/internal/feature/clock/face"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/visual"
)

var noon = time.Date(2026, 9, 27, 12, 34, 0, 0, time.UTC)

func inked(img *ui.Image, box ui.Rect, bg theme.Color) bool {
	for y := box.Y; y < box.Y+box.H; y++ {
		for x := box.X; x < box.X+box.W; x++ {
			if img.At(x, y) != bg {
				return true
			}
		}
	}
	return false
}

func TestTwoVisualsSplitTheScreenTheLongWay(t *testing.T) {
	if got := Areas(1920, 1200, 1); !slices.Equal(got, []ui.Rect{{W: 1920, H: 1200}}) {
		t.Errorf("one: %v", got)
	}
	if got := Areas(1920, 1200, 2); !slices.Equal(got, []ui.Rect{{W: 960, H: 1200}, {X: 960, W: 960, H: 1200}}) {
		t.Errorf("landscape: %v", got)
	}
	if got := Areas(1200, 1920, 2); !slices.Equal(got, []ui.Rect{{W: 1200, H: 960}, {Y: 960, W: 1200, H: 960}}) {
		t.Errorf("portrait: %v", got)
	}
	if Areas(1200, 1920, 0) != nil {
		t.Error("areas for no visual")
	}
}

func TestItIsDueOnlyAfterTheWait(t *testing.T) {
	if due(0, time.Hour) {
		t.Error("never came up")
	}
	if due(time.Minute, 30*time.Second) {
		t.Error("came up before the wait")
	}
	if !due(time.Minute, 61*time.Second) {
		t.Error("did not come up after the wait")
	}
}

func TestTheClockSitsWhereItIsPut(t *testing.T) {
	const w, h = 1920, 1200
	left := dashboard.Place(config.PositionTop, config.AlignLeft, config.SizeSmall, w, h)
	right := dashboard.Place(config.PositionBottom, config.AlignRight, config.SizeSmall, w, h)
	center := dashboard.Place(config.PositionCenter, config.AlignCenter, config.SizeSmall, w, h)

	if left.X != 0 || left.Y != 0 {
		t.Errorf("top left at %v", left)
	}
	if right.X+right.W != w || right.Y+right.H != h {
		t.Errorf("bottom right at %v", right)
	}
	if d, e := w-(center.X*2+center.W), h-(center.Y*2+center.H); d < 0 || d > 1 || e < 0 || e > 1 {
		t.Errorf("center at %v", center)
	}
	if dashboard.Place(config.PositionCenter, config.AlignCenter, config.SizeLarge, w, h) != dashboard.Box(config.PositionCenter, config.SizeLarge, w, h) {
		t.Error("centered placement differs from the dashboard's box")
	}
}

func idled(change func(*config.Idle)) config.Config {
	cfg := config.Defaults()
	cfg.Screen.Logo = false
	cfg.Clock.Date = false
	change(&cfg.Idle)
	return cfg
}

func TestNoFaceAndNoVisualIsABlankScreen(t *testing.T) {
	palette := theme.Default()
	img := ui.NewImage(1920, 1200, theme.Color{R: 1})
	Paint(img, idled(func(c *config.Idle) { c.Face = config.FaceNone }), nil, nil, palette, noon)
	if inked(img, ui.Rect{W: 1920, H: 1200}, palette.Background) {
		t.Error("something drew with no face and no visual")
	}
}

func TestTheFaceIsDrawnOnTheSideItIsAligned(t *testing.T) {
	palette := theme.Default()
	for _, at := range []struct {
		align      config.Align
		ink, blank ui.Rect
	}{
		{config.AlignLeft, ui.Rect{W: 960, H: 1200}, ui.Rect{X: 1400, W: 520, H: 1200}},
		{config.AlignRight, ui.Rect{X: 960, W: 960, H: 1200}, ui.Rect{W: 520, H: 1200}},
	} {
		img := ui.NewImage(1920, 1200, palette.Background)
		cfg := idled(func(c *config.Idle) { c.Align, c.Size = at.align, config.SizeSmall })
		Paint(img, cfg, nil, nil, palette, noon)
		if !inked(img, at.ink, palette.Background) {
			t.Errorf("%s: nothing drawn on its side", at.align)
		}
		if inked(img, at.blank, palette.Background) {
			t.Errorf("%s: drawn on the far side", at.align)
		}
	}
}

func TestEachVisualLeavesItsHalfOpenForItsLayer(t *testing.T) {
	palette := theme.Default()
	img := ui.NewImage(1920, 1200, palette.Background)
	slots := []config.IdleVisual{{Kind: string(visual.DigitalVU), Source: config.SourceMic}, {Kind: string(visual.Orb), Source: config.SourceSpeaker}}
	cfg := idled(func(c *config.Idle) { c.Face, c.First, c.Second = config.FaceNone, slots[0], slots[1] })

	Paint(img, cfg, slots, nil, palette, noon)
	for _, half := range Areas(1920, 1200, 2) {
		if inked(img, half, theme.Color{}) {
			t.Errorf("%v was painted over", half)
		}
	}
}

func TestTheClockTakesItsColoursFromTheVisualUnderIt(t *testing.T) {
	var light theme.Theme
	for _, th := range theme.All {
		if !th.Dark {
			light = th
			break
		}
	}
	dark := []visual.Traits{{}, {}}
	areas := Areas(1920, 1200, 2)

	one := Pieces(light, dark, areas, ui.Rect{X: 100, Y: 100, W: 400, H: 200})
	if len(one) != 1 || !one[0].Palette.Dark || theme.Contrast(one[0].Palette.Text, one[0].Palette.Background) < 4.5 {
		t.Errorf("a light theme over a dark visual gave %+v", one)
	}
	if got := Pieces(light, dark, areas, ui.Rect{X: 700, Y: 100, W: 600, H: 200}); len(got) != 1 || got[0].Clip.W != 0 {
		t.Errorf("a clock across two dark visuals was split: %+v", got)
	}
	if got := Pieces(light, nil, nil, ui.Rect{W: 400, H: 200}); len(got) != 1 || got[0].Palette != light {
		t.Error("with no visual the clock left the theme")
	}
}

func TestAClockAcrossDifferentVisualsIsSplitAtTheSeam(t *testing.T) {
	palette := theme.Default()
	areas := Areas(1920, 1200, 2)
	box := ui.Rect{X: 700, Y: 400, W: 600, H: 300}

	got := Pieces(palette, []visual.Traits{{}, {Light: true}}, areas, box)
	if len(got) != 2 {
		t.Fatalf("%d pieces", len(got))
	}
	if got[0].Clip != (ui.Rect{X: 700, Y: 400, W: 260, H: 300}) || got[1].Clip != (ui.Rect{X: 960, Y: 400, W: 340, H: 300}) {
		t.Errorf("clips %v %v", got[0].Clip, got[1].Clip)
	}
	if !got[0].Palette.Dark || got[1].Palette.Dark {
		t.Error("each half did not take its own visual's tone")
	}
}

func TestASplitClockLinesUpAcrossTheSeam(t *testing.T) {
	palette := theme.Default()
	cfg := idled(func(c *config.Idle) { c.Size = config.SizeMedium })
	box := dashboard.Place(cfg.Idle.Position, cfg.Idle.Align, cfg.Idle.Size, 1920, 1200)
	pieces := []Piece{{Clip: ui.Rect{W: 960, H: 1200}, Palette: palette}, {Clip: ui.Rect{X: 960, W: 960, H: 1200}, Palette: palette}}

	whole := ui.NewImage(1920, 1200, palette.Background)
	face.Of(cfg.Idle.Face).Draw(whole, box, reading(cfg, noon), palette)
	split := ui.NewImage(1920, 1200, palette.Background)
	for _, p := range pieces {
		face.Of(cfg.Idle.Face).Draw(ui.Within(split, p.Clip), box, reading(cfg, noon), p.Palette)
	}
	for y := range 1200 {
		for x := range 1920 {
			if whole.At(x, y) != split.At(x, y) {
				t.Fatalf("the split clock differs at %d,%d", x, y)
			}
		}
	}
}

func TestALightThemeClockReadsOverADarkVisual(t *testing.T) {
	var light theme.Theme
	for _, th := range theme.All {
		if !th.Dark {
			light = th
			break
		}
	}
	slots := []config.IdleVisual{{Kind: string(visual.Orb), Source: config.SourceBoth}}
	cfg := idled(func(c *config.Idle) { c.First = slots[0] })
	img := ui.NewImage(1920, 1200, light.Background)
	Paint(img, cfg, slots, nil, light, noon)

	box := dashboard.Place(cfg.Idle.Position, cfg.Idle.Align, cfg.Idle.Size, 1920, 1200)
	brightest := 0.0
	for y := box.Y; y < box.Y+box.H; y += 2 {
		for x := box.X; x < box.X+box.W; x += 2 {
			c := img.At(x, y)
			brightest = max(brightest, (0.2126*float64(c.R)+0.7152*float64(c.G)+0.0722*float64(c.B))/255)
		}
	}
	if brightest < 0.8 {
		t.Errorf("the clock over the orb is no brighter than %.2f", brightest)
	}
}

func TestOnlyChosenVisualsAreDrawn(t *testing.T) {
	got := chosen(config.Idle{Second: config.IdleVisual{Kind: "orb"}})
	if len(got) != 1 || got[0].Kind != "orb" {
		t.Errorf("chosen %v", got)
	}
}

func TestNoneIsOfferedFirst(t *testing.T) {
	labels := KindLabels()
	if labels[0] != KindLabel("") || len(labels) != len(visual.Built())+1 {
		t.Errorf("labels %v", labels)
	}
	if k, ok := kindByLabel(KindLabel("orb")); !ok || k != "orb" {
		t.Errorf("orb by label: %q %v", k, ok)
	}
}

func TestRenderTheIdleScreen(t *testing.T) {
	dir := os.Getenv("LANOVO_RENDER")
	if dir == "" {
		t.Skip("set LANOVO_RENDER to a directory")
	}
	palette := theme.Default()
	for name, at := range map[string]struct {
		w, h  int
		slots []config.IdleVisual
		idle  func(*config.Idle)
	}{
		"idle-landscape-two": {1920, 1200,
			[]config.IdleVisual{{Kind: string(visual.ClassicVU), Source: config.SourceMic}, {Kind: string(visual.Orb), Source: config.SourceSpeaker}},
			func(c *config.Idle) {
				c.Position, c.Align, c.Size = config.PositionTop, config.AlignRight, config.SizeSmall
			}},
		"idle-portrait-one": {1200, 1920,
			[]config.IdleVisual{{Kind: string(visual.Orb), Source: config.SourceBoth}},
			func(c *config.Idle) {
				c.Position, c.Align, c.Size = config.PositionBottom, config.AlignLeft, config.SizeMedium
			}},
	} {
		img := ui.NewImage(at.w, at.h, palette.Background)
		Paint(img, idled(at.idle), at.slots, nil, palette, noon)
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		out := image.NewRGBA(image.Rect(0, 0, at.w, at.h))
		for y := range at.h {
			for x := range at.w {
				c := img.At(x, y)
				out.Set(x, y, color.RGBA{c.R, c.G, c.B, 255})
			}
		}
		if err := png.Encode(f, out); err != nil {
			t.Error(err)
		}
		f.Close()
	}
}

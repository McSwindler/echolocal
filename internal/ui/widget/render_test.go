package widget

import (
	"golang.org/x/exp/shiny/materialdesign/icons"

	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// Writes each screen out so the design can be looked at rather than reasoned about. Off by
// default: it is a darkroom, not a check.
func TestRenderScreens(t *testing.T) {
	dir := os.Getenv("ECHOLOCAL_RENDER")
	if dir == "" {
		t.Skip("set ECHOLOCAL_RENDER to a directory to write the screens out")
	}

	const w, h = 1920, 1200
	palette := theme.Default()
	m := New(w, h)

	pages := map[string]Page{
		"root": {Title: "Settings", Rows: []Row{
			{Label: "Display", Kind: Chevron, Value: "Midnight"},
			{Label: "Sound", Kind: Chevron, Value: "60%"},
			{Label: "Privacy", Kind: Chevron, Value: "On"},
			{Label: "Network", Kind: Chevron, Value: "Attic"},
			{Label: "About", Kind: Chevron, Value: "1.4.0"},
		}},
		"display": {Title: "Display", Rows: []Row{
			{Label: "Brightness", Kind: Slider, Level: 70},
			{Label: "Auto brightness", Kind: Toggle, On: true},
			{Label: "Theme", Kind: Chevron, Value: "Midnight"},
			{Label: "24 hour clock", Kind: Toggle, On: true},
		}},
		"volume": {Title: "Volume", Rows: []Row{
			{Label: "Media", Kind: Slider, Level: 60, Icon: icons.AVVolumeUp},
			{Label: "Alerts", Kind: Slider, Level: 0, Icon: icons.AVVolumeOff},
			{Label: "Voice", Kind: Slider, Level: 45, Icon: icons.AVVolumeDown},
			{Label: "Feedback", Kind: Slider, Level: 5, Icon: icons.AVVolumeDown},
		}},
		"theme": {Title: "Theme", Rows: []Row{
			{Label: "Midnight", Chosen: true},
			{Label: "Amethyst"},
			{Label: "Slate"},
			{Label: "Ember"},
		}},
	}

	for name, p := range pages {
		img := ui.NewImage(w, h, palette.Background)
		m.DrawPage(img, p, palette)
		write(t, dir+"/"+name+".png", img, w, h)
	}
}

func write(t *testing.T, path string, src *ui.Image, w, h int) {
	t.Helper()

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := src.At(x, y)
			out.Set(x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, out); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

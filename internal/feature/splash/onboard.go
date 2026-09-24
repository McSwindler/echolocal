package splash

import (
	"image"
	"image/color"
	"log/slog"

	"github.com/ygelfand/echolocal/internal/feature/api"
	"github.com/ygelfand/echolocal/internal/hardware/screen"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	uitheme "github.com/ygelfand/echolocal/internal/ui/theme"
)

// How the pairing side is laid out, as fractions of its shorter edge.
const (
	codeShare    = 0.70
	captionShare = 0.075
	captionGap   = 0.05
)

// addFlow starts the ESPHome flow on whichever Home Assistant the phone is signed in to. The
// redirect carries a domain and nothing else, so a key cannot travel this way — it is for a device
// Home Assistant provisions itself.
const addFlow = "https://my.home-assistant.io/redirect/config_flow_start/?domain=esphome"

// pairing is what the code carries.
func pairing() string {
	if key := api.Key(); key != "" {
		return key
	}
	return addFlow
}

// drawOnboard paints the mark beside the key to pair with.
//
// Black on white whatever the theme is: a scanner expects dark modules on a light field, and a
// device nobody has added yet is not worth losing a scan over.
func drawOnboard(p *screen.Panel, t uitheme.Theme, key string) {
	bg := colour(t.Background)
	p.Fill(bg)

	logo, list := split(p.Bounds())
	p.Draw(ui.Mark(t.Dark), inset(logo, min(logo.Dx(), logo.Dy())/8), bg)

	short := min(list.Dx(), list.Dy())

	code, err := ui.Code(key, int(float64(short)*codeShare), color.Black, color.White)
	if err != nil {
		slog.Error("the pairing code would not render", "err", err)
		return
	}

	caption := say.T("onboard.title")

	font := ui.MustLoad(ui.Medium, int(float64(short)*captionShare))
	gap := int(float64(short) * captionGap)
	width, height := font.Measure(caption)

	// The code and the line under it centred together, rather than the code centred with the line
	// hung off it.
	at := code.Bounds()
	tall := at.Dy() + gap + height
	top := list.Min.Y + (list.Dy()-tall)/2

	p.Blit(code, image.Pt(list.Min.X+(list.Dx()-at.Dx())/2, top), bg)

	ui.DrawText(ui.Of(p), font,
		list.Min.X+(list.Dx()-width)/2, top+at.Dy()+gap, t.Text, t.Background, caption)
}

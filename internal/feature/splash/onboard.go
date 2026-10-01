package splash

import (
	"image"
	"image/color"
	"log/slog"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/api"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/hardware/wifi"
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

// networked reports whether the device is on a network.
func networked() bool {
	if !component.Board().Has(board.Wifi) {
		return true
	}
	return wifi.Get().Network() != ""
}

// drawOnboard paints the mark beside the key to pair with. An empty key is a device with no network
// to be added over, which says so instead.
//
// Black on white whatever the theme is: a scanner expects dark modules on a light field, and a
// device nobody has added yet is not worth losing a scan over.
func drawOnboard(p *display.Panel, t uitheme.Theme, key string) {
	s := ui.Of(p)
	ui.Fill(s, t.Background)

	logo, list := split(bounds(p))
	ui.DrawLogo(s, box(inset(logo, min(logo.Dx(), logo.Dy())/8)), t.Background)

	short := min(list.Dx(), list.Dy())

	if key == "" {
		drawAsk(s, t, list, short)
		return
	}

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

	ui.DrawImageScaled(s, code, ui.Rect{X: list.Min.X + (list.Dx()-at.Dx())/2, Y: top, W: at.Dx(), H: at.Dy()}, t.Background)

	ui.DrawText(s, font,
		list.Min.X+(list.Dx()-width)/2, top+at.Dy()+gap, t.Text, t.Background, caption)
}

// askLines is how many lines the caption may run to.
const askLines = 3

// drawAsk is what a device with no network to be added over says.
func drawAsk(s ui.Surface, t uitheme.Theme, list image.Rectangle, short int) {
	font := ui.MustLoad(ui.Medium, int(float64(short)*captionShare))
	said := ui.Wrap(font, say.T("onboard.network"), list.Dx()-short/8, askLines)

	_, height := font.Measure("Ag")
	gap := int(float64(short) * captionGap)

	y := list.Min.Y + (list.Dy()-len(said)*(height+gap)+gap)/2
	for _, line := range said {
		width, _ := font.Measure(line)
		ui.DrawText(s, font,
			list.Min.X+(list.Dx()-width)/2, y, t.Text, t.Background, line)
		y += height + gap
	}
}

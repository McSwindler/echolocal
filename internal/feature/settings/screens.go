package settings

import (
	"strconv"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/privacy"
	"github.com/ygelfand/echolocal/internal/feature/screen"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/feature/volume"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

func open(v shell.View) func(int) { return func(int) { shell.Get().Push(v) } }

func percent(v int) string { return strconv.Itoa(v) + "%" }

type section struct {
	label string
	glyph string
	page  func() *shell.Page
}

var sections []section

// root is the top of the settings.
func root() *shell.Page {
	return &shell.Page{
		Title: say.T("settings.title"),
		Build: func() ([]widget.Row, []func(int)) {
			rows := []widget.Row{
				{Glyph: gogui.IconSunnyO, Label: say.T("settings.display"), Kind: widget.Chevron,
					Value: config.Get().Screen.Theme},
				{Glyph: gogui.IconSpeaker, Label: say.T("settings.volume"), Kind: widget.Chevron,
					Value: percent(volume.Get().Level(config.StreamMain))},
			}
			taps := []func(int){open(displayPage()), open(volume.Page())}
			for _, s := range sections {
				rows = append(rows, widget.Row{Glyph: s.glyph, Label: say.T(s.label), Kind: widget.Chevron})
				taps = append(taps, open(s.page()))
			}
			rows = append(rows,
				widget.Row{Glyph: gogui.IconPlug, Label: say.T("settings.features"), Kind: widget.Chevron, Value: featuresOn()},
				widget.Row{Glyph: gogui.IconCode, Label: say.T("settings.debug"), Kind: widget.Chevron},
			)
			taps = append(taps, open(featuresPage()), open(debugPage()))
			return rows, taps
		},
	}
}

// displayPage is the panel.
func displayPage() *shell.Page {
	return &shell.Page{
		Title: say.T("settings.display"),
		Build: func() ([]widget.Row, []func(int)) {
			cfg := config.Get()
			auto := cfg.Screen.Mode == config.ModeAuto

			return []widget.Row{
					{Label: say.T("settings.brightness"), Kind: widget.Slider, Level: cfg.Screen.Backlight},
					{Label: say.T("settings.brightness.auto"), Kind: widget.Toggle, On: auto},
					{Glyph: gogui.IconPalette, Label: say.T("settings.theme"), Kind: widget.Chevron, Value: cfg.Screen.Theme},
					{Glyph: gogui.IconClock, Label: say.T("settings.clock"), Kind: widget.Chevron, Value: cfg.Clock.Face.Label()},
					{Glyph: gogui.IconMoon, Label: say.T("settings.idle"), Kind: widget.Chevron, Value: cfg.Idle.After.Label()},
					{Label: say.T("settings.drawer"), Kind: widget.Chevron, Value: cfg.Screen.Drawer.Label()},
					{Label: say.T("settings.marks"), Kind: widget.Toggle, On: cfg.Screen.Marks},
				}, []func(int){
					func(level int) { screen.Get().SetBacklight(level) },
					func(int) { toggleAuto(auto) },
					open(themePage()),
					open(clockPage()),
					open(idlePage()),
					open(drawerPage()),
					func(int) { privacy.Get().SetMarks(!cfg.Screen.Marks) },
				}
		},
	}
}

func toggleAuto(auto bool) {
	want := config.ModeAuto
	if auto {
		want = config.ModeManual
	}
	screen.Get().SetMode(want)
}

// themePage is the palette.
func themePage() *shell.Page {
	return &shell.Page{
		Title: say.T("settings.theme"),

		Tiles: func() ([]widget.Cell, []func(int)) {
			now := screen.Get().Theme().Name

			cells := make([]widget.Cell, 0, len(theme.All))
			acts := make([]func(int), 0, len(theme.All))

			for _, t := range theme.All {
				cells = append(cells, widget.Cell{
					Label:   t.Name,
					Chosen:  t.Name == now,
					Paint:   swatch(t),
					Palette: &t,
				})
				acts = append(acts, use(t.Name))
			}
			return cells, acts
		},
	}
}

// How a swatch is laid out, as fractions of its height.
const (
	swatchPad = 0.12
	swatchGap = 0.07
)

// swatch draws a theme: its background as the card, with the rest as squares on it.
//
// Mean RGB distance between themes per role: Background 197, Surface 197, Text 172, Accent 158,
// Accent2 115, then Success 63, Warning 51, Muted 45, Danger 43.
func swatch(t theme.Theme) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, at ui.Rect, _ theme.Theme) {
		ui.FillRect(s, at, t.Background)

		// A light theme's card on a light page has no edge of its own.
		ui.Border(s, at, max(at.H/40, 1), t.Text.Blend(t.Background, 0.7))

		on := []theme.Color{t.Surface, t.Text, t.Accent, t.Accent2}

		pad := int(float64(at.H) * swatchPad)
		gap := int(float64(at.H) * swatchGap)

		wide := (at.W - pad*2 - gap*(len(on)-1)) / len(on)
		side := min(at.H-pad*2, wide)

		x := at.X + (at.W-side*len(on)-gap*(len(on)-1))/2
		y := at.Y + (at.H-side)/2

		for i, c := range on {
			ui.FillRect(s, ui.Rect{X: x + i*(side+gap), Y: y, W: side, H: side}, c)
		}
	}
}

func use(name string) func(int) {
	return func(int) {
		if err := config.Set().Screen().Theme(name); err != nil {
			return
		}
		screen.Get().Use(name)
	}
}

// drawerPage is which side the rail comes in from.
func drawerPage() *shell.Page {
	return &shell.Page{
		Title: say.T("settings.drawer"),
		Build: func() ([]widget.Row, []func(int)) {
			now := config.Get().Screen.Drawer

			edges := config.Edges()
			rows := make([]widget.Row, 0, len(edges))
			acts := make([]func(int), 0, len(edges))

			for _, e := range edges {
				rows = append(rows, widget.Row{Label: e.Label(), Kind: widget.Plain, Chosen: e == now})
				acts = append(acts, func(int) { screen.Get().SetDrawer(e) })
			}
			return rows, acts
		},
	}
}

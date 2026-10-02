package settings

import (
	"slices"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/feature/visuals"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/visual"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

func debugPage() *shell.Page {
	return &shell.Page{
		Title: say.T("debug.title"),
		Build: func() ([]widget.Row, []func(int)) {
			fps, seed := config.Get().Visual.MaxFPS, config.Get().Visual.Seed
			return []widget.Row{
					{Label: say.T("debug.visuals"), Kind: widget.Chevron},
					{Label: say.T("debug.fps"), Kind: widget.Slider, Level: fpsLevel(fps), Snap: fpsSnap, Value: fpsSays(fps)},
					{Label: say.T("debug.seed"), Kind: widget.Slider, Level: seedLevel(seed), Value: seedSays(seed)},
				}, []func(int){
					open(visualPage()),
					func(level int) { visuals.Get().SetMaxFPS(config.MaxFPSSteps[fpsIndex(level)]) },
					func(level int) { visuals.Get().SetSeed(seedOf(level)) },
				}
		},
	}
}

func fpsIndex(level int) int {
	n := len(config.MaxFPSSteps) - 1
	return min(max((level*n+50)/100, 0), n)
}

func fpsLevel(fps int) int {
	i := max(slices.Index(config.MaxFPSSteps, fps), 0)
	return i * 100 / (len(config.MaxFPSSteps) - 1)
}

func fpsSnap(level int) int { return fpsLevel(config.MaxFPSSteps[fpsIndex(level)]) }

func seedOf(level int) int {
	if level <= 0 {
		return 0
	}
	return max(1, (min(level, 100)*visual.SeedMost+50)/100)
}

func seedLevel(seed int) int {
	if seed <= 0 {
		return 0
	}
	return max(1, (seed*100+visual.SeedMost/2)/visual.SeedMost)
}

func seedSays(seed int) string {
	if seed <= 0 {
		return say.T("debug.seed.random")
	}
	return say.F("debug.seed.value", map[string]any{"N": seed})
}

func fpsSays(fps int) string { return say.F("debug.fps.value", map[string]any{"N": fps}) }

func visualPage() *shell.Page {
	return visualPicker(say.T("debug.visuals"), nil, nil, func(k visual.Kind) {
		visuals.Get().SetKind(k)
		shell.Get().Push(visuals.Get().View())
	})
}

func visualPicker(title string, none func(), chosen func() string, pick func(visual.Kind)) *shell.Page {
	return &shell.Page{
		Title: title,
		Tiles: func() ([]widget.Cell, []func(int)) {
			now := ""
			if chosen != nil {
				now = chosen()
			}
			var cells []widget.Cell
			var acts []func(int)
			if none != nil {
				cells = append(cells, widget.Cell{Label: say.T("idle.none"), Chosen: chosen != nil && now == "", Paint: noneTile})
				acts = append(acts, func(int) { none() })
			}
			for _, k := range visual.Built() {
				cells = append(cells, widget.Cell{Label: k.Label(), Chosen: chosen != nil && string(k) == now, Paint: thumbTile(k)})
				acts = append(acts, func(int) { pick(k) })
			}
			return cells, acts
		},
	}
}

func noneTile(s ui.Surface, at ui.Rect, palette theme.Theme) { ui.FillRect(s, at, palette.Surface) }

func thumbTile(k visual.Kind) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		img := visual.ThumbnailFit(k, at.W, at.H)
		if img == nil {
			ui.FillRect(s, at, palette.Surface)
			return
		}
		ui.DrawRGBA(s, at.X, at.Y, img, 1, ui.ClipOf(s))
	}
}

package settings

import (
	"time"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/clock"
	"github.com/ygelfand/echolocal/internal/feature/clock/face"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

func clockPage() *shell.Page {
	return &shell.Page{
		Title: say.T("clock.title"),
		Build: func() ([]widget.Row, []func(int)) {
			cfg := config.Get()

			return []widget.Row{
					{Label: say.T("clock.face"), Kind: widget.Chevron, Value: cfg.Clock.Face.Label()},
					{Label: say.T("clock.position"), Kind: widget.Chevron, Value: cfg.Clock.Position.Label()},
					{Label: say.T("clock.size"), Kind: widget.Chevron, Value: cfg.Clock.Size.Label()},
					{Label: say.T("clock.color"), Kind: widget.Chevron, Value: cfg.Clock.Ink.Label()},
					{Label: say.T("clock.date"), Kind: widget.Toggle, On: cfg.Clock.Date},
					{Label: say.T("clock.hours"), Kind: widget.Toggle, On: cfg.Screen.Hours != config.TwelveHour},
					{Label: say.T("clock.logo"), Kind: widget.Toggle, On: cfg.Screen.Logo},
				}, []func(int){
					open(facePage()),
					open(positionPage()),
					open(sizePage()),
					open(colorPage()),
					func(int) { clock.Get().SetDate(!cfg.Clock.Date) },
					func(int) { toggleHours(cfg.Screen.Hours) },
					func(int) { clock.Get().SetLogo(!cfg.Screen.Logo) },
				}
		},
	}
}

func toggleHours(now config.HourFormat) {
	if now == config.TwelveHour {
		clock.Get().SetHours(config.TwentyFourHour)
		return
	}
	clock.Get().SetHours(config.TwelveHour)
}

func facePage() *shell.Page {
	return picker(say.T("clock.face"), config.Faces(),
		func() config.Face { return config.Get().Clock.Face }, preview, clock.Get().SetFace)
}

func positionPage() *shell.Page {
	return picker(say.T("clock.position.title"), config.Positions(),
		func() config.Position { return config.Get().Clock.Position }, placing, clock.Get().SetPosition)
}

func sizePage() *shell.Page {
	return picker(say.T("clock.size.title"), config.Sizes(),
		func() config.Size { return config.Get().Clock.Size }, sizing, clock.Get().SetSize)
}

func colorPage() *shell.Page {
	return picker(say.T("clock.color.title"), config.Inks(),
		func() config.Ink { return config.Get().Clock.Ink }, inking, clock.Get().SetInk)
}

type drawing = func(ui.Surface, ui.Rect, theme.Theme)

func picker[T interface {
	comparable
	config.Labelled
}](title string, values []T, now func() T, draw func(T) drawing, use func(T)) *shell.Page {
	return &shell.Page{
		Title: title,
		Build: func() ([]widget.Row, []func(int)) {
			chosen := now()

			rows := make([]widget.Row, 0, len(values))
			acts := make([]func(int), 0, len(values))
			for _, v := range values {
				row := widget.Row{Label: v.Label(), Chosen: v == chosen}
				if draw != nil {
					row.Preview = draw(v)
				}
				rows = append(rows, row)
				acts = append(acts, func(int) { use(v) })
			}
			return rows, acts
		},
	}
}

func reading() face.Reading {
	return face.Read(time.Now(), config.Get().Screen.Hours != config.TwelveHour).Undated()
}

func preview(f config.Face) drawing {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		face.Of(f).Draw(s, at, reading(), config.Get().Clock.Ink.Over(palette))
	}
}

func placing(at config.Position) drawing {
	return func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		laid(s, box, palette, at, config.Get().Clock.Size)
	}
}

func sizing(size config.Size) drawing {
	return func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		laid(s, box, palette, config.Get().Clock.Position, size)
	}
}

func laid(s ui.Surface, box ui.Rect, palette theme.Theme, at config.Position, size config.Size) {
	cfg := config.Get()
	within := clock.Box(at, size, box.W, box.H)
	within.X += box.X
	within.Y += box.Y
	face.Of(cfg.Clock.Face).Draw(s, within, reading(), cfg.Clock.Ink.Over(palette))
}

func inking(ink config.Ink) drawing {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		face.Of(config.Get().Clock.Face).Draw(s, at, reading(), ink.Over(palette))
	}
}

package settings

import (
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/dashboard"
	"github.com/ygelfand/echolocal/internal/feature/dashboard/face"
	"github.com/ygelfand/echolocal/internal/feature/idle"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/visual"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

func idlePage() *shell.Page {
	return &shell.Page{
		Title: say.T("idle.title"),
		Build: func() ([]widget.Row, []func(int)) {
			c := config.Get().Idle
			i := idle.Get()
			return []widget.Row{
					{Label: say.T("idle.after"), Kind: widget.Chevron, Value: c.After.Label()},
					{Label: say.T("idle.visual1"), Kind: widget.Chevron, Value: idle.KindLabel(c.First.Kind)},
					{Label: say.T("idle.source1"), Kind: widget.Chevron, Value: c.First.Source.Label()},
					{Label: say.T("idle.visual2"), Kind: widget.Chevron, Value: idle.KindLabel(c.Second.Kind)},
					{Label: say.T("idle.source2"), Kind: widget.Chevron, Value: c.Second.Source.Label()},
					{Label: say.T("idle.face"), Kind: widget.Chevron, Value: c.Face.Label()},
					{Label: say.T("idle.vertical"), Kind: widget.Chevron, Value: c.Position.Label()},
					{Label: say.T("idle.horizontal"), Kind: widget.Chevron, Value: c.Align.Label()},
					{Label: say.T("idle.size"), Kind: widget.Chevron, Value: c.Size.Label()},
				}, []func(int){
					open(picker(say.T("idle.after.title"), config.Delays(),
						func() config.Delay { return config.Get().Idle.After }, nil, i.SetAfter)),
					open(idleVisualPage(0)),
					open(idleSourcePage(0)),
					open(idleVisualPage(1)),
					open(idleSourcePage(1)),
					open(picker(say.T("idle.face"), config.IdleFaces(),
						func() config.Face { return config.Get().Idle.Face }, idleFacePreview, i.SetFace)),
					open(picker(say.T("idle.vertical"), config.Positions(),
						func() config.Position { return config.Get().Idle.Position },
						func(p config.Position) drawing { return idlePlaced(func(c *config.Idle) { c.Position = p }) },
						i.SetPosition)),
					open(picker(say.T("idle.horizontal"), config.Aligns(),
						func() config.Align { return config.Get().Idle.Align },
						func(a config.Align) drawing { return idlePlaced(func(c *config.Idle) { c.Align = a }) },
						i.SetAlign)),
					open(picker(say.T("idle.size"), config.Sizes(),
						func() config.Size { return config.Get().Idle.Size },
						func(s config.Size) drawing { return idlePlaced(func(c *config.Idle) { c.Size = s }) },
						i.SetSize)),
				}
		},
	}
}

func slotVisual(slot int) config.IdleVisual {
	if slot == 0 {
		return config.Get().Idle.First
	}
	return config.Get().Idle.Second
}

func idleVisualPage(slot int) *shell.Page {
	title := say.T("idle.visual1")
	if slot == 1 {
		title = say.T("idle.visual2")
	}
	return visualPicker(title,
		func() { idle.Get().SetKind(slot, "") },
		func() string { return slotVisual(slot).Kind },
		func(k visual.Kind) { idle.Get().SetKind(slot, string(k)) })
}

func idleSourcePage(slot int) *shell.Page {
	title := say.T("idle.source1")
	if slot == 1 {
		title = say.T("idle.source2")
	}
	return picker(title, config.Sources(),
		func() config.Source { return slotVisual(slot).Source }, nil,
		func(s config.Source) { idle.Get().SetSource(slot, s) })
}

func idleFacePreview(f config.Face) drawing {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		if f == config.FaceNone {
			return
		}
		face.Of(f).Draw(s, at, reading(), config.Get().Clock.Ink.Over(palette))
	}
}

func idlePlaced(change func(*config.Idle)) drawing {
	return func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		cfg := config.Get()
		c := cfg.Idle
		change(&c)
		if c.Face == config.FaceNone {
			return
		}
		within := dashboard.Place(c.Position, c.Align, c.Size, box.W, box.H)
		within.X += box.X
		within.Y += box.Y
		face.Of(c.Face).Draw(s, within, reading(), cfg.Clock.Ink.Over(palette))
	}
}

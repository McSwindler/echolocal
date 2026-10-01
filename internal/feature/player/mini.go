package player

import (
	"fmt"
	"sync"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	apptheme "github.com/ygelfand/echolocal/internal/feature/theme"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/hardware/touch"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

// mini is what is playing, kept small along the bottom of the clock once the card has been left.
type mini struct {
	p *Player

	mu    sync.Mutex
	claim *display.Claim
	shown bool
	bar   ui.Rect
	hits  []miniHit
	drawn string
}

type miniHit struct {
	at ui.Rect
	do func()
}

func (m *mini) wanted(now media.Now) bool {
	return (now.Playing || now.Paused) && !shell.Get().Open()
}

func (m *mini) follow() {
	now := media.Get().Now()
	if !m.wanted(now) {
		m.hide()
		return
	}
	key := fmt.Sprintf("%s|%s|%v|%v|%d|%v|%d", now.Title, now.Artist, now.Playing, now.Paused, now.Can, media.Get().From(), now.ArtID)
	m.mu.Lock()
	if m.claim == nil {
		m.claim = display.Get().Overlay(display.PrioritySetup)
	}
	if m.shown && key == m.drawn {
		m.mu.Unlock()
		return
	}
	m.shown, m.drawn = true, key
	claim := m.claim
	m.mu.Unlock()

	palette := apptheme.Get().Current()
	claim.Show(func(p *display.Panel) error {
		m.draw(ui.Of(p), now, palette)
		return nil
	})
}

func (m *mini) hide() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.shown {
		return
	}
	m.shown, m.drawn, m.hits = false, "", nil
	m.claim.Clear()
}

func (m *mini) draw(s ui.Surface, now media.Now, palette theme.Theme) {
	w, h := s.Size()
	mt := widget.New(w, h)
	height := mt.Row * 3 / 2
	bar := ui.Rect{X: mt.Pad * 2, Y: h - height - mt.Pad*2, W: w - mt.Pad*4, H: height}
	ui.FillRounded(s, bar, height/4, palette.Surface)

	pad := height / 5
	mark := ui.Rect{X: bar.X + pad, Y: bar.Y + pad, W: height - pad*2, H: height - pad*2}
	if art := cover(now); art != nil {
		ui.DrawScaled(s, art, mark)
	} else {
		ui.DrawIcon(s, kindIcon(media.Get().From()), mark.Inset(mark.W/6), palette.Muted, palette.Surface)
	}

	type control struct {
		icon ui.Icon
		do   func()
	}
	middle := control{icons.AVPlayArrow, func() { transport().Play() }}
	if now.Playing && !now.Paused {
		middle = control{icons.AVPause, func() { transport().Pause() }}
	}
	controls := []control{middle}
	if now.Can.Has(media.CanNext) {
		controls = append(controls, control{icons.AVSkipNext, func() { transport().Next() }})
	}
	controls = append(controls, control{icons.NavigationFullscreen, func() {
		m.hide()
		m.p.Open()
	}})
	side := height - pad*2
	x := bar.X + bar.W - pad
	var hits []miniHit
	for i := len(controls) - 1; i >= 0; i-- {
		x -= side
		at := ui.Rect{X: x, Y: bar.Y + pad, W: side, H: side}
		ui.DrawIcon(s, controls[i].icon, at.Inset(side/8), palette.Text, palette.Surface)
		hits = append(hits, miniHit{at: at.Inset(-pad / 2), do: controls[i].do})
		x -= pad
	}

	textX := mark.X + mark.W + pad
	room := x - textX - pad
	_, th := mt.Label.Measure("Ag")
	_, sh := mt.Hint.Measure("Ag")
	top := bar.Y + (height-th-sh)/2
	ui.DrawText(s, mt.Label, textX, top, palette.Text, palette.Surface, fit(mt.Label, heading(now), room))
	if under := now.Artist; under != "" {
		ui.DrawText(s, mt.Hint, textX, top+th, palette.Muted, palette.Surface, fit(mt.Hint, under, room))
	}

	m.mu.Lock()
	m.bar, m.hits = bar, hits
	m.mu.Unlock()
}

func (m *mini) on(g touch.Gesture) {
	if g.Kind != touch.Tap || shell.Get().Open() {
		return
	}
	m.mu.Lock()
	shown, bar, hits := m.shown, m.bar, m.hits
	m.mu.Unlock()
	if !shown || !bar.Contains(g.EndX, g.EndY) {
		return
	}
	for _, h := range hits {
		if h.at.Contains(g.EndX, g.EndY) {
			h.do()
			m.p.refresh()
			return
		}
	}
}

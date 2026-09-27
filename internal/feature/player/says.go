package player

import (
	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

// lines is what the card says beside the artwork, broken to the width it has.
type lines struct {
	head, under *ui.Font
	gap         int

	title   []string
	credits []string
}

// says breaks what is playing to a width.
func says(m widget.Metrics, now media.Now, width int) lines {
	return lines{
		head:    m.Title,
		under:   m.Value,
		gap:     m.Pad / 2,
		title:   ui.Wrap(m.Title, heading(now), width, titleLines),
		credits: credits(m.Value, now, width),
	}
}

func (l lines) height() int {
	_, th := l.head.Measure("Ag")
	_, uh := l.under.Measure("Ag")
	return len(l.title)*th + l.gap + len(l.credits)*uh
}

// draw puts them in what they were measured for.
func (l lines) draw(s ui.Surface, at ui.Rect, palette theme.Theme) {
	if at.W <= 0 || at.H <= 0 {
		return
	}

	_, th := l.head.Measure("Ag")
	_, uh := l.under.Measure("Ag")

	y := at.Y
	for _, said := range l.title {
		ui.DrawText(s, l.head, at.X, y, palette.Text, palette.Background, said)
		y += th
	}
	y += l.gap

	for _, said := range l.credits {
		ui.DrawText(s, l.under, at.X, y, palette.Text.Blend(palette.Muted, 0.4), palette.Background, said)
		y += uh
	}
}

// credits is the artist and the album on one line, and on one each where that will not fit.
func credits(f *ui.Font, now media.Now, width int) []string {
	switch {
	case now.Artist == "" && now.Album == "":
		return nil
	case now.Artist == "":
		return ui.Wrap(f, now.Album, width, creditLines)
	case now.Album == "":
		return ui.Wrap(f, now.Artist, width, creditLines)
	}

	both := now.Artist + " · " + now.Album
	if w, _ := f.Measure(both); w <= width {
		return []string{both}
	}
	return []string{ui.Fit(f, now.Artist, width), ui.Fit(f, now.Album, width)}
}

func heading(now media.Now) string {
	switch {
	case now.Title != "":
		return now.Title
	case now.Paused:
		return say.T("player.paused")
	case now.Playing:
		return say.T("player.playing")
	}
	return say.T("player.idle")
}

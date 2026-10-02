package splash

import (
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/feature/theme"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
)

const (
	strandedMarkShare = 0.34
	strandedSignShare = 0.28
	strandedWordShare = 0.045
)

func init() { display.Get().Stranded(drawStranded) }

func drawStranded(p *display.Panel) error {
	s := ui.Of(p)
	w, h := s.Size()
	t := theme.Get().Current()
	ui.Fill(s, t.Background)

	short := min(w, h)
	mark := int(float64(short) * strandedMarkShare)
	sign := int(float64(short) * strandedSignShare)

	word := say.T("boot.stranded")
	font := ui.MustLoad(ui.Medium, int(float64(short)*strandedWordShare))
	wordW, wordH := font.Measure(word)

	gap := wordH
	top := (h - (mark + gap + sign + gap + wordH)) / 2

	ui.DrawLogo(s, ui.Rect{X: (w - mark) / 2, Y: top, W: mark, H: mark}, t.Background)
	ui.DrawIcon(s, icons.AlertWarning, ui.Rect{X: (w - sign) / 2, Y: top + mark + gap, W: sign, H: sign}, t.Danger, t.Background)
	ui.DrawText(s, font, (w-wordW)/2, top+mark+gap+sign+gap, t.Text, t.Background, word)
	return nil
}

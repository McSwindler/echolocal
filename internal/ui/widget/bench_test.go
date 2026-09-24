package widget

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

func benchPage() Page {
	return Page{Title: "Volume", Rows: []Row{
		{Label: "Media", Kind: Slider, Level: 60},
		{Label: "Alerts", Kind: Slider, Level: 80},
		{Label: "Voice", Kind: Slider, Level: 45},
		{Label: "Feedback", Kind: Slider, Level: 20},
	}}
}

func BenchmarkDrawPage(b *testing.B) {
	palette := theme.Default()
	img := ui.NewImage(screenW, screenH, palette.Background)
	m := New(screenW, screenH)
	p := benchPage()

	b.ResetTimer()
	for b.Loop() {
		m.DrawPage(img, p, palette)
	}
}

func BenchmarkFillTheScreen(b *testing.B) {
	palette := theme.Default()
	img := ui.NewImage(screenW, screenH, palette.Background)

	b.ResetTimer()
	for b.Loop() {
		ui.Fill(img, palette.Background)
	}
}

// One row, which is all a drag actually changes.
func BenchmarkDrawOneRow(b *testing.B) {
	palette := theme.Default()
	img := ui.NewImage(screenW, screenH, palette.Background)
	m := New(screenW, screenH)

	at := m.Rows(m.Body(ui.Rect{W: screenW, H: screenH}), 4)[0]
	row := Row{Label: "Media", Kind: Slider, Level: 60}

	b.ResetTimer()
	for b.Loop() {
		ui.FillRect(img, at, palette.Background)
		m.Draw(img, at, row, palette, palette.Background)
	}
}

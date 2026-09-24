package widget

import (
	"testing"
	"unicode/utf8"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// A row draws inside the rectangle it is handed, and this is load bearing today rather than tidiness.
//
// shell.Page.Damaged reports one row's rectangle while a slider is being dragged, and the shell
// repaints only that. Anything a row paints outside its own rectangle during a drag is a mark
// nothing repaints over until the whole page is drawn again.
func TestARowStaysInsideTheRectangleItIsGiven(t *testing.T) {
	palette := theme.Default()

	// Room on all four sides, so an overrun lands where it can be seen.
	const pad = 40
	m := New(1200, 1920)
	at := ui.Rect{X: pad, Y: pad, W: 1200 - 2*pad, H: m.Row}

	w, h := at.X+at.W+pad, at.Y+at.H+pad

	for _, tc := range []struct {
		name string
		row  Row
	}{
		{"plain", Row{Label: "Wi-Fi", Value: "Home", Kind: Plain}},
		{"chevron", Row{Label: "Power", Kind: Chevron, Icon: icons.ActionPowerSettingsNew}},
		{"toggle on", Row{Label: "Privacy marks", Kind: Toggle, On: true}},
		{"toggle off", Row{Label: "Privacy marks", Kind: Toggle}},
		{"slider 0", Row{Label: "Media", Kind: Slider}},
		{"slider 50", Row{Label: "Media", Kind: Slider, Level: 50}},
		{"slider 100", Row{Label: "Media", Kind: Slider, Level: 100}},
		{"hint", Row{Label: "Sleep", Hint: "after a minute with nobody there", Kind: Chevron}},
		{"chosen", Row{Label: "Ember", Kind: Plain, Chosen: true}},
		{"slider with hint", Row{Label: "Media", Hint: "how loud", Kind: Slider, Level: 40}},
		{"slider with icon", Row{Label: "Media", Kind: Slider, Level: 40, Icon: icons.AVVolumeUp}},
		{"level out of range", Row{Label: "Media", Kind: Slider, Level: 140}},
		{"negative level", Row{Label: "Media", Kind: Slider, Level: -20}},
	} {
		blank := ui.NewImage(w, h, palette.Background)
		img := ui.NewImage(w, h, palette.Background)
		ui.Fill(img, palette.Background)

		m.Draw(img, at, tc.row, palette, palette.Background)

		if painted := ui.Changed(blank, img); !at.Holds(painted) {
			t.Errorf("%s painted %+v, outside its row %+v", tc.name, painted, at)
		}
	}
}

// A row with something to draw for itself is handed a box, and a preview that overran it would land
// on the rows either side.
func TestARowPreviewStaysInsideItsRow(t *testing.T) {
	palette := theme.Default()

	const pad = 40
	m := New(1200, 1920)
	at := ui.Rect{X: pad, Y: pad, W: 1200 - 2*pad, H: m.Row}
	w, h := at.X+at.W+pad, at.Y+at.H+pad

	// A preview that tries to paint well past whatever it is given.
	greedy := func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		ui.FillRect(s, box, palette.Accent)
	}

	blank := ui.NewImage(w, h, palette.Background)
	img := ui.NewImage(w, h, palette.Background)
	ui.Fill(img, palette.Background)

	m.Draw(img, at, Row{Label: "Cards", Kind: Plain, Preview: greedy}, palette, palette.Background)

	if painted := ui.Changed(blank, img); !at.Holds(painted) {
		t.Errorf("a row with a preview painted %+v, outside its row %+v", painted, at)
	}
}

// Rows are stretched to fill a short list and squeezed to fit a long one, so a row is handed
// anything from its natural height to half again, and may not spill onto its neighbours at any of
// them.
func TestARowStaysInsideItselfAtEveryHeight(t *testing.T) {
	palette := theme.Default()

	const pad = 40
	m := New(1200, 1920)

	for _, tall := range []int{m.Row, m.Row * 3 / 2, m.Row * 3 / 4} {
		at := ui.Rect{X: pad, Y: pad, W: 1200 - 2*pad, H: tall}
		w, h := at.X+at.W+pad, at.Y+at.H+pad

		blank := ui.NewImage(w, h, palette.Background)
		img := ui.NewImage(w, h, palette.Background)
		ui.Fill(img, palette.Background)

		m.Draw(img, at, Row{Label: "Media", Hint: "how loud", Kind: Slider, Level: 40},
			palette, palette.Background)

		if painted := ui.Changed(blank, img); !at.Holds(painted) {
			t.Errorf("a row %d tall painted %+v, outside %+v", tall, painted, at)
		}
	}
}

// Text longer than the row it is in.
//
// Not our own strings: those are short and stay short. A value is data — a network name, a time
// zone, whatever the device was called at install — and a label becomes data the moment it is
// translated.
func TestALongRowStaysInsideTheRectangleItIsGiven(t *testing.T) {
	palette := theme.Default()

	const pad = 40
	m := New(1200, 1920)
	at := ui.Rect{X: pad, Y: pad, W: 1200 - 2*pad, H: m.Row}

	w, h := at.X+at.W+pad, at.Y+at.H+pad

	// Long enough that no font size brings it back inside the row.
	const long = "an enormously long piece of text that no row was ever going to hold, " +
		"and here is as much of it again so there is no argument about whether it fits"

	for _, tc := range []struct {
		name string
		row  Row
	}{
		{"long label", Row{Label: long, Kind: Plain}},
		{"long hint", Row{Label: "Sleep", Hint: long, Kind: Chevron}},
		{"long value, plain", Row{Label: "Network", Value: long, Kind: Plain}},
		{"long value, chevron", Row{Label: "Network", Value: long, Kind: Chevron}},
		{"long both", Row{Label: long, Value: long, Kind: Plain}},
		{"long everything", Row{Label: long, Hint: long, Value: long, Kind: Chevron,
			Icon: icons.DeviceNetworkWiFi}},
		{"long on a slider", Row{Label: long, Hint: long, Kind: Slider, Level: 40}},
		{"long and chosen", Row{Label: long, Kind: Plain, Chosen: true}},
	} {
		blank := ui.NewImage(w, h, palette.Background)
		img := ui.NewImage(w, h, palette.Background)
		ui.Fill(img, palette.Background)

		m.Draw(img, at, tc.row, palette, palette.Background)

		if painted := ui.Changed(blank, img); !at.Holds(painted) {
			t.Errorf("%s painted %+v, outside its row %+v", tc.name, painted, at)
		}
	}
}

// A value keeps room even beside a label that wants the whole row, since the value is the part that
// changes and the label is the part somebody already read.
func TestALongLabelDoesNotSqueezeTheValueOut(t *testing.T) {
	m := New(1200, 1920)
	words := ui.Rect{W: 1000}

	const long = "an enormously long piece of text that no row was ever going to hold"

	if got := m.room(words, Row{Label: long}); got < words.W/leastValue {
		t.Errorf("a value beside a long label got %d of %d", got, words.W)
	}

	// And a short label leaves it more than that floor.
	short := m.room(words, Row{Label: "Network"})
	if short <= words.W/leastValue {
		t.Errorf("a value beside a short label got only %d of %d", short, words.W)
	}
}

// Cut by rune. Cutting bytes splits a multi-byte character into something that is not one.
func TestFittingDoesNotSplitACharacter(t *testing.T) {
	m := New(1200, 1920)

	for _, s := range []string{"ümlaut everywhere ünd ëverywhere", "日本語のとても長い文字列です", "🎵🎵🎵🎵🎵🎵🎵🎵🎵🎵"} {
		for _, width := range []int{0, 1, 10, 40, 120, 400} {
			got := fit(m.Label, s, width)

			if !utf8.ValidString(got) {
				t.Errorf("fitting %q to %d gave bytes that are not text: %q", s, width, got)
			}
			if w, _ := m.Label.Measure(got); w > width {
				t.Errorf("fitting %q to %d gave %q, which is %d wide", s, width, got, w)
			}
		}
	}
}

// What fits is left alone, rather than being cut one character short for the sake of it.
func TestWhatFitsIsNotCut(t *testing.T) {
	m := New(1200, 1920)

	if got := fit(m.Label, "Network", 1000); got != "Network" {
		t.Errorf("a label with room to spare came back as %q", got)
	}
	if got := fit(m.Label, "", 1000); got != "" {
		t.Errorf("an empty label came back as %q", got)
	}
}

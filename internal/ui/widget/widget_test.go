package widget

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// The picture as this device shows it.
const (
	screenW = 1920
	screenH = 1200
)

func body() (Metrics, ui.Rect) {
	m := New(screenW, screenH)
	return m, m.Body(ui.Rect{W: screenW, H: screenH})
}

func TestRowsStackWithoutOverlapping(t *testing.T) {
	m, in := body()

	rows := m.Rows(in, 5)
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}
	for i, at := range rows {
		if at.X != in.X || at.W != in.W {
			t.Errorf("row %d is not the width of the body: %+v", i, at)
		}
		if i > 0 && at.Y < rows[i-1].Y+rows[i-1].H {
			t.Errorf("row %d overlaps the one before it", i)
		}
	}
}

// A short list spreads out rather than huddling at the top of an otherwise empty screen.
func TestAShortListFillsTheScreen(t *testing.T) {
	m, in := body()

	rows := m.Rows(in, 4)
	used := rows[3].Y + rows[3].H - in.Y

	if used < in.H/2 {
		t.Errorf("four rows use %d of %d: the list is huddled at the top", used, in.H)
	}
}

// But only so far. Past that a list stops reading as a list.
func TestRowsAreNotStretchedWithoutLimit(t *testing.T) {
	m, in := body()

	one := m.Rows(in, 1)[0]
	if one.H > int(float64(m.Row)*looseShare)+1 {
		t.Errorf("a lone row is %d tall against a natural %d", one.H, m.Row)
	}
}

// A long list keeps its natural height and runs off the bottom, which Fit is how a caller notices.
func TestALongListKeepsItsRowHeight(t *testing.T) {
	m, in := body()

	rows := m.Rows(in, 40)
	if rows[0].H != m.Row {
		t.Errorf("row height is %d, want the natural %d", rows[0].H, m.Row)
	}
	if fits := m.Fit(in); fits >= 40 {
		t.Errorf("Fit says %d of 40 rows fit, which cannot be right", fits)
	}
}

func TestNoRows(t *testing.T) {
	m, in := body()
	if got := m.Rows(in, 0); got != nil {
		t.Errorf("Rows(0) = %v, want nothing", got)
	}
}

func TestLevelFollowsTheTrack(t *testing.T) {
	m, in := body()
	at := m.Rows(in, 4)[0]
	track := m.Track(at)

	tests := []struct {
		name string
		x    int
		want int
	}{
		{"the left end", track.X, 0},
		{"the middle", track.X + track.W/2, 50},
		{"the right end", track.X + track.W, 100},
		{"off the left", track.X - 500, 0},
		{"off the right", track.X + track.W + 500, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.Level(at, tt.x)
			if d := got - tt.want; d > 1 || d < -1 {
				t.Errorf("Level at %s = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestTheTrackStaysInsideItsRow(t *testing.T) {
	m, in := body()

	for i, at := range m.Rows(in, 4) {
		track := m.Track(at)
		if track.X < at.X || track.X+track.W > at.X+at.W {
			t.Errorf("row %d: the track runs outside it horizontally: %+v in %+v", i, track, at)
		}
		if track.Y < at.Y || track.Y+track.H > at.Y+at.H {
			t.Errorf("row %d: the track runs outside it vertically: %+v in %+v", i, track, at)
		}
	}
}

// Every row is drawn inside the rectangle it was given. A row that painted over its neighbor
// would only show up as the wrong thing under a finger.
func TestARowStaysInItsOwnRectangle(t *testing.T) {
	palette := theme.Default()
	m, in := body()

	rows := []Row{
		{Label: "Brightness", Kind: Slider, Level: 70},
		{Label: "Auto brightness", Kind: Toggle, On: true},
		{Label: "Theme", Kind: Chevron, Value: "Midnight"},
		{Label: "Midnight", Chosen: true},
	}

	for i, r := range rows {
		img := ui.NewImage(screenW, screenH, palette.Background)
		at := m.Rows(in, len(rows))[i]
		m.Draw(img, at, r, palette, palette.Background)

		for y := range screenH {
			for x := range screenW {
				if at.Contains(x, y) {
					continue
				}
				if img.At(x, y) != palette.Background {
					t.Fatalf("row %d (%q) painted at %d,%d, outside %+v", i, r.Label, x, y, at)
				}
			}
		}
	}
}

// A slider with no level still shows a track, so an empty row does not look like a missing one.
func TestASliderAtZeroStillDrawsItsTrack(t *testing.T) {
	palette := theme.Default()
	m, in := body()

	img := ui.NewImage(screenW, screenH, palette.Background)
	at := m.Rows(in, 1)[0]
	m.Draw(img, at, Row{Label: "Media", Kind: Slider, Level: 0}, palette, palette.Background)

	track := m.Track(at)
	x, y := track.Center()
	if img.At(x, y) == palette.Background {
		t.Error("an empty slider drew no track")
	}
}

func TestDrawInEveryTheme(t *testing.T) {
	rows := []Row{
		{Label: "Brightness", Kind: Slider, Level: 40},
		{Label: "Auto brightness", Kind: Toggle},
		{Label: "Theme", Kind: Chevron, Value: "Ember"},
		{Label: "Address", Kind: Plain, Value: "10.0.0.4"},
	}

	for _, palette := range theme.All {
		img := ui.NewImage(screenW, screenH, palette.Background)
		New(screenW, screenH).DrawPage(img, Page{Title: "Display", Rows: rows}, palette)
	}
}

// Portrait is the same screen turned, and everything is measured off the shorter side, so the type
// should not change size when the device is stood up.
func TestTypeIsTheSameInEitherOrientation(t *testing.T) {
	land, port := New(screenW, screenH), New(screenH, screenW)

	if land.Row != port.Row {
		t.Errorf("row height is %d landscape and %d portrait", land.Row, port.Row)
	}
	if land.Icon != port.Icon {
		t.Errorf("icon size is %d landscape and %d portrait", land.Icon, port.Icon)
	}
}

// Every page came from somewhere, so every page has the mark that says so. A page that drew no
// arrow would be one whose header does nothing when it is touched.
func TestEveryPageDrawsTheArrow(t *testing.T) {
	palette := theme.Default()
	m := New(screenW, screenH)

	img := ui.NewImage(screenW, screenH, palette.Background)
	m.DrawPage(img, Page{Title: "Settings", Rows: []Row{{Label: "Display", Kind: Chevron}}}, palette)

	arrow := m.Arrow(ui.Rect{W: screenW, H: screenH})

	var on int
	for y := arrow.Y; y < arrow.Y+arrow.H; y++ {
		for x := arrow.X; x < arrow.X+arrow.W; x++ {
			if x >= 0 && img.At(x, y) != palette.Background {
				on++
			}
		}
	}
	if on == 0 {
		t.Error("nothing was drawn where the back arrow goes")
	}
}

// The rows start below the header, or the first one sits under the title.
func TestTheBodyClearsTheHeader(t *testing.T) {
	m := New(screenW, screenH)
	in := ui.Rect{W: screenW, H: screenH}

	head, body := m.Header(in), m.Body(in)
	if body.Y < head.Y+head.H {
		t.Errorf("the body starts at %d, inside a header ending at %d", body.Y, head.Y+head.H)
	}
}

// A finger aims at the bar, and the whole row is what the touch is resolved against, so the bar
// belongs in the middle of it. Off-center means reaching for one slider and moving another.
func TestTheTrackIsCentredInItsRow(t *testing.T) {
	m, in := body()

	for i, at := range m.Rows(in, 4) {
		track := m.Track(at)

		above := track.Y + track.H/2 - at.Y
		below := at.Y + at.H - (track.Y + track.H/2)

		if d := above - below; d > 1 || d < -1 {
			t.Errorf("row %d: %d of live area above the bar and %d below", i, above, below)
		}
	}
}

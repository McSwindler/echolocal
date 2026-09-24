package drawer

import (
	"testing"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// The picture as this device shows it.
const (
	screenW = 1920
	screenH = 1200
)

// called is a name that does not change, which is what a test wants.
func called(s string) func() string { return func() string { return s } }

func three() []Entry {
	mark := func(i ui.Icon) func() ui.Icon { return func() ui.Icon { return i } }
	return []Entry{
		{Name: called("Volume"), Icon: mark(icons.AVVolumeUp)},
		{Name: called("Display"), Icon: mark(icons.DeviceBrightnessMedium)},
		{Name: called("Settings"), Icon: mark(icons.ActionSettings)},
	}
}

// none is what the rail passes when nobody is touching it.
func none(int) bool { return false }

func TestStripHangsOffItsEdge(t *testing.T) {
	tests := []struct {
		name string
		edge config.Edge
		near func(ui.Rect) bool
	}{
		{"left", config.EdgeLeft, func(r ui.Rect) bool { return r.X < screenW/4 }},
		{"right", config.EdgeRight, func(r ui.Rect) bool { return r.X+r.W > screenW*3/4 }},
		{"top", config.EdgeTop, func(r ui.Rect) bool { return r.Y < screenH/4 }},
		{"bottom", config.EdgeBottom, func(r ui.Rect) bool { return r.Y+r.H > screenH*3/4 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strip := Strip(tt.edge, screenW, screenH)
			if !tt.near(strip) {
				t.Errorf("a %s rail is at %+v, which is not against that edge", tt.name, strip)
			}
			if strip.W <= 0 || strip.H <= 0 {
				t.Errorf("the rail has no area: %+v", strip)
			}
		})
	}
}

// The clock has to stay readable beside it, which is the whole reason the rail is a rail.
func TestTheRailLeavesMostOfTheScreen(t *testing.T) {
	for _, edge := range []config.Edge{config.EdgeLeft, config.EdgeRight, config.EdgeTop, config.EdgeBottom} {
		strip := Strip(edge, screenW, screenH)

		taken := strip.W * strip.H
		if whole := screenW * screenH; taken > whole/4 {
			t.Errorf("a %v rail takes %d%% of the screen", edge, taken*100/whole)
		}
	}
}

func TestStripInPortrait(t *testing.T) {
	strip := Strip(config.EdgeRight, screenH, screenW)

	if strip.X+strip.W > screenH {
		t.Errorf("the rail runs off the side: %+v", strip)
	}
	if strip.Y+strip.H > screenW {
		t.Errorf("the rail runs off the bottom: %+v", strip)
	}
}

func TestCellsStackWithoutOverlapping(t *testing.T) {
	strip := Strip(config.EdgeRight, screenW, screenH)
	cells := Cells(strip, 3)

	if len(cells) != 3 {
		t.Fatalf("got %d cells, want 3", len(cells))
	}
	for i, at := range cells {
		if at.X < strip.X || at.X+at.W > strip.X+strip.W {
			t.Errorf("cell %d is outside the rail: %+v", i, at)
		}
		if i > 0 && at.Y < cells[i-1].Y+cells[i-1].H {
			t.Errorf("cell %d overlaps the one before it", i)
		}
	}
}

// Three icons on a full-height rail should sit together in the middle rather than be spread down
// it, which would read as a rail with gaps in.
func TestCellsAreCentredAsABlock(t *testing.T) {
	strip := Strip(config.EdgeRight, screenW, screenH)
	cells := Cells(strip, 3)

	above := cells[0].Y - strip.Y
	below := strip.Y + strip.H - (cells[2].Y + cells[2].H)

	if d := above - below; d > 2 || d < -2 {
		t.Errorf("%d above the icons and %d below: the block is not centered", above, below)
	}
}

func TestCellsOfNothing(t *testing.T) {
	if got := Cells(Strip(config.EdgeRight, screenW, screenH), 0); got != nil {
		t.Errorf("Cells(0) = %v, want nothing", got)
	}
}

func TestEveryIconCanBeHit(t *testing.T) {
	strip := Strip(config.EdgeRight, screenW, screenH)

	for i, at := range Cells(strip, 3) {
		x, y := at.Center()
		if !strip.Contains(x, y) {
			t.Errorf("the middle of icon %d is not on the rail", i)
		}
	}
}

func TestDrawPaintsTheRailAndLeavesTheRest(t *testing.T) {
	palette := theme.Default()
	img := ui.NewImage(screenW, screenH, palette.Background)

	strip := Strip(config.EdgeRight, screenW, screenH)
	Draw(img, strip, three(), palette, none)

	x, y := strip.Center()
	if got := img.At(x, y); got == palette.Background {
		t.Error("the rail is empty")
	}

	// The clock's side of the screen is untouched.
	if got := img.At(40, screenH/2); got != palette.Background {
		t.Errorf("the far side is %v, want the screen behind it", got)
	}
}

func TestDrawInEveryThemeAndEdge(t *testing.T) {
	for _, palette := range theme.All {
		for _, edge := range []config.Edge{config.EdgeLeft, config.EdgeRight, config.EdgeTop, config.EdgeBottom} {
			img := ui.NewImage(screenW, screenH, palette.Background)
			Draw(img, Strip(edge, screenW, screenH), three(), palette, none)
		}
	}
}

// The name goes under the mark, or the rail is a column of glyphs and two of them are speakers
// leading to different places.
func TestTheNameIsDrawnUnderTheMark(t *testing.T) {
	palette := theme.Default()
	strip := Strip(config.EdgeRight, screenW, screenH)

	ink := func(entries []Entry) int {
		img := ui.NewImage(screenW, screenH, palette.Background)
		Draw(img, strip, entries, palette, none)

		var lit int
		for y := strip.Y; y < strip.Y+strip.H; y++ {
			for x := strip.X; x < strip.X+strip.W; x++ {
				if img.At(x, y) == palette.Muted {
					lit++
				}
			}
		}
		return lit
	}

	bare := make([]Entry, len(three()))
	for i, e := range three() {
		bare[i] = Entry{Icon: e.Icon}
	}

	if named, plain := ink(three()), ink(bare); named <= plain {
		t.Errorf("named entries drew %d label pixels and unnamed ones %d", named, plain)
	}
}

// A finger on a cell has to show, which the rail never did while every other screen does.
func TestAPressedCellIsMarked(t *testing.T) {
	palette := theme.Default()
	strip := Strip(config.EdgeRight, screenW, screenH)
	at := Cells(strip, 3)[1]

	same := func(pressed func(int) bool) *ui.Image {
		img := ui.NewImage(screenW, screenH, palette.Background)
		Draw(img, strip, three(), palette, pressed)
		return img
	}

	quiet := same(none)
	held := same(func(i int) bool { return i == 1 })

	var moved int
	for y := at.Y; y < at.Y+at.H; y++ {
		for x := at.X; x < at.X+at.W; x++ {
			if quiet.At(x, y) != held.At(x, y) {
				moved++
			}
		}
	}
	if moved == 0 {
		t.Error("a pressed cell looks the same as an untouched one")
	}

	// And only that cell: a press must not light the whole rail.
	other := Cells(strip, 3)[0]
	for y := other.Y; y < other.Y+other.H; y++ {
		for x := other.X; x < other.X+other.W; x++ {
			if quiet.At(x, y) != held.At(x, y) {
				t.Fatalf("pressing one cell changed another at %d,%d", x, y)
			}
		}
	}
}

// Every cell has to hold a mark and its name without running into the one below.
func TestTheCellsDoNotOverlap(t *testing.T) {
	for _, edge := range []config.Edge{config.EdgeLeft, config.EdgeRight, config.EdgeTop, config.EdgeBottom} {
		strip := Strip(edge, screenW, screenH)

		cells := Cells(strip, 4)
		for i, a := range cells {
			for j, b := range cells {
				if i != j && a.Overlaps(b) {
					t.Errorf("%s: cells %d and %d overlap", edge, i, j)
				}
			}
			if !strip.Contains(a.X, a.Y) {
				t.Errorf("%s: cell %d starts outside the rail", edge, i)
			}
		}
	}
}

// An entry with no mark is skipped rather than drawn as a hole.
func TestAnEntryWithNoIconIsSkipped(t *testing.T) {
	palette := theme.Default()
	img := ui.NewImage(screenW, screenH, palette.Background)

	Draw(img, Strip(config.EdgeRight, screenW, screenH), []Entry{{}}, palette, none)
}

func TestEdgesRoundTripToTheTouchscreen(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range []config.Edge{config.EdgeLeft, config.EdgeRight, config.EdgeTop, config.EdgeBottom} {
		got := asTouch(e).String()
		if seen[got] {
			t.Errorf("two edges both map to %q", got)
		}
		seen[got] = true
	}
}

// Added out of order, drawn in order. The rail is learned by position, so where each thing sits
// cannot depend on which feature happened to be built first.
func TestTheOrderIsDeclaredNotTheOrderAdded(t *testing.T) {
	r := &Rail{}

	for _, e := range []Entry{
		{Name: called("Settings"), Order: OrderSettings},
		{Name: called("Player"), Order: OrderPlayer},
		{Name: called("Display"), Order: OrderDisplay},
		{Name: called("Volume"), Order: OrderVolume},
	} {
		r.Add(e)
	}

	want := []string{"Volume", "Display", "Settings", "Player"}
	for i, e := range r.Entries() {
		if e.Label() != want[i] {
			t.Errorf("position %d is %s, want %s", i, e.Label(), want[i])
		}
	}
}

// Two features picking the same number still land the same way every boot, rather than however
// the import graph came out that build.
func TestEqualOrdersFallBackToTheName(t *testing.T) {
	first, second := &Rail{}, &Rail{}

	first.Add(Entry{Name: called("Zebra"), Order: 50})
	first.Add(Entry{Name: called("Aardvark"), Order: 50})

	second.Add(Entry{Name: called("Aardvark"), Order: 50})
	second.Add(Entry{Name: called("Zebra"), Order: 50})

	for i, e := range first.Entries() {
		if got := second.Entries()[i].Label(); got != e.Label() {
			t.Errorf("position %d is %s one way round and %s the other", i, e.Label(), got)
		}
	}
	if first.Entries()[0].Label() != "Aardvark" {
		t.Errorf("the tiebreak is not the name: %s came first", first.Entries()[0].Label())
	}
}

// The places on the rail are distinct, or two of them would sort by name and read as arbitrary.
func TestTheDeclaredPlacesAreDistinct(t *testing.T) {
	seen := map[int]bool{}

	for _, at := range []int{OrderVolume, OrderDisplay, OrderSettings, OrderPlayer} {
		if seen[at] {
			t.Errorf("two things are declared at %d", at)
		}
		seen[at] = true
	}
}

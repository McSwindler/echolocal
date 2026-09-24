package shell

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

// reading is a page of things to look at and nothing to press, which is what the network and about
// screens are: rows, and no actions at all.
func reading(rows ...string) *Page {
	p := &Page{
		Title: "Reading",
		Build: func() ([]widget.Row, []func(int)) {
			out := make([]widget.Row, 0, len(rows))
			for _, label := range rows {
				out = append(out, widget.Row{Label: label, Kind: widget.Plain, Value: "yes"})
			}
			return out, nil
		},
	}

	p.Draw(ui.NewImage(1200, 1920, theme.Default().Background), theme.Default())
	return p
}

// Tapping a row on a page that has no actions used to index past the end of an empty list and take
// the touch service down with it. Every row, because the first one is the one a hand-written test
// would try and the fourth is the one somebody actually hit.
func TestTappingARowThatDoesNothing(t *testing.T) {
	p := reading("Network", "Address", "Hardware address", "Certificates")

	if len(p.places) != 4 {
		t.Fatalf("the page drew %d rows, want 4", len(p.places))
	}

	for i, at := range p.places {
		x, y := at.Center()

		if !p.Tap(x, y) {
			t.Errorf("row %d did not take the touch", i)
		}
	}
}

// The same page must not think a finger has taken hold of anything either, or a drag across it
// reaches for a row that has no slider.
func TestGrabbingARowThatDoesNothing(t *testing.T) {
	p := reading("Network", "Address")

	for i, at := range p.places {
		x, y := at.Center()

		if p.Grab(x, y) {
			t.Errorf("row %d reported something to drag", i)
		}
	}
}

// A page whose actions are shorter than its rows is the same bug with one row covered: the rows
// that have an action still run it.
func TestAPageWithFewerActionsThanRows(t *testing.T) {
	var ran int

	p := &Page{
		Title: "Short",
		Build: func() ([]widget.Row, []func(int)) {
			return []widget.Row{
					{Label: "Does something", Kind: widget.Chevron},
					{Label: "Does nothing", Kind: widget.Plain},
					{Label: "Also nothing", Kind: widget.Plain},
				}, []func(int){
					func(int) { ran++ },
				}
		},
	}
	p.Draw(ui.NewImage(1200, 1920, theme.Default().Background), theme.Default())

	for _, at := range p.places {
		x, y := at.Center()
		p.Tap(x, y)
	}

	if ran != 1 {
		t.Errorf("the one action ran %d times, want once", ran)
	}
}

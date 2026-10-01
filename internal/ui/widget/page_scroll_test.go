package widget

import (
	"fmt"
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

func long(n, scroll int) Page {
	rows := make([]Row, n)
	for i := range rows {
		rows[i] = Row{Label: fmt.Sprintf("Row %d", i), Kind: Chevron, Value: "value"}
	}
	return Page{Title: "Long", Rows: rows, Scroll: scroll}
}

func TestAShortPageHasNothingToScroll(t *testing.T) {
	m := New(1200, 1920)
	d := m.DrawScrolled(ui.NewImage(1200, 1920, theme.Color{}), long(3, 500), theme.Default())
	if d.Overflow != 0 || d.Scroll != 0 {
		t.Errorf("overflow %d scroll %d", d.Overflow, d.Scroll)
	}
	if m.ScrollBar(d) != (ui.Rect{}) {
		t.Error("a scroll bar on a page that fits")
	}
}

func TestALongPageScrollsAsFarAsItsLastRow(t *testing.T) {
	m := New(1200, 1920)
	img := ui.NewImage(1200, 1920, theme.Color{})
	d := m.DrawScrolled(img, long(30, 1<<20), theme.Default())
	if d.Overflow <= 0 {
		t.Fatal("thirty rows fit on the page")
	}
	if d.Scroll != d.Overflow {
		t.Errorf("scrolled %d, want the overflow %d", d.Scroll, d.Overflow)
	}
	last := d.Places[len(d.Places)-1]
	if last.Y+last.H != d.Body.Y+d.Body.H {
		t.Errorf("last row ends at %d, body at %d", last.Y+last.H, d.Body.Y+d.Body.H)
	}
	if d.Places[0].Y >= d.Body.Y {
		t.Error("the first row is still in view at the bottom of the scroll")
	}

	if d := m.DrawScrolled(img, long(30, -40), theme.Default()); d.Scroll != 0 {
		t.Errorf("a negative scroll gave %d", d.Scroll)
	}
}

func TestScrolledRowsStayOutOfTheHeader(t *testing.T) {
	m := New(1200, 1920)
	palette := theme.Default()
	top := ui.NewImage(1200, 1920, theme.Color{})
	down := ui.NewImage(1200, 1920, theme.Color{})
	d := m.DrawScrolled(top, long(30, 0), palette)
	m.DrawScrolled(down, long(30, d.Overflow/2), palette)

	for y := range d.Body.Y {
		for x := range 1200 {
			if top.At(x, y) != down.At(x, y) {
				t.Fatalf("scrolling changed the header at %d,%d", x, y)
			}
		}
	}
}

func TestTheScrollBarFollowsTheScroll(t *testing.T) {
	m := New(1200, 1920)
	img := ui.NewImage(1200, 1920, theme.Color{})
	top := m.DrawScrolled(img, long(30, 0), theme.Default())
	end := m.DrawScrolled(img, long(30, 1<<20), theme.Default())

	a, b := m.ScrollBar(top), m.ScrollBar(end)
	if a.Y != top.Body.Y {
		t.Errorf("bar starts at %d, body at %d", a.Y, top.Body.Y)
	}
	if b.Y+b.H != end.Body.Y+end.Body.H {
		t.Errorf("bar ends at %d, body at %d", b.Y+b.H, end.Body.Y+end.Body.H)
	}
	if a.X < top.Body.X+top.Body.W {
		t.Error("the bar sits over the rows")
	}
}

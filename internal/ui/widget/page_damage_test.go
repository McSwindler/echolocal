package widget

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

func settings() Page {
	return Page{
		Title: "Screen",
		Rows: []Row{
			{Label: "Theme", Value: "Ember", Kind: Chevron},
			{Label: "Brightness", Kind: Slider, Level: 70},
			{Label: "Privacy marks", Kind: Toggle},
			{Label: "Media", Kind: Slider, Level: 40},
			{Label: "Sleep", Hint: "after a minute", Kind: Chevron},
		},
	}
}

// One row changing changes only that row's rectangle.
//
// This is what shell.Page.Damaged promises and what the shell acts on: while a slider is dragged it
// repaints the one rectangle DrawPage reported for that row. If a row's change reached outside it,
// the rest would be left stale until something repainted the whole page.
func TestChangingOneRowChangesOnlyItsOwnRectangle(t *testing.T) {
	palette := theme.Default()
	w, h := 1200, 1920
	m := New(w, h)

	for _, tc := range []struct {
		name string
		row  int
		edit func(*Row)
	}{
		{"slider dragged", 1, func(r *Row) { r.Level = 15 }},
		{"slider to nothing", 3, func(r *Row) { r.Level = 0 }},
		{"slider to full", 3, func(r *Row) { r.Level = 100 }},
		{"toggle flipped", 2, func(r *Row) { r.On = true }},
		{"value changed", 0, func(r *Row) { r.Value = "Slate" }},
	} {
		before := ui.NewImage(w, h, palette.Background)
		places := m.DrawPage(before, settings(), palette)

		after := ui.NewImage(w, h, palette.Background)
		edited := settings()
		tc.edit(&edited.Rows[tc.row])
		m.DrawPage(after, edited, palette)

		moved := ui.Changed(before, after)
		if moved.W == 0 || moved.H == 0 {
			t.Errorf("%s: nothing changed on the page", tc.name)
			continue
		}
		if !places[tc.row].Holds(moved) {
			t.Errorf("%s: row %d changed %+v, outside its own rectangle %+v",
				tc.name, tc.row, moved, places[tc.row])
		}
	}
}

// A finger landing on a row marks that row, and the mark is the same promise: the shell repaints
// one rectangle for it.
func TestPressingOneRowMarksOnlyItsOwnRectangle(t *testing.T) {
	palette := theme.Default()
	w, h := 1200, 1920
	m := New(w, h)

	for row := range len(settings().Rows) {
		before := ui.NewImage(w, h, palette.Background)
		places := m.DrawPage(before, settings(), palette)

		after := ui.NewImage(w, h, palette.Background)
		pressed := settings()
		pressed.Pressed = func(i int) bool { return i == row }
		m.DrawPage(after, pressed, palette)

		moved := ui.Changed(before, after)
		if moved.W == 0 || moved.H == 0 {
			t.Errorf("row %d: pressing it changed nothing", row)
			continue
		}
		if !places[row].Holds(moved) {
			t.Errorf("row %d: pressing it changed %+v, outside %+v", row, moved, places[row])
		}
	}
}

// The same promise for a grid, which is the other half of a page.
func TestChangingOneCellChangesOnlyItsOwnTile(t *testing.T) {
	palette := theme.Default()
	w, h := 1200, 1920
	m := New(w, h)

	page := func(chosen int) Page {
		p := Page{Title: "Theme", Across: 2}
		for i, name := range []string{"Ember", "Slate", "Bone", "Ink", "Moss", "Rust"} {
			p.Cells = append(p.Cells, Cell{Label: name, Chosen: i == chosen})
		}
		return p
	}

	before := ui.NewImage(w, h, palette.Background)
	places := m.DrawPage(before, page(0), palette)

	after := ui.NewImage(w, h, palette.Background)
	m.DrawPage(after, page(0), palette)

	// Nothing edited yet: two draws of the same page must agree, or nothing below means anything.
	if moved := ui.Changed(before, after); moved.W != 0 || moved.H != 0 {
		t.Fatalf("drawing the same page twice differed over %+v", moved)
	}

	// Moving the mark touches two tiles, so each has to stay inside its own.
	third := ui.NewImage(w, h, palette.Background)
	m.DrawPage(third, page(3), palette)

	moved := ui.Changed(before, third)
	if moved.W == 0 || moved.H == 0 {
		t.Fatal("moving the chosen mark changed nothing")
	}

	both := places[0].Union(places[3])
	if !both.Holds(moved) {
		t.Errorf("moving the mark changed %+v, outside the two tiles %+v", moved, both)
	}
}

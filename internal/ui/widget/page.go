package widget

import (
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// The page, as fractions of the shorter side of the picture.
const (
	titleShare  = 0.052
	headerShare = 0.16
	edgeShare   = 0.05

	// pressedShare is how far a row shifts towards the text color under a finger.
	pressedShare = 0.12
)

// Page is a titled list filling the picture.
type Page struct {
	Title string

	Rows []Row

	// Cells and Across are a grid instead of a list, for a choice with more options than a screen
	// of rows holds. A page is one or the other: cells win where both are set. Across is how many
	// tiles to a line, and zero is as many as it takes to fit them all.
	Cells  []Cell
	Across int

	// Pressed reports whether a finger is on a row, for the mark that says a touch landed. Nil is
	// a page nobody is touching.
	Pressed func(row int) bool

	// Aside draws beside the rows: across the bottom when the screen is taller than wide, down
	// the left when it is wider.
	Aside func(s ui.Surface, at ui.Rect, palette theme.Theme, m Metrics)
}

// asideShare is how much of the body the aside takes in portrait.
const asideShare = 0.45

// aside splits the body between the picture and the rows.
func aside(body ui.Rect) (box, rest ui.Rect) {
	if body.W > body.H {
		w := body.W / 2
		return ui.Rect{X: body.X, Y: body.Y, W: w, H: body.H},
			ui.Rect{X: body.X + w, Y: body.Y, W: body.W - w, H: body.H}
	}

	h := int(float64(body.H) * asideShare)
	return ui.Rect{X: body.X, Y: body.Y + body.H - h, W: body.W, H: h},
		ui.Rect{X: body.X, Y: body.Y, W: body.W, H: body.H - h}
}

// Header is the band across the top holding the back arrow and the title.
func (m Metrics) Header(in ui.Rect) ui.Rect {
	return ui.Rect{X: in.X, Y: in.Y, W: in.W, H: int(float64(m.Unit) * headerShare)}
}

// Arrow is where the back arrow is drawn. The whole header goes back, not just this.
func (m Metrics) Arrow(in ui.Rect) ui.Rect {
	h := m.Header(in)
	edge := int(float64(m.Unit) * edgeShare)

	// Square, and taller than the mark in it: a finger aimed at the corner of the screen should
	// not have to be accurate.
	side := h.H
	return ui.Rect{X: h.X + edge - side/4, Y: h.Y, W: side, H: side}
}

// Body is where the rows go.
func (m Metrics) Body(in ui.Rect) ui.Rect {
	h := m.Header(in)
	edge := int(float64(m.Unit) * edgeShare)
	return ui.Rect{X: in.X + edge, Y: in.Y + h.H, W: in.W - edge*2, H: in.H - h.H - edge}
}

// DrawPage paints the whole picture and answers where each row landed, so the caller can work out
// what a finger hit without measuring the same thing twice.
func (m Metrics) DrawPage(s ui.Surface, p Page, palette theme.Theme) []ui.Rect {
	w, h := s.Size()
	in := ui.Rect{W: w, H: h}

	ui.Fill(s, palette.Background)

	title := m.Title
	head := m.Header(in)

	arrow := m.Arrow(in)
	m.arrow(s, arrow, palette.Text)
	x := arrow.X + arrow.W

	_, th := title.Measure(p.Title)
	ui.DrawText(s, title, x, head.Y+(head.H-th)/2+head.H/8, palette.Text, palette.Background, p.Title)

	// Only what the surface is accepting. A row outside the clip would be rasterized and then
	// thrown away a pixel at a time, which is the whole cost of drawing it.
	clip := ui.ClipOf(s)

	body := m.Body(in)
	if p.Aside != nil {
		var box ui.Rect
		box, body = aside(body)
		p.Aside(s, box, palette, m)
	}

	if len(p.Cells) > 0 {
		return m.cells(s, body, p, palette, clip)
	}

	places := m.Rows(body, len(p.Rows))
	for i, r := range p.Rows {
		if !places[i].Overlaps(clip) {
			continue
		}

		on := palette.Background
		if p.Pressed != nil && p.Pressed(i) {
			on = palette.Background.Blend(palette.Text, pressedShare)
			ui.FillRounded(s, places[i], m.Pad, on)
		}
		m.Draw(s, places[i], r, palette, on)
	}
	return places
}

// cells draws the grid half of a page and answers where each tile landed.
func (m Metrics) cells(s ui.Surface, body ui.Rect, p Page, palette theme.Theme, clip ui.Rect) []ui.Rect {
	across := p.Across
	if across <= 0 {
		across = m.Across(body, len(p.Cells))
	}

	places := m.Cells(body, len(p.Cells), across)
	for i, c := range p.Cells {
		if !places[i].Overlaps(clip) {
			continue
		}

		at := places[i]
		if p.Pressed != nil && p.Pressed(i) {
			// Inset rather than tinted: a tile is mostly somebody else's colors, and a wash over
			// the top of a theme swatch would be a lie about what the theme looks like.
			at = at.Inset(m.Pad / 2)
		}
		m.DrawCell(s, at, c, palette)
	}
	return places
}

// arrow is the mark that says this page came from somewhere.
func (m Metrics) arrow(s ui.Surface, in ui.Rect, c theme.Color) {
	cx, cy := in.Center()
	w := m.Icon / 3
	t := max(m.Icon/10, 2)

	ui.FillPolygon(s, []ui.Point{
		{X: cx + w/2, Y: cy - w},
		{X: cx + w/2, Y: cy - w - t},
		{X: cx - w/2 - t, Y: cy},
		{X: cx + w/2, Y: cy + w + t},
		{X: cx + w/2, Y: cy + w},
		{X: cx - w/2 + t, Y: cy},
	}, c)
}

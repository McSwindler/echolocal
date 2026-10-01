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
	// of rows holds. A page is one or the other: cells win where both are set.
	Cells  []Cell
	Across int
	Tall   float64

	// Pressed reports whether a finger is on a row, for the mark that says a touch landed. Nil is
	// a page nobody is touching.
	Pressed func(row int) bool

	// Aside draws beside the rows: across the bottom when the screen is taller than wide, down
	// the left when it is wider.
	Aside func(s ui.Surface, at ui.Rect, palette theme.Theme, m Metrics)

	Action ui.Icon

	Scroll int
}

type Drawn struct {
	Places   []ui.Rect
	Body     ui.Rect
	Scroll   int
	Overflow int
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

func (m Metrics) Action(in ui.Rect) ui.Rect {
	h := m.Header(in)
	edge := int(float64(m.Unit) * edgeShare)
	side := h.H * 3 / 5
	return ui.Rect{X: h.X + h.W - edge - side, Y: h.Y + (h.H-side)/2 + h.H/16, W: side, H: side}
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
	return m.DrawScrolled(s, p, palette).Places
}

func (m Metrics) DrawScrolled(s ui.Surface, p Page, palette theme.Theme) Drawn {
	w, h := s.Size()
	in := ui.Rect{W: w, H: h}

	ui.Fill(s, palette.Background)

	title := m.Title
	head := m.Header(in)

	arrow := m.Arrow(in)
	m.Back(s, arrow, palette.Text)
	x := arrow.X + arrow.W

	_, th := title.Measure(p.Title)
	ui.DrawText(s, title, x, head.Y+(head.H-th)/2+head.H/8, palette.Text, palette.Background, p.Title)

	if p.Action != nil {
		box := m.Action(in)
		ui.FillRounded(s, box, box.W/2, palette.Accent)
		ui.DrawIcon(s, p.Action, box.Inset(box.W/4), palette.Background, palette.Accent)
	}

	// Only what the surface is accepting. A row outside the clip would be rasterized and then
	// thrown away a pixel at a time, which is the whole cost of drawing it.
	clip := ui.ClipOf(s)

	body := m.Body(in)
	if p.Aside != nil {
		var box ui.Rect
		box, body = aside(body)
		p.Aside(s, box, palette, m)
	}

	var places []ui.Rect
	if len(p.Cells) > 0 {
		across := p.Across
		if across <= 0 {
			across = m.Across(body, len(p.Cells))
		}
		places = m.CellsShaped(body, len(p.Cells), across, p.Tall)
	} else {
		places = m.Rows(body, len(p.Rows))
	}

	d := Drawn{Places: places, Body: body}
	if n := len(places); n > 0 {
		d.Overflow = max(places[n-1].Y+places[n-1].H-(body.Y+body.H), 0)
	}
	d.Scroll = min(max(p.Scroll, 0), d.Overflow)
	for i := range places {
		places[i].Y -= d.Scroll
	}

	view := body
	if clip.W > 0 && clip.H > 0 {
		view = intersect(view, clip)
	}
	if view.W <= 0 || view.H <= 0 {
		return d
	}
	inside := ui.Within(s, view)

	if len(p.Cells) > 0 {
		m.cells(inside, places, p, palette, view)
	} else {
		for i, r := range p.Rows {
			if !places[i].Overlaps(view) {
				continue
			}

			on := palette.Background
			if p.Pressed != nil && p.Pressed(i) {
				on = palette.Background.Blend(palette.Text, pressedShare)
				ui.FillRounded(inside, places[i], m.Pad, on)
			}
			m.Draw(inside, places[i], r, palette, on)
		}
	}

	if d.Overflow > 0 {
		m.scrollBar(s, d, palette)
	}
	return d
}

func intersect(a, b ui.Rect) ui.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return ui.Rect{}
	}
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func (m Metrics) ScrollBar(d Drawn) ui.Rect {
	if d.Overflow <= 0 {
		return ui.Rect{}
	}
	edge := int(float64(m.Unit) * edgeShare)
	w := max(m.Pad/3, 3)
	body := d.Body
	tall := max(body.H*body.H/(body.H+d.Overflow), m.Row/3)
	return ui.Rect{
		X: body.X + body.W + (edge-w)/2,
		Y: body.Y + (body.H-tall)*d.Scroll/d.Overflow,
		W: w,
		H: tall,
	}
}

func (m Metrics) scrollBar(s ui.Surface, d Drawn, palette theme.Theme) {
	bar := m.ScrollBar(d)
	track := ui.Rect{X: bar.X, Y: d.Body.Y, W: bar.W, H: d.Body.H}
	ui.FillRounded(s, track, bar.W/2, palette.Background.Blend(palette.Text, 0.08))
	ui.FillRounded(s, bar, bar.W/2, palette.Background.Blend(palette.Text, 0.45))
}

// cells draws the grid half of a page.
func (m Metrics) cells(s ui.Surface, places []ui.Rect, p Page, palette theme.Theme, clip ui.Rect) {
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
}

// arrow is the mark that says this page came from somewhere.
func (m Metrics) Back(s ui.Surface, in ui.Rect, c theme.Color) {
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

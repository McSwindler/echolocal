// Package widget is the controls a screen is built from.
//
// A row is described rather than held: the caller says what it wants, Metrics says where it goes,
// and Draw paints it. Nothing here keeps state, so a screen redraws by describing itself again.
package widget

import (
	"fmt"
	"math"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// Kind is what a row lets someone do.
type Kind int

const (
	// Plain states something and takes no touch.
	Plain Kind = iota

	// Chevron opens something else.
	Chevron

	// Toggle is on or off.
	Toggle

	// Slider is a level from 0 to 100, dragged.
	Slider
)

// Row is one line of a screen.
type Row struct {
	Label string

	// Hint is the smaller line under the label, for a setting whose name does not explain it.
	Hint string

	// Snap turns where the finger is into where the value can actually sit, for a slider whose
	// range is coarser than the hundred steps a track has. Nil is a level that means itself.
	Snap func(level int) int

	// Value is what a Plain or Chevron row says on the right.
	Value string

	Kind  Kind
	On    bool
	Level int

	Icon ui.Icon

	// Chosen marks the row as the one in effect, for a list someone is picking from.
	Chosen bool

	// Dim is a row that is showing but has nothing to do, because something else has taken it
	// over. It draws faded and does not answer a touch.
	Dim bool

	// Preview draws the thing the row names, small, inside the box it is given.
	//
	// For a choice that cannot be described in a word: a clock face, a theme, a photograph. A list
	// that says "Cards" is asking somebody to imagine it, and the option can draw itself.
	Preview func(s ui.Surface, at ui.Rect, palette theme.Theme)
}

// The shape of a screen, as fractions of the shorter side of the picture. Off the picture rather
// than off the row: a row is as tall as there is room for, and type that grew with it would be
// unreadable long before the row ran out of room.
const (
	// dimShare is how far a row with nothing to do fades toward the background.
	dimShare = 0.6

	rowShare   = 0.105
	labelShare = 0.030
	hintShare  = 0.021
	valueShare = 0.028
	iconShare  = 0.048
	padShare   = 0.022

	// looseShare is the most a row is stretched to fill a short list.
	looseShare = 1.5
	railShare  = 0.028

	// previewWide is how many row heights wide a preview is. Wider than tall, because what it is
	// previewing is a screen and a square would be a preview of a different shape of panel.
	previewWide = 2
)

// Metrics is everything a screen is measured and drawn with, for one size of picture.
type Metrics struct {
	Unit int
	Row  int
	Pad  int
	Icon int

	// The type scale, largest first: a page's name and the one thing a screen is about, the name of
	// a row, what a row says on the right, and the line under it.
	Title *ui.Font
	Label *ui.Font
	Value *ui.Font
	Hint  *ui.Font
}

// New measures for a picture.
func New(w, h int) Metrics {
	unit := min(w, h)
	return Metrics{
		Unit:  unit,
		Row:   int(float64(unit) * rowShare),
		Pad:   int(float64(unit) * padShare),
		Icon:  int(float64(unit) * iconShare),
		Title: ui.MustLoad(ui.Bold, int(float64(unit)*titleShare)),
		Label: ui.MustLoad(ui.Medium, int(float64(unit)*labelShare)),
		Hint:  ui.MustLoad(ui.Regular, int(float64(unit)*hintShare)),
		Value: ui.MustLoad(ui.Regular, int(float64(unit)*valueShare)),
	}
}

// Rows is where n rows sit inside an area, stacked from the top.
//
// A few rows are spread out to fill rather than huddled at the top with the rest of the screen
// empty, up to a point: past looseShare a list stops reading as a list. Rows past the bottom are
// still returned, so a caller can tell there is more than fits.
func (m Metrics) Rows(in ui.Rect, n int) []ui.Rect {
	if n <= 0 {
		return nil
	}

	row := min(max(m.Row, in.H/n), int(float64(m.Row)*looseShare))

	out := make([]ui.Rect, 0, n)
	for i := range n {
		out = append(out, ui.Rect{X: in.X, Y: in.Y + i*row, W: in.W, H: row})
	}
	return out
}

// Fit is how many whole rows an area holds.
func (m Metrics) Fit(in ui.Rect) int {
	if m.Row <= 0 {
		return 0
	}
	return in.H / m.Row
}

// Draw paints one row over the color it sits on.
func (m Metrics) Draw(s ui.Surface, at ui.Rect, r Row, palette theme.Theme, on theme.Color) {
	if r.Dim {
		palette.Text = palette.Text.Blend(palette.Background, dimShare)
		palette.Muted = palette.Muted.Blend(palette.Background, dimShare)
		palette.Accent = palette.Accent.Blend(palette.Background, dimShare)
	}
	left, right := at.X+m.Pad, at.X+at.W-m.Pad
	x := left

	if r.Icon != nil {
		// Level with the words. On a slider that is not the middle of the row — the bar is there,
		// and an icon behind it shows only as the corners that stick out past the bar.
		mid := at.Y + at.H/2
		if r.Kind == Slider {
			top, lh := m.wordsTop(at, r)
			mid = top + lh/2
		}

		box := ui.Rect{X: x, Y: mid - m.Icon/2, W: m.Icon, H: m.Icon}
		ui.DrawIcon(s, r.Icon, box, palette.Text, on)
		x += m.Icon + m.Pad
	}

	words := ui.Rect{X: x, Y: at.Y, W: right - x, H: at.H}

	switch r.Kind {
	case Toggle:
		w := m.Icon * 2
		box := ui.Rect{X: right - w, Y: at.Y + (at.H-m.Icon)/2, W: w, H: m.Icon}
		m.toggle(s, box, r.On, palette)
		words.W -= w + m.Pad

	case Chevron:
		m.chevron(s, ui.Rect{X: right - m.Icon, Y: at.Y, W: m.Icon, H: at.H}, palette.Muted)
		words.W -= m.Icon + m.Pad

		if r.Value != "" {
			said := fit(m.Value, r.Value, m.room(words, r))
			vw, _ := m.Value.Measure(said)
			ui.DrawTextIn(s, m.Value,
				ui.Rect{X: right - m.Icon - m.Pad - vw, Y: at.Y, W: vw, H: at.H},
				palette.Muted, on, said)
			words.W -= vw + m.Pad
		}

	case Plain:
		if r.Value != "" {
			said := fit(m.Value, r.Value, m.room(words, r))
			ui.DrawTextRight(s, m.Value, words, palette.Muted, on, said)
			vw, _ := m.Value.Measure(said)
			words.W -= vw + m.Pad
		}
	}

	if r.Chosen {
		tick := ui.Rect{X: right - m.Icon, Y: at.Y + (at.H-m.Icon)/2, W: m.Icon, H: m.Icon}
		m.tick(s, tick, palette.Accent)
		words.W -= m.Icon + m.Pad
	}

	// The preview takes what is left of the row, inset so it does not touch the words.
	//
	// The tick's room is left whether this row has one or not, so the previews line up down the
	// page rather than the chosen one sitting a tick's width apart from the rest.
	if r.Preview != nil {
		side := at.H - m.Pad
		edge := right - m.Icon - m.Pad

		box := ui.Rect{X: edge - side*previewWide, Y: at.Y + m.Pad/2, W: side * previewWide, H: side}

		if box.X > words.X {
			r.Preview(s, box, palette)
			words.W = box.X - m.Pad - words.X
		}
	}

	m.words(s, words, r, palette, on)

	if r.Kind == Slider {
		m.track(s, m.Track(at), r.Level, palette)
	}
}

// wordsTop is where a row's label starts, and how tall it is. Shared so the mark beside it lands
// on the same line rather than being placed against the row and hoping.
func (m Metrics) wordsTop(in ui.Rect, r Row) (y, lh int) {
	_, lh = m.Label.Measure(r.Label)
	_, hh := m.Hint.Measure(r.Hint)

	tall := lh
	if r.Hint != "" {
		tall += hh
	}

	// A slider's words sit just above its track rather than in the middle of the row.
	//
	// Not above the row, though: at the natural row height the arithmetic lands a few pixels over
	// the top edge, and a row that paints outside itself breaks what Page.Damaged promises — that
	// repainting the row is all a drag needs. The level is drawn on this line and changes as the
	// finger moves, so those pixels would be left stale.
	if r.Kind == Slider {
		return max(in.Y, in.Y+(in.H-m.trackHeight())/2-m.Pad/2-tall), lh
	}
	return in.Y + (in.H-tall)/2, lh
}

// ellipsis is what a string that did not fit ends with.
const ellipsis = "…"

// leastValue is the share of a row a value keeps even when the label wants all of it. A value the
// label has squeezed to nothing says less than a label cut short, since the value is the part that
// changes.
const leastValue = 3

// room is how wide a value may be drawn: what the label does not need, and never less than a third
// of the row.
//
// The label is ours and short; the value is usually data — a network name, a time zone, whatever
// the device was called at install. Splitting the row evenly would cut those for the sake of a
// label that had room to spare.
func (m Metrics) room(words ui.Rect, r Row) int {
	lw, _ := m.Label.Measure(r.Label)
	if hw, _ := m.Hint.Measure(r.Hint); hw > lw {
		lw = hw
	}
	return max(words.W/leastValue, words.W-lw-m.Pad)
}

// fit is s, cut short enough to draw in width and ended with an ellipsis when it was cut.
//
// A row that paints outside itself leaves marks nothing repaints over: the shell repaints one row
// while a slider is dragged, so anything past the edge stays until the whole page is drawn again.
//
// Cut by rune. Cutting bytes splits a multi-byte character into something that is not one.
func fit(font *ui.Font, s string, width int) string {
	if s == "" || width <= 0 {
		return ""
	}
	if w, _ := font.Measure(s); w <= width {
		return s
	}

	dots, _ := font.Measure(ellipsis)
	if dots > width {
		return ""
	}

	runes := []rune(s)
	for n := len(runes) - 1; n > 0; n-- {
		if w, _ := font.Measure(string(runes[:n])); w+dots <= width {
			return string(runes[:n]) + ellipsis
		}
	}
	return ellipsis
}

// words is the label, and the hint under it when there is one.
func (m Metrics) words(s ui.Surface, in ui.Rect, r Row, palette theme.Theme, on theme.Color) {
	if in.W <= 0 {
		return
	}

	y, lh := m.wordsTop(in, r)

	ui.DrawText(s, m.Label, in.X, y, palette.Text, on, fit(m.Label, r.Label, in.W))
	if r.Hint != "" {
		ui.DrawText(s, m.Hint, in.X, y+lh, palette.Muted, on, fit(m.Hint, r.Hint, in.W))
	}

	// A slider says its own level, level with the label, so the number cannot drift from the bar.
	// Its Value instead when it has one: a rate reads as 10 fps, not as a third of the way along.
	if r.Kind == Slider {
		said := r.Value
		if said == "" {
			said = fmt.Sprintf("%d%%", clamp(r.Level))
		}
		vw, _ := m.Value.Measure(said)
		ui.DrawText(s, m.Value, in.X+in.W-vw, y, palette.Muted, on, said)
	}
}

// Track is where a slider row's bar sits, which is also what a drag is measured against.
//
// Centered in the row, because the row is what a touch is resolved against and the bar is what a
// finger aims at. With the bar at the bottom of its row there was half a row of live area above it
// and almost none below, so aiming just under the bar moved the slider beneath.
func (m Metrics) Track(at ui.Rect) ui.Rect {
	h := m.trackHeight()
	return ui.Rect{
		X: at.X + m.Pad,
		Y: at.Y + (at.H-h)/2,
		W: at.W - m.Pad*2,
		H: h,
	}
}

func (m Metrics) trackHeight() int { return int(float64(m.Unit) * railShare) }

// Level is what a touch at x means on a slider row.
func (m Metrics) Level(at ui.Rect, x int) int {
	t := m.Track(at)
	if t.W <= 0 {
		return 0
	}
	return clamp((x - t.X) * 100 / t.W)
}

func (m Metrics) track(s ui.Surface, t ui.Rect, level int, palette theme.Theme) {
	r := t.H / 2
	ui.FillRounded(s, t, r, palette.Surface.Blend(palette.Text, 0.18))

	w := t.W * clamp(level) / 100
	w = max(w, t.H)
	ui.FillRounded(s, ui.Rect{X: t.X, Y: t.Y, W: w, H: t.H}, r, palette.Accent)
}

func (m Metrics) toggle(s ui.Surface, box ui.Rect, on bool, palette theme.Theme) {
	DrawToggle(s, box, on, palette)
}

// DrawToggle draws a switch in the box it is given.
func DrawToggle(s ui.Surface, box ui.Rect, on bool, palette theme.Theme) {
	r := box.H / 2

	fill := palette.Surface.Blend(palette.Text, 0.22)
	if on {
		fill = palette.Accent
	}
	ui.FillRounded(s, box, r, fill)

	// The knob is the theme's own ink rather than whatever contrasts with the track, so it stays
	// the same object as the track changes color under it.
	gap := max(box.H/10, 2)
	knob := box.H - gap*2

	x := box.X + gap
	if on {
		x = box.X + box.W - knob - gap
	}
	ui.FillRounded(s, ui.Rect{X: x, Y: box.Y + gap, W: knob, H: knob}, knob/2, palette.Text)
}

// chevron is two strokes meeting at a point, rather than a filled arrowhead: at this size a solid
// triangle reads as a button.
func (m Metrics) chevron(s ui.Surface, in ui.Rect, c theme.Color) {
	cx, cy := in.Center()
	reach := in.W / 4
	t := max(in.W/16, 2)

	ui.FillPolygon(s, []ui.Point{
		{X: cx - reach/2, Y: cy - reach},
		{X: cx - reach/2 + t, Y: cy - reach - t},
		{X: cx + reach/2 + t, Y: cy},
		{X: cx - reach/2 + t, Y: cy + reach + t},
		{X: cx - reach/2, Y: cy + reach},
		{X: cx + reach/2 - t, Y: cy},
	}, c)
}

func (m Metrics) tick(s ui.Surface, in ui.Rect, c theme.Color) {
	cx, cy := in.Center()
	w := in.W / 3
	t := max(in.W/9, 2)

	ui.FillPolygon(s, []ui.Point{
		{X: cx - w, Y: cy},
		{X: cx - w + t, Y: cy - t},
		{X: cx - w/3, Y: cy + w - t*2},
		{X: cx + w, Y: cy - w},
		{X: cx + w, Y: cy - w + t},
		{X: cx - w/3, Y: cy + w},
	}, c)
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// Cell is one tile in a grid: a thing that draws itself, with its name on it.
//
// For a list long enough that a row each would scroll. A row spends its width on a label and a
// small preview; a tile spends it on the preview and puts the label inside, which is what lets a
// dozen of them share a screen.
type Cell struct {
	Label string

	// Chosen marks the tile as the one in effect. It is drawn as a ring around the tile rather
	// than a tick beside it: in a grid there is no beside.
	Chosen bool

	// Paint draws the thing the tile names, filling it.
	Paint func(s ui.Surface, at ui.Rect, palette theme.Theme)
}

// How a grid is laid out, as fractions of a cell.
const (
	// cellGap is the space between tiles. Enough to read as separate things, small enough that the
	// tiles themselves keep the width.
	cellGap = 0.06

	// cellTall is a tile's height against its width. Landscape, because what a tile previews is a
	// screen.
	cellTall = 0.72

	cellRound = 0.09
	cellRing  = 0.035

	// cellLabelShare is the label's size against the tile's height.
	cellLabelShare = 0.17
)

// Across is how many tiles to a line for n of them to fit the area whole.
func (m Metrics) Across(in ui.Rect, n int) int {
	for across := 1; across < n; across++ {
		gap := int(float64(in.W) / float64(across) * cellGap)
		wide := (in.W - gap*(across-1)) / across
		rows := (n + across - 1) / across

		if rows*(int(float64(wide)*cellTall)+gap) <= in.H+gap {
			return across
		}
	}
	return max(n, 1)
}

// Cells is where n tiles sit inside an area, in rows of across.
func (m Metrics) Cells(in ui.Rect, n, across int) []ui.Rect {
	return m.CellsShaped(in, n, across, 0)
}

func (m Metrics) CellsShaped(in ui.Rect, n, across int, shape float64) []ui.Rect {
	if n <= 0 || across <= 0 {
		return nil
	}
	if shape <= 0 {
		shape = cellTall
	}

	gap := int(float64(in.W) / float64(across) * cellGap)
	wide := (in.W - gap*(across-1)) / across
	tall := int(float64(wide) * shape)

	out := make([]ui.Rect, 0, n)
	for i := range n {
		out = append(out, ui.Rect{
			X: in.X + (i%across)*(wide+gap),
			Y: in.Y + (i/across)*(tall+gap),
			W: wide,
			H: tall,
		})
	}
	return out
}

// DrawCell paints one tile.
func (m Metrics) DrawCell(s ui.Surface, at ui.Rect, c Cell, palette theme.Theme) {
	base := math.Min(float64(at.H), float64(at.W)*cellTall)
	round := int(base * cellRound)

	if c.Paint != nil {
		c.Paint(s, at, palette)
	} else {
		ui.FillRounded(s, at, round, palette.Surface)
	}

	if c.Label != "" {
		font := ui.MustLoad(ui.Medium, int(base*cellLabelShare))
		m.name(s, at, font, c.Label, palette)
	}

	// The ring is drawn last and outside everything, so a tile whose preview reaches its own edge
	// cannot hide which one is chosen.
	if c.Chosen {
		ui.Border(s, at, max(int(base*cellRing), 2), palette.Accent)
	}
}

// name writes a tile's label across the bottom of it, on a band, because what is behind it is the
// preview and a theme's background is as likely to be white as black.
func (m Metrics) name(s ui.Surface, at ui.Rect, font *ui.Font, label string, palette theme.Theme) {
	_, th := font.Measure(label)
	high := th + m.Pad

	band := ui.Rect{X: at.X, Y: at.Y + at.H - high, W: at.W, H: high}
	ui.FillRect(s, band, palette.Surface)

	ui.DrawTextIn(s, font, band.Inset(m.Pad/3), palette.Text, palette.Surface, label)
}

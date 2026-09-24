package shell

import (
	"time"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

// Page is a titled list of rows, which is what most screens are.
//
// Rows are built each time it is drawn rather than held, because a row says what a setting is at
// and that changes under it: a slider being dragged, a theme taken from Home Assistant.
type Page struct {
	Touches

	Title string

	// Build is the rows as they are now, and what each one does when it is touched. An action is
	// given the level for a slider and ignores it for anything else; nil is a row that does
	// nothing.
	Build func() ([]widget.Row, []func(level int))

	// Tiles is the grid form, for a page with more choices than a screen of rows holds. A page has
	// one or the other, and Across is how many tiles to a line.
	Tiles  func() ([]widget.Cell, []func(int))
	Across int

	// Aside draws beside the rows, along the bottom or down the left as the screen is shaped.
	Aside func(s ui.Surface, at ui.Rect, palette theme.Theme, m widget.Metrics)

	// AsideTap is offered a touch before the rows are, and says whether it took it.
	AsideTap func(x, y int) bool

	places []ui.Rect
	rows   []widget.Row
	acts   []func(int)

	// metrics is what the rows were last measured with, so a touch is resolved against the screen
	// they were drawn on rather than whatever shape the panel is now.
	metrics widget.Metrics

	// held is the row a finger took hold of, and holding whether it has one. A drag stays with
	// the row it started on: a finger pulling a slider wanders up and down, and re-deciding which
	// row it is over on every move hands it whichever one it drifted into.
	held    int
	holding bool
}

func (p *Page) Covers() bool { return true }

// Timeout is the settings one: a page is read and worked through, not glanced at.
func (p *Page) Timeout() time.Duration { return SettingsTimeout }

func (p *Page) Draw(s ui.Surface, palette theme.Theme) {
	w, h := s.Size()
	p.metrics = widget.New(w, h)

	page := widget.Page{Title: p.Title, Pressed: p.Pressed, Across: p.Across, Aside: p.Aside}
	if p.Tiles != nil {
		p.rows, page.Cells, p.acts = nil, nil, nil
		page.Cells, p.acts = p.Tiles()
	} else {
		p.rows, p.acts = p.Build()
		page.Rows = p.rows
	}

	p.places = p.metrics.DrawPage(s, page, palette)

	p.Places(p.places)
}

// Tap runs what the touched row does. A touch in the band above the rows goes back, so the way out
// is the whole header rather than one small mark.
//
// Every page goes back, including the one at the bottom of the stack: back from there is the
// dashboard, which Pop does by closing.
func (p *Page) Tap(x, y int) bool {
	if p.AsideTap != nil && p.AsideTap(x, y) {
		return true
	}
	if at, ok := p.hit(x, y); ok {
		if act := p.act(at); act != nil {
			act(p.level(at))
		}
		return true
	}

	if p.header(y) {
		Get().Pop()
	}
	return true
}

// Grab reports whether a finger landed on a slider, which is what the shell then drags, and
// remembers which one.
func (p *Page) Grab(x, y int) bool {
	at, ok := p.hit(x, y)
	p.holding = ok && at < len(p.rows) && p.rows[at].Kind == widget.Slider
	p.held = at
	return p.holding
}

// Drag sets a slider to wherever the finger is, and says whether that changed anything.
//
// Most moves change nothing: a level has a hundred steps and the track is most of the screen wide,
// so a finger crosses twenty-odd pixels per step. Redrawing each move repaints the whole picture
// for a bar that has not moved.
// Only x is read: the row was settled when the finger went down, and how far up or down it has
// wandered since says nothing about the level.
func (p *Page) Drag(x, _ int) bool {
	at := p.held
	if !p.holding || at >= len(p.rows) || at >= len(p.places) {
		return false
	}

	want := p.metrics.Level(p.places[at], x)
	if want == p.rows[at].Level {
		return false
	}

	// Recorded here rather than left to the redraw that follows: the level the row was last drawn
	// at is what says whether the finger has moved a step, and a gate that only closes once
	// something has been redrawn is a gate that depends on the redraw it is meant to avoid.
	p.rows[at].Level = want

	if act := p.act(at); act != nil {
		act(want)
	}
	return true
}

// act is what the row at this index does, or nil for one that does nothing.
//
// The list of actions is allowed to be shorter than the rows, or missing entirely: a page of things
// to read — the network, the about screen — has no actions at all and says so by passing none.
// Indexing it without asking is what made tapping any row on those pages panic the touch service.
func (p *Page) act(at int) func(int) {
	if at < 0 || at >= len(p.acts) {
		return nil
	}
	return p.acts[at]
}

// level is what a row's action is given. A tile has none: it is the thing it names, not a value.
func (p *Page) level(at int) int {
	if at < 0 || at >= len(p.rows) {
		return 0
	}
	return p.rows[at].Level
}

func (p *Page) hit(x, y int) (int, bool) {
	for i, at := range p.places {
		if at.Contains(x, y) {
			return i, true
		}
	}
	return 0, false
}

func (p *Page) header(y int) bool {
	return len(p.places) > 0 && y < p.places[0].Y
}

// Damaged is the row a drag is moving, which is all a slider step changes.
func (p *Page) Damaged() ui.Rect {
	if !p.holding || p.held >= len(p.places) {
		return ui.Rect{}
	}
	return p.places[p.held]
}

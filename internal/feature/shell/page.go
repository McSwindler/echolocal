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
	Tiles     func() ([]widget.Cell, []func(int))
	Across    int
	AcrossFor func(w, h int) int
	Screened  bool

	Action   ui.Icon
	OnAction func()
	actionAt ui.Rect

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

	scroll, overflow int
	body             ui.Rect
	visible          []ui.Rect
	scrolled         bool
}

func (p *Page) Covers() bool { return true }

// Timeout is the settings one: a page is read and worked through, not glanced at.
func (p *Page) Timeout() time.Duration { return SettingsTimeout }

func (p *Page) Draw(s ui.Surface, palette theme.Theme) {
	w, h := s.Size()
	p.metrics = widget.New(w, h)

	page := widget.Page{Title: p.Title, Pressed: p.Pressed, Across: p.Across, Aside: p.Aside, Action: p.Action, Scroll: p.scroll}
	p.actionAt = ui.Rect{}
	if p.Action != nil {
		p.actionAt = p.metrics.Action(ui.Rect{W: w, H: h})
	}
	if p.AcrossFor != nil {
		page.Across = p.AcrossFor(w, h)
	}
	if p.Screened && w > 0 {
		page.Tall = float64(h) / float64(w)
	}
	if p.Tiles != nil {
		p.rows, page.Cells, p.acts = nil, nil, nil
		page.Cells, p.acts = p.Tiles()
	} else {
		p.rows, p.acts = p.Build()
		page.Rows = p.rows
	}

	d := p.metrics.DrawScrolled(s, page, palette)
	p.places, p.body, p.overflow = d.Places, d.Body, d.Overflow
	p.scroll = min(max(p.scroll, 0), p.overflow)
	p.visible = make([]ui.Rect, len(p.places))
	for i, at := range p.places {
		p.visible[i] = clipTo(at, p.body)
	}

	p.Places(p.visible)
}

func (p *Page) Scroll(by int) bool {
	to := min(max(p.scroll+by, 0), p.overflow)
	if to == p.scroll {
		return false
	}
	p.scroll, p.scrolled = to, true
	return true
}

func clipTo(a, b ui.Rect) ui.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return ui.Rect{}
	}
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
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
	if p.OnAction != nil && p.actionAt.Inset(-p.actionAt.W/4).Contains(x, y) {
		p.OnAction()
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

	// Where the value can sit, not where the finger is. A rate of 1 to 30 has thirty places along
	// a hundred step track, and a bar drawn between two of them is showing a number nothing holds.
	if snap := p.rows[at].Snap; snap != nil {
		want = snap(want)
	}
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
	for i, at := range p.visible {
		if at.W > 0 && at.Contains(x, y) {
			return i, true
		}
	}
	return 0, false
}

func (p *Page) header(y int) bool {
	return len(p.places) > 0 && y < p.body.Y
}

// Damaged is the row a drag is moving, which is all a slider step changes.
func (p *Page) Damaged() ui.Rect {
	if p.scrolled {
		p.scrolled = false
		bar := p.metrics.ScrollBar(widget.Drawn{Body: p.body, Scroll: p.scroll, Overflow: p.overflow})
		return ui.Rect{X: p.body.X, Y: p.body.Y, W: bar.X + bar.W - p.body.X, H: p.body.H}
	}
	if !p.holding || p.held >= len(p.visible) {
		return ui.Rect{}
	}
	return p.visible[p.held]
}

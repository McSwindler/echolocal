// Package shell is what the screen shows while someone is touching it.
//
// It owns one claim and a stack of views. Whatever is on top is drawn and gets the touches; going
// back uncovers what was under it, and the last one going away hands the panel back to the
// dashboard. A view describes itself and is asked where it was touched, so nothing here knows what
// any particular screen is for.
package shell

import (
	"slices"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/component"
	apptheme "github.com/ygelfand/echolocal/internal/feature/theme"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/hardware/touch"
	"github.com/ygelfand/echolocal/internal/lib/hook"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

func init() {
	component.Register(component.Device, Get, component.Order(34))
}

// How long the shell stays up with nobody touching it. A view that wants SettingsTimeout says so
// with Timeout; everything else gets DockTimeout.
const (
	DockTimeout = 15 * time.Second

	// settling is how long after a screen closes its gesture still counts as spent.
	settling        = 400 * time.Millisecond
	SettingsTimeout = 45 * time.Second
)

// Timeouter is a view that wants its own timeout.
type Timeouter interface {
	Timeout() time.Duration
}

// How far a finger travels, and what that means.
//
// Two thresholds rather than one, because "not a tap any more" and "this was a swipe" are different
// distances and treating them as one is what made a slightly imprecise tap close the screen. A
// fingertip on a ten inch panel is eighty pixels across and its center wanders as the pressure
// changes, so a tap has to be allowed to move.
const (
	// still is how far a finger may travel and still count as a tap rather than a drag.
	still = 28

	// sweep is a deliberate swipe: the same tenth of the shorter side the gesture recognizer uses,
	// so what puts a screen away here is what counts as a swipe everywhere else.
	sweep = 120
)

// View is one screen.
type View interface {
	// Draw paints the whole of it. The surface is the whole picture: a view that wants to sit in
	// part of it draws the part it wants and leaves the rest.
	Draw(s ui.Surface, palette theme.Theme)

	// Tap is a finger put down and lifted in one place. Returning false means the touch was not
	// the view's, which closes the shell.
	Tap(x, y int) bool

	// Covers is a view that hides what is under it. A rail hanging off one edge does not, so the
	// dashboard carries on showing beside it.
	Covers() bool
}

// Damager is a view that knows which part of itself a drag changed. The shell passes it on, so a
// slider step repaints a row rather than the screen.
type Damager interface {
	// Damaged is the part that changed, in viewed coordinates. An empty rectangle means all of it.
	Damaged() ui.Rect
}

// Urgent is a view that has to be seen, whatever else is on the screen.
//
// A screen someone opened sits at PriorityUI, under a volume notice, which is right: turning the
// volume while the settings are open should say so. An alarm going off is the other way round, and
// nothing about the stack of views can express that on its own.
type Urgent interface {
	// Urgent reports whether this view outranks everything the device shows by itself. Asked each
	// time the screen is picked, so a view can stop being urgent without being taken away.
	Urgent() bool
}

// urgent reports whether a view is one.
func urgent(v View) bool {
	u, ok := v.(Urgent)
	return ok && u.Urgent()
}

// Presser is a view that shows what a finger is on, so a touch is acknowledged before it has done
// anything.
type Presser interface {
	// Press is where a finger went down, or that it has lifted or slid off. Reporting true means
	// the picture changed and wants painting again.
	Press(x, y int, down bool) bool
}

type Scroller interface {
	Scroll(by int) bool
}

// Dragger is a view with something to pull, such as a slider.
type Dragger interface {
	// Grab takes a finger going down and reports whether it landed on something draggable.
	Grab(x, y int) bool

	// Drag is that finger moving. Returning false means nothing changed, which is most moves: a
	// slider has a hundred steps and a track is a thousand pixels wide.
	Drag(x, y int) bool
}

type Releaser interface {
	Let(x, y int) bool
}

// Shell is the stack.
type Shell struct {
	mu    sync.Mutex
	stack []View
	timer *time.Timer

	// holds are the views something is holding up rather than a person having opened them. Going
	// idle puts away the rest and leaves these.
	holds map[View]bool

	Changed hook.Hook[Change]

	claim *display.Claim

	// covering is what the held claim was taken as, so a view that covers replaces one that does
	// not rather than being drawn into a claim that lets the dashboard through.
	covering bool

	// at is the priority the held claim was taken at, which follows the top view: an urgent one
	// outranks a notice, and an ordinary one does not.
	at display.Priority

	// closed is when the last screen went away, so the gesture that did it is not read twice.
	closed time.Time

	// sliding is the page on its way out, or nil.
	sliding *slide

	// held is what a finger is currently dragging, and where it went down.
	held                  Dragger
	downX, downY          int
	edged                 bool
	moved, swept, dragged bool
	scrolling             bool
	lastY                 int
}

var (
	once   sync.Once
	shared *Shell
)

func Get() *Shell {
	once.Do(func() {
		shared = &Shell{}
		touch.Contacts.Listen(shared.on)
		apptheme.Changed.Listen(func(theme.Theme) { shared.Redraw() })
	})
	return shared
}

func (s *Shell) Name() string { return "shell" }

// Open reports whether anything is showing.
func (s *Shell) Open() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.stack) > 0
}

// Closing says a screen has just gone away, so the gesture that did it is not also an invitation
// to open something else. One contact, one thing.
func (s *Shell) Closing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Since(s.closed) < settling
}

// Top is the view being shown, or nil. For anything that has to know whether the screen someone is
// looking at is its own.
func (s *Shell) Top() View {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.stack) == 0 {
		return nil
	}
	return s.stack[len(s.stack)-1]
}

func (s *Shell) Visible(v View) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Contains(showing(s.stack), v)
}

type Stacked struct {
	View View
	Held bool
}

func (s *Shell) Stack() []Stacked {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Stacked, len(s.stack))
	for i, v := range s.stack {
		out[i] = Stacked{View: v, Held: s.holds[v]}
	}
	return out
}

type Sleeper interface{ Asleep() bool }

type Waker interface{ Wakes() bool }

func (s *Shell) refuses(v View) bool {
	n := len(s.stack)
	if n == 0 || s.stack[n-1] == v {
		return false
	}
	if sl, ok := s.stack[n-1].(Sleeper); !ok || !sl.Asleep() {
		return false
	}
	if !v.Covers() {
		return false
	}
	w, ok := v.(Waker)
	return !ok || !w.Wakes()
}

type Change struct{ From, To View }

func (s *Shell) top() View {
	if n := len(s.stack); n > 0 {
		return s.stack[n-1]
	}
	return nil
}

func (s *Shell) changed(from, to View) {
	if from != to {
		s.Changed.Emit(Change{From: from, To: to})
	}
}

// Push puts a view on top, sliding it in from the side.
func (s *Shell) Push(v View) {
	s.mu.Lock()
	if s.refuses(v) {
		s.mu.Unlock()
		return
	}
	under := s.top()
	s.stack = append(s.stack, v)

	m := begin(under, v, false)
	s.sliding = m
	s.mu.Unlock()
	s.changed(under, v)

	s.wake()
	s.start(m)
}

// Hold puts a view up on something else's behalf and hands back the hold on it. Nothing else has to
// remember whether it was the one that opened a screen: releasing takes away that view and no other,
// wherever it has ended up in the stack.
func (s *Shell) Hold(v View) *Hold {
	s.mu.Lock()
	if s.refuses(v) {
		s.mu.Unlock()
		return nil
	}
	if s.holds == nil {
		s.holds = map[View]bool{}
	}
	s.holds[v] = true

	// A view that is still on screen, no longer held and not yet timed out, is taken back rather
	// than pushed a second time.
	up := slices.Contains(s.stack, v)
	s.mu.Unlock()

	if up {
		s.wake()
	} else {
		s.Push(v)
	}
	return &Hold{shell: s, view: v}
}

// Hold is one thing's claim on the screen. Safe to keep after releasing, and safe when nil.
type Hold struct {
	shell *Shell
	view  View
}

// Keep says whether to go on holding the view up.
//
// Stopping is not the same as releasing: the view stays on the screen and goes with the next idle
// timeout, or when somebody closes it, like anything a person opened.
func (h *Hold) Keep(on bool) {
	if h == nil {
		return
	}

	h.shell.mu.Lock()
	if on {
		if h.shell.holds == nil {
			h.shell.holds = map[View]bool{}
		}
		h.shell.holds[h.view] = true
	} else {
		delete(h.shell.holds, h.view)
	}
	h.shell.mu.Unlock()

	h.shell.wake()
}

// Release takes the held view away. What was under it comes back; a screen opened over it stays.
func (h *Hold) Release() {
	if h == nil {
		return
	}

	h.shell.mu.Lock()
	delete(h.shell.holds, h.view)
	h.shell.mu.Unlock()

	h.shell.remove(h.view)
}

// Held reports whether the view is still up. The shell is asked rather than remembered, so a hold
// that the screen closed out from under does not claim to still have it.
func (h *Hold) Held() bool {
	if h == nil {
		return false
	}

	h.shell.mu.Lock()
	defer h.shell.mu.Unlock()
	return h.shell.holds[h.view]
}

// remove takes one view out of the stack wherever it is.
func (s *Shell) remove(v View) {
	s.mu.Lock()
	from := s.top()
	for i, have := range s.stack {
		if have == v {
			s.stack = append(s.stack[:i], s.stack[i+1:]...)
			break
		}
	}
	empty := len(s.stack) == 0

	// A slide showing the view being taken away would carry on drawing it.
	if s.sliding != nil && s.sliding.from == v {
		s.sliding = nil
	}
	to := s.top()
	s.mu.Unlock()
	s.changed(from, to)

	if empty {
		s.Close()
		return
	}
	s.Redraw()
}

// Pop takes the top view away, sliding it back out the way it came. Popping the last one closes
// the shell, because back from the bottom of the stack is the dashboard.
func (s *Shell) Pop() {
	s.mu.Lock()
	var gone View
	if n := len(s.stack); n > 0 {
		gone = s.stack[n-1]
		s.stack = s.stack[:n-1]
	}
	empty := len(s.stack) == 0

	var m *slide
	if !empty {
		m = begin(gone, s.stack[len(s.stack)-1], true)
		s.sliding = m
	}
	to := s.top()
	s.mu.Unlock()
	s.changed(gone, to)

	if empty {
		s.Close()
		return
	}

	s.wake()
	s.start(m)
}

// Close puts everything away and gives the panel back.
func (s *Shell) Close() {
	s.mu.Lock()
	from := s.top()
	claim, timer := s.claim, s.timer
	s.closed = time.Now()
	s.stack, s.claim, s.timer, s.held = nil, nil, nil, nil
	s.sliding = nil
	clear(s.holds)
	s.mu.Unlock()
	s.changed(from, nil)

	if timer != nil {
		timer.Stop()
	}
	if claim != nil {
		claim.Release()
	}
}

// Redraw paints the top view again, for a view whose contents changed under it.
func (s *Shell) Redraw() { s.redraw(ui.Rect{}) }

// RedrawIn paints again and says which part of the screen changed, for a view that knows.
//
// The rectangle has to hold every pixel that differs, including the ones under an overlay, because
// everything is clipped to it. ui.Changed is how a view proves its own rectangle in a test.
func (s *Shell) RedrawIn(damage ui.Rect) { s.redraw(damage) }

func (s *Shell) redraw(damage ui.Rect) {
	s.mu.Lock()
	if len(s.stack) == 0 {
		s.mu.Unlock()
		return
	}
	top := s.stack[len(s.stack)-1]
	moving := s.sliding

	draw := showing(s.stack)
	covers, at := draw[0].Covers(), rank(draw)

	if s.claim == nil || s.covering != covers || s.at != at {
		if s.claim != nil {
			s.claim.Release()
		}
		if covers {
			s.claim = display.Get().Claim(at)
		} else {
			s.claim = display.Get().Overlay(at)
		}
		s.covering, s.at = covers, at
	}
	claim := s.claim
	s.mu.Unlock()

	palette := apptheme.Get().Current()
	paint := func(p *display.Panel) error {
		// Where the slide has got to is read here rather than when the frame was asked for: the
		// render loop runs behind, and a page placed by an old reading would jump backwards.
		if moving != nil {
			moving.draw(top, ui.Of(p), palette)
			return nil
		}
		for _, v := range draw {
			v.Draw(ui.Of(p), palette)
		}
		return nil
	}

	// Not while a page is sliding: the whole screen is moving, so a rectangle one view worked out
	// for itself describes none of it.
	if damage.W > 0 && damage.H > 0 && moving == nil {
		claim.ShowIn(display.Rect{X: damage.X, Y: damage.Y, W: damage.W, H: damage.H}, paint)
		return
	}
	claim.Show(paint)
}

// showing is the views to draw, lowest first: the highest one that covers the screen, and every
// overlay above it.
//
// The same rule the display driver follows when it picks claims, one level up. Drawing only the top
// view meant an overlay over a screen — the volume card over the settings, or over an alarm going
// off — replaced it rather than sitting on it, and what showed through underneath was the
// dashboard.
func showing(stack []View) []View {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].Covers() {
			return stack[i:]
		}
	}
	return stack
}

// rank is the priority the claim is taken at: anything urgent anywhere in what is being drawn
// lifts the lot.
//
// Anywhere rather than on top, because the shell holds one claim for the whole stack. A volume
// notice over a ringing alarm is still an alarm on the screen, and dropping to PriorityUI for it
// would put both under whatever else is showing.
func rank(draw []View) display.Priority {
	if slices.ContainsFunc(draw, urgent) {
		return display.PriorityAlert
	}
	return display.PriorityUI
}

// idled puts away what a person opened and leaves what something is holding, so a player stays up
// while it plays and the screens over it go.
func (s *Shell) idled() {
	s.mu.Lock()
	from := s.top()
	kept := make([]View, 0, len(s.stack))
	for _, v := range s.stack {
		if s.holds[v] {
			kept = append(kept, v)
		}
	}
	s.stack = kept
	empty := len(s.stack) == 0
	s.sliding = nil
	to := s.top()
	s.mu.Unlock()
	s.changed(from, to)

	if empty {
		s.Close()
		return
	}
	s.Redraw()
}

// wake restarts the idle countdown. Touching the screen and opening or leaving a view, not
// drawing one: a view that repaints itself would otherwise hold the whole stack up.
func (s *Shell) wake() {
	s.mu.Lock()
	defer s.mu.Unlock()

	idle := s.timeout()
	if s.timer == nil {
		s.timer = time.AfterFunc(idle, s.idled)
		return
	}
	s.timer.Reset(idle)
}

// timeout is the longest any view in the stack asks for, or DockTimeout if none of them do. The
// rail is usually under a settings screen rather than replaced by it, and the screen's timeout is
// the one that matters.
func (s *Shell) timeout() time.Duration {
	var idle time.Duration
	for _, v := range s.stack {
		if t, ok := v.(Timeouter); ok {
			idle = max(idle, t.Timeout())
		}
	}
	if idle == 0 {
		return DockTimeout
	}
	return idle
}

// on follows a finger. Contacts rather than finished gestures, because a slider is pulled while it
// is being touched rather than once it has been let go.
func (s *Shell) on(c touch.Contact) {
	s.mu.Lock()
	open := len(s.stack) > 0
	top := View(nil)
	if open {
		top = s.stack[len(s.stack)-1]
	}
	moving := s.sliding != nil
	s.mu.Unlock()

	// A page part way across is not where it looks, so a touch would land on the wrong thing.
	if !open || moving {
		return
	}
	s.wake()

	switch c.Phase {
	case touch.Down:
		s.down(top, c)
	case touch.Move:
		s.move(top, c)
	case touch.Up:
		s.up(top, c)
	}
}

func (s *Shell) down(top View, c touch.Contact) {
	var held Dragger
	if d, ok := top.(Dragger); ok && d.Grab(c.X, c.Y) {
		held = d
	}

	edged := atBackEdge(c.X)
	s.mu.Lock()
	s.held, s.downX, s.downY, s.edged = held, c.X, c.Y, edged
	s.moved, s.swept, s.dragged = false, false, false
	s.scrolling, s.lastY = false, c.Y
	s.mu.Unlock()

	// Nothing draggable took it, so it may yet be a tap and is worth showing as one.
	if p, ok := top.(Presser); ok && held == nil && p.Press(c.X, c.Y, true) {
		s.repaint(p)
	}
}

// unpress takes the mark off whatever the finger was on.
func (s *Shell) unpress(top View) {
	if p, ok := top.(Presser); ok && p.Press(0, 0, false) {
		s.repaint(p)
	}
}

func (s *Shell) move(top View, c touch.Contact) {
	s.mu.Lock()
	held := s.held
	dx, dy := abs(c.X-s.downX), abs(c.Y-s.downY)

	far := dx > still || dy > still
	if far {
		s.moved = true
	}

	sc, scrollable := top.(Scroller)
	if scrollable && !s.scrolling && !s.swept && !s.dragged && far && dy > dx {
		s.scrolling, s.held, held = true, nil, nil
	}

	// Across, not just far: a screen is put away sideways, the way the chevron at its top left says.
	if !s.scrolling && s.edged && dx > sweep && dx > dy {
		s.swept = true
	}

	drag := held != nil && s.moved
	if drag {
		s.dragged = true
	}
	by := 0
	if s.scrolling {
		by, s.lastY = s.lastY-c.Y, c.Y
	}
	s.mu.Unlock()

	// A finger that has wandered is no longer on what it went down on.
	if far && held == nil {
		s.unpress(top)
	}

	if drag && held.Drag(c.X, c.Y) {
		s.repaint(held)
	}
	if by != 0 && sc.Scroll(by) {
		s.repaint(top)
	}
}

func (s *Shell) up(top View, c touch.Contact) {
	s.mu.Lock()
	held, moved, swept, dragged, scrolling := s.held, s.moved, s.swept, s.dragged, s.scrolling
	s.held, s.scrolling = nil, false
	s.mu.Unlock()

	s.unpress(top)

	switch {
	case scrolling:

	case dragged:
		// The pull is already applied; letting go is not also a tap.

	case held != nil:
		// Put down and lifted on something draggable without moving: a tap on the track, which
		// sets the level where it landed.
		if held.Drag(c.X, c.Y) {
			s.repaint(held)
		}

	case swept:
		// A sideways swipe that grabbed nothing goes back one, the same as the header does. Pop
		// closes when there is nothing under it.
		s.Pop()

	case moved:
		// Too far to be a tap, not sideways enough to be a swipe. The row under the finger now is
		// not the one it went down on, so nothing.

	case !top.Tap(c.X, c.Y):
		// The view did not want it. A page takes every touch on itself, so this is the rail being
		// tapped beside rather than on, which is how it is put away.
		s.Close()

	default:
		s.Redraw()
	}

	if r, ok := held.(Releaser); ok && !scrolling && r.Let(c.X, c.Y) {
		s.repaint(held)
	}
}

func atBackEdge(x int) bool {
	w := panelWidth()
	return x >= w-w/edgeShare
}

// panelWidth is how wide the picture is viewed.
var panelWidth = func() int {
	w, _ := display.Get().Size()
	return w
}

const edgeShare = 8

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// repaint draws again, saying what changed when the view knows. Everything else goes through
// Redraw, which repaints the lot.
func (s *Shell) repaint(v any) {
	d, ok := v.(Damager)
	if !ok {
		s.Redraw()
		return
	}

	damage := d.Damaged()
	if damage.W <= 0 || damage.H <= 0 {
		s.Redraw()
		return
	}

	s.mu.Lock()
	claim := s.claim
	s.mu.Unlock()

	if claim == nil {
		s.Redraw()
		return
	}

	claim.Refresh(display.Rect{X: damage.X, Y: damage.Y, W: damage.W, H: damage.H})
}

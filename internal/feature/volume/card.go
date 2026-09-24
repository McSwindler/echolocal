package volume

import (
	"slices"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// How long the card stays up with nobody touching it. Opened out it stays longer.
const (
	linger = 5 * time.Second
	opened = 8 * time.Second
)

// card is the volume control over whatever is underneath, and the timer that takes it away.
type card struct {
	shell.Touches

	mu    sync.Mutex
	held  *shell.Hold
	timer *time.Timer
	open  bool

	// stream is the one in the head, which is the one the bar moves.
	stream config.Stream
	places Places
	pulls  bool
}

func (c *card) Covers() bool { return false }

// selected is the kind of sound in the head, and whether the card is up at all.
func (c *card) selected() (config.Stream, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.held.Held() {
		return "", false
	}
	return c.stream, true
}

// show puts the card up for a kind of sound, or brings it round to that one.
func (c *card) show(s config.Stream) {
	c.mu.Lock()
	c.stream = s

	if !c.held.Held() {
		c.held = shell.Get().Hold(c)
		c.mu.Unlock()

		c.wait()
		return
	}
	bounds := c.places.Bounds()
	c.mu.Unlock()

	repaint(bounds)
	c.wait()
}

// repaint is a variable so a test can watch it being asked.
var repaint = func(at ui.Rect) {
	if at.W > 0 && at.H > 0 {
		shell.Get().RedrawIn(at)
		return
	}
	shell.Get().Redraw()
}

// wait restarts the countdown.
func (c *card) wait() {
	c.mu.Lock()
	defer c.mu.Unlock()

	stay := linger
	if c.open {
		stay = opened
	}

	if c.timer == nil {
		c.timer = time.AfterFunc(stay, c.hide)
		return
	}
	c.timer.Reset(stay)
}

// hide takes the card away and folds it back.
func (c *card) hide() {
	c.mu.Lock()
	held := c.held
	c.held, c.open, c.pulls = nil, false, false

	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()

	held.Release()
}

func (c *card) Draw(s ui.Surface, palette theme.Theme) {
	streams := config.Streams()

	levels := make([]int, len(streams))
	for i, stream := range streams {
		levels[i] = Get().Level(stream)
	}

	c.mu.Lock()
	open, stream := c.open, c.stream
	c.mu.Unlock()

	selected := max(0, slices.Index(streams, stream))

	places := DrawCard(s, Card{
		Streams:  streams,
		Levels:   levels,
		Selected: selected,
		Edge:     edge(),
		Open:     open,
	}, palette, c.Pressed)

	c.mu.Lock()
	c.places = places
	c.mu.Unlock()

	spots := []shell.Spot{{At: places.Head, Do: c.toggle}}
	for i, at := range places.Others {
		if at.W == 0 {
			continue
		}
		spots = append(spots, shell.Spot{At: at, Do: c.selects(streams[i])})
	}
	c.Spots(spots)
}

// toggle shows the kinds the bar is not moving, or folds them away again.
func (c *card) toggle() {
	c.mu.Lock()
	c.open = !c.open
	c.mu.Unlock()

	// The whole screen: the card is about to be a different size, and the part it gives up is not
	// in any rectangle it can work out for itself.
	repaint(ui.Rect{})
	c.wait()
}

// selects moves the bar to another kind of sound and folds the list away.
func (c *card) selects(s config.Stream) func() {
	return func() {
		c.mu.Lock()
		c.stream, c.open = s, false
		c.mu.Unlock()

		repaint(ui.Rect{})
		c.wait()
	}
}

// Tap off the card puts it away.
func (c *card) Tap(x, y int) bool {
	if c.Tapped(x, y) {
		return true
	}

	c.mu.Lock()
	on := c.places.Capsule.Contains(x, y) || c.places.Head.Contains(x, y)
	c.mu.Unlock()

	if !on {
		c.hide()
		return true
	}

	c.wait()
	return true
}

// Grab takes a finger going down on the bar.
func (c *card) Grab(x, y int) bool {
	c.mu.Lock()
	track := c.places.Track
	c.pulls = track.Contains(x, y)
	pulls := c.pulls
	c.mu.Unlock()

	if !pulls {
		return false
	}

	c.Dragging(track)
	c.wait()
	return true
}

// Drag sets the level the finger is at, and says whether that changed anything.
func (c *card) Drag(_, y int) bool {
	c.mu.Lock()
	pulls, track, stream := c.pulls, c.places.Track, c.stream
	c.mu.Unlock()

	if !pulls {
		return false
	}

	want := LevelAt(track, y)
	if want == Get().Level(stream) {
		return false
	}

	Get().Set(stream, want)
	c.wait()
	return true
}

// Damaged is the whole card while the bar is being pulled. The number sits outside the track.
func (c *card) Damaged() ui.Rect {
	c.mu.Lock()
	pulls, places := c.pulls, c.places
	c.mu.Unlock()

	if pulls {
		return places.Bounds()
	}
	return c.Touches.Damaged()
}

// Shows reports that this level is the one already on the bar.
func (c *card) Shows(s config.Stream) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.held.Held() && c.stream == s
}

// edge is the side the card hangs off: the one the drawer does not come in from.
func edge() config.Edge {
	if config.Get().Screen.Drawer == config.EdgeLeft {
		return config.EdgeRight
	}
	return config.EdgeLeft
}

// Package buttons owns the controls on the outside of the device: the four buttons on top, and the
// camera cover on a board that has one.
//
// It reads the input nodes and says what happened. It does not know what any of it means: that a long
// press of the action button reaches the second assistant belongs to whatever is listening, and
// keeping the two apart is what lets the buttons work when nothing else does.
package buttons

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/lib/hook"
	"github.com/ygelfand/echolocal/internal/lib/input"
	"github.com/ygelfand/echolocal/internal/service"
)

func init() {
	// Early: the buttons should work whatever else is wrong, so they must not be downstream of a
	// network listener or lost to one read error.
	component.Register(component.Hardware, Get, component.Order(10),
		component.Supervise(service.Restart(time.Second, 30*time.Second)))
}

// Name is the control, which the board names: the same button is a different code per board.
type Name = board.Key

const (
	Mute       = board.Mute
	VolumeDown = board.VolumeDown
	VolumeUp   = board.VolumeUp
	Action     = board.Action
	Power      = board.Power
)

// cronos's driver sends KEY_POWER down and up 40µs apart about a second after each cut; real presses measured 90-230ms.
const instant = time.Millisecond

// codes is what a board that says nothing reports.
var codes = map[uint16]Name{
	113: Mute,
	114: VolumeDown,
	115: VolumeUp,
	138: Action,
}

// keyFor is the control a code stands for. The board is asked first, so a board that reuses a
// keycode is read its own way and every other board keeps the defaults untouched.
func keyFor(code uint16) (Name, bool) {
	if n, ok := component.Board().Keys[code]; ok {
		return n, true
	}
	n, ok := codes[code]
	return n, ok
}

// LongPress is how long a button must be held to count as held rather than pressed, and
// RepeatInterval how fast a repeating button ramps while it is down.
const (
	LongPress      = 700 * time.Millisecond
	RepeatInterval = 200 * time.Millisecond
)

// Kind is what happened to a button.
type Kind string

const (
	// Tap is a short press. A repeating button reports it the moment it goes down, so a tap moves one
	// step; anything else reports it on release, once its length is known.
	Tap Kind = "tap"

	// Hold is the button still being down after LongPress. It is reported once, as it happens rather
	// than on release: waiting for release to act on a hold feels broken.
	Hold Kind = "hold"

	// Repeat is a repeating button still being down, every RepeatInterval.
	Repeat Kind = "repeat"
)

// Event is one thing a button did.
type Event struct {
	Name Name
	Kind Kind
}

func (e Event) String() string { return string(e.Name) + " " + string(e.Kind) }

// repeats reports whether holding a button should keep acting. Volume ramps; nothing else does.
func repeats(n Name) bool { return n == VolumeUp || n == VolumeDown }

// Controller reads the buttons for the life of the process.
//
// One press usually means several things — the device acts on it, Home Assistant hears about it —
// so it is a hook rather than a callback. Listeners run on the reader goroutine and must not block:
// one that waits on something is one that stops the next button working.
type Controller struct {
	Events hook.Hook[Event]

	// Shutter carries the camera cover's position, true when the camera is covered.
	Shutter hook.Hook[bool]

	mu      sync.Mutex
	devices []*input.Device
	shutter bool
	covered bool

	cut, cutKnown bool
}

var (
	once   sync.Once
	shared *Controller
)

// Get is the buttons. There is one set.
func Get() *Controller {
	once.Do(func() { shared = &Controller{} })
	return shared
}

func (c *Controller) Name() string { return "buttons" }

// Start opens the input nodes. It is an error to find none: the buttons are the one part of the
// device that should work whatever else is wrong, so silently having none is worth a restart.
func (c *Controller) Start(context.Context) error {
	devices, err := input.List()
	if err != nil {
		return fmt.Errorf("buttons: listing input devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("buttons: no input devices")
	}

	shutter := component.Board().Has(board.Shutter)

	c.mu.Lock()
	c.devices = devices
	c.shutter = shutter
	c.mu.Unlock()

	if shutter {
		c.findShutter(devices)
	}
	c.findMute(devices)

	slog.Debug("buttons ready", "devices", len(devices))
	return nil
}

// findShutter reads where the cover is sitting now.
func (c *Controller) findShutter(devices []*input.Device) {
	for _, d := range devices {
		if !d.HasSwitch(input.SwCameraLensCover) {
			continue
		}
		open, err := d.Switch(input.SwCameraLensCover)
		if err != nil {
			slog.Warn("reading the camera cover", "device", d.Path, "err", err)
			return
		}
		c.setCovered(!open)
		return
	}
	slog.Warn("no camera cover on a board that should have one")
}

// findMute reads where the microphone cut is now, on a board whose driver reports it as a switch.
func (c *Controller) findMute(devices []*input.Device) {
	for _, d := range devices {
		if !d.HasSwitch(input.SwMuteDevice) {
			continue
		}
		cut, err := d.Switch(input.SwMuteDevice)
		if err != nil {
			slog.Warn("reading the microphone cut", "device", d.Path, "err", err)
			return
		}
		c.setCut(cut)
		return
	}
}

func (c *Controller) setCut(cut bool) {
	c.mu.Lock()
	c.cut, c.cutKnown = cut, true
	c.mu.Unlock()
}

// MuteSwitch is the microphone cut as the input switch last reported it; known is false without one.
func (c *Controller) MuteSwitch() (cut, known bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cut, c.cutKnown
}

// setCovered records the cover's position and tells anyone listening, if it moved.
func (c *Controller) setCovered(covered bool) {
	c.mu.Lock()
	changed := c.covered != covered
	c.covered = covered
	c.mu.Unlock()

	if changed {
		slog.Info("camera cover", "covered", covered)
		c.Shutter.Emit(covered)
	}
}

// Covered is whether the camera is covered, and false on a board with no cover.
func (c *Controller) Covered() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.covered
}

// Close releases the nodes, which is also what unblocks the readers: a read on an input node waits
// for a key, so there is no stopping one except by closing it underneath.
func (c *Controller) Close() error {
	c.mu.Lock()
	devices := c.devices
	c.devices = nil
	c.mu.Unlock()

	for _, d := range devices {
		_ = d.Close()
	}
	return nil
}

// Run reads until ctx is cancelled, or until a node fails. A failure is returned rather than logged
// so the supervisor reopens the nodes: a device that has gone away and come back is the usual reason,
// and a reader that has quietly exited leaves the device with dead buttons.
func (c *Controller) Run(ctx context.Context) error {
	c.mu.Lock()
	devices := c.devices
	c.mu.Unlock()

	failed := make(chan error, len(devices))
	for _, d := range devices {
		go func(d *input.Device) { failed <- c.watch(ctx, d) }(d)
	}

	// Returning on cancellation without waiting is deliberate: the readers are blocked in a read that
	// only Close can interrupt, and the supervisor closes after Run returns.
	select {
	case <-ctx.Done():
		return nil
	case err := <-failed:
		return err
	}
}

// held tracks one button between its press and its release. This keypad emits no autorepeat, only
// press and release, so holding is timed here rather than counted from repeat events.
type held struct {
	at     time.Duration
	mu     sync.Mutex
	long   bool
	timer  *time.Timer
	ticker *time.Ticker
	done   chan struct{}
}

func (h *held) stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.timer != nil {
		h.timer.Stop()
	}
	if h.done != nil {
		close(h.done)
		h.done = nil
	}
	if h.ticker != nil {
		h.ticker.Stop()
	}
}

func (c *Controller) emit(name Name, kind Kind) {
	c.Events.Emit(Event{Name: name, Kind: kind})
}

func (c *Controller) watch(ctx context.Context, d *input.Device) error {
	down := map[uint16]*held{}
	defer func() {
		for _, h := range down {
			h.stop()
		}
	}()

	for {
		e, err := d.Read()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("buttons: reading %s: %w", d.Path, err)
		}
		switch e.Type {
		case input.EvKey:
			c.key(e, down)
		case input.EvSw:
			if c.shutter && e.Code == input.SwCameraLensCover {
				c.setCovered(!open(e.Value))
			}
			if e.Code == input.SwMuteDevice {
				c.setCut(e.Value != 0)
			}
		}
	}
}

// 0 is covered and 1 is open, the opposite way up from the switch's name. Measured on checkers.
func open(value int32) bool { return value != 0 }

func (c *Controller) key(e input.Event, down map[uint16]*held) {
	name, ok := keyFor(e.Code)
	if !ok {
		return
	}

	switch e.Value {
	case 1:
		h := c.pressed(name)
		h.at = e.At()
		down[e.Code] = h
	case 0:
		h, ok := down[e.Code]
		if !ok {
			return
		}
		delete(down, e.Code)
		if name == Mute && e.At()-h.at < instant {
			name = Power
		}
		c.released(name, h)
	}
}

// pressed starts tracking a button that has just gone down.
func (c *Controller) pressed(name Name) *held {
	h := &held{}

	// A repeating button acts at once and then ramps, so a tap moves one step.
	if repeats(name) {
		c.emit(name, Tap)

		h.done = make(chan struct{})
		h.ticker = time.NewTicker(RepeatInterval)
		go func(done <-chan struct{}, t *time.Ticker) {
			for {
				select {
				case <-done:
					return
				case <-t.C:
					c.emit(name, Repeat)
				}
			}
		}(h.done, h.ticker)
	}

	h.timer = time.AfterFunc(LongPress, func() {
		h.mu.Lock()
		h.long = true
		h.mu.Unlock()

		// A repeating button keeps ramping until it is let go; stopping here would end the ramp a few
		// ticks in.
		if !repeats(name) {
			h.stop()
		}
		c.emit(name, Hold)
	})
	return h
}

// released finishes a button. A tap is reported here for anything that did not already report one on
// the way down, and nothing is reported for a button that was held: the hold was the event.
func (c *Controller) released(name Name, h *held) {
	h.stop()

	h.mu.Lock()
	long := h.long
	h.mu.Unlock()

	if long || repeats(name) {
		return
	}
	c.emit(name, Tap)
}

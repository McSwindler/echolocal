// Package touch reads the touchscreen.
//
// A Goodix gt9xx in multi-touch protocol B. It reports in the panel's own portrait, so positions are
// turned into the coordinates the drawing uses before anyone sees them.
package touch

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/hardware/screen"
	"github.com/ygelfand/echolocal/internal/lib/hook"
	"github.com/ygelfand/echolocal/internal/lib/input"
	"github.com/ygelfand/echolocal/internal/service"
)

func init() {
	component.Register(component.Hardware, Get, component.Order(30),
		component.Needs(board.Panel),
		component.Supervise(service.Restart(time.Second, time.Minute)))
}

// Name is what the driver calls itself.
const Name = "goodix-ts"

// Contacts is every finger, as it happens.
var Contacts hook.Hook[Contact]

// Gestures is what those fingers amounted to: a tap, or a swipe and which edge it came from.
var Gestures hook.Hook[Gesture]

type Touch struct {
	mu  sync.Mutex
	dev *input.Device
	rec *Recognizer
}

// recognize feeds a contact to the recognizer, sized from the panel the first time it is needed.
// The rotation is fixed on these boards, so it is never rebuilt.
func (t *Touch) recognize(c Contact) (Gesture, bool) {
	t.mu.Lock()
	if t.rec == nil {
		w, h := 0, 0
		if p := screen.Get().Panel(); p != nil {
			w, h = p.Width, p.Height
		}
		t.rec = NewRecognizer(w, h)
	}
	rec := t.rec
	t.mu.Unlock()

	return rec.Feed(c)
}

var (
	once   sync.Once
	shared *Touch
)

func Get() *Touch { once.Do(func() { shared = &Touch{} }); return shared }

func (t *Touch) Name() string { return "touch" }

func (t *Touch) Start(context.Context) error {
	d, err := find()
	if err != nil {
		// A device that cannot be touched still answers, so this is said rather than fatal.
		slog.Error("the touchscreen would not open", "err", err)
		return nil
	}

	t.mu.Lock()
	t.dev = d
	t.mu.Unlock()

	slog.Info("touchscreen", "name", d.Name, "path", d.Path)
	return nil
}

func (t *Touch) Startup() component.Progress {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.dev == nil {
		return component.Progress{Failed: true, Doing: "no touchscreen"}
	}
	return component.Progress{Done: true}
}

func (t *Touch) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.dev == nil {
		return nil
	}
	err := t.dev.Close()
	t.dev = nil
	return err
}

// Run reports contacts until the device goes away, which is what the supervisor restarts on.
func (t *Touch) Run(ctx context.Context) error {
	t.mu.Lock()
	d := t.dev
	t.mu.Unlock()

	if d == nil {
		<-ctx.Done()
		return nil
	}

	dec := decoder{place: place}
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		e, err := d.Read()
		if err != nil {
			return fmt.Errorf("touch: reading %s: %w", d.Path, err)
		}
		for _, c := range dec.event(e, time.Now()) {
			t.Feed(c)
		}
	}
}

// Feed puts one contact through, as a finger on the panel and as whatever gesture it completes.
func (t *Touch) Feed(c Contact) {
	Contacts.Emit(c)

	if g, ok := t.recognize(c); ok {
		Gestures.Emit(g)
	}
}

// place turns a panel position into a viewed one. The touchscreen reports the same 480x960 the
// framebuffer is, so there is nothing to scale.
//
// Without a panel there is no rotation to apply and the raw position is the best that can be said.
// Said once: passing these off as viewed coordinates silently would put every touch in the wrong
// place on a device where nothing looked broken.
func place(px, py int) (int, int) {
	p := screen.Get().Panel()
	if p == nil {
		unplaced.Do(func() {
			slog.Warn("no panel, so touches are reported where the hardware put them")
		})
		return px, py
	}
	return p.Viewed(px, py)
}

var unplaced sync.Once

func find() (*input.Device, error) {
	devices, err := input.List()
	if err != nil {
		return nil, err
	}

	var found *input.Device
	for _, d := range devices {
		if d.Name == Name {
			found = d
			continue
		}
		d.Close()
	}
	if found == nil {
		return nil, fmt.Errorf("touch: no input device called %q", Name)
	}
	return found, nil
}

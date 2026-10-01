// Package theme is which palette the panel draws with.
//
// It holds the choice and nothing else. What is drawn, and where, belongs to whatever is drawing
// it, so a new screen costs nothing to theme and a new theme costs nothing to apply.
package theme

import (
	"log/slog"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/lib/hook"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

func init() {
	// Before anything that draws, so the first frame is already in the chosen palette.
	component.Register(component.Device, Get, component.Order(5), component.Needs(board.Panel))
}

// Changed fires when the palette changes, for whatever is showing something in it.
var Changed hook.Hook[theme.Theme]

type Theme struct {
	sel *esphome.Select

	mu  sync.RWMutex
	now theme.Theme
}

var (
	once   sync.Once
	shared *Theme
)

func Get() *Theme { once.Do(func() { shared = build() }); return shared }

func build() *Theme {
	t := &Theme{now: theme.Default()}

	t.sel = &esphome.Select{
		Base: esphome.Base{
			ObjectID: "theme",
			Name:     "Theme",
			Icon:     "mdi:palette",
			Category: esphome.CategoryConfig,
		},
		Options: theme.Names(),
	}
	t.sel.OnCommand = t.choose

	return t
}

func (t *Theme) Name() string { return "theme" }

func (t *Theme) Entities() []esphome.Entity { return []esphome.Entity{t.sel} }

// Current is the palette in force.
func (t *Theme) Current() theme.Theme {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.now
}

func (t *Theme) Restore(c config.Config) {
	t.apply(c.Screen.Theme)
	slog.Info("restored", "what", t.sel.ObjectID, "using", t.sel.Get())
}

// Choose applies a palette and remembers it, which is what the panel's own settings do. Home
// Assistant comes through the same path, so both ends agree on what is showing.
func (t *Theme) Choose(name string) { t.choose(name) }

func (t *Theme) choose(name string) {
	t.apply(name)

	if err := config.Set().Screen().Theme(t.sel.Get()); err != nil {
		slog.Error("saving a setting failed", "setting", t.sel.ObjectID, "err", err)
	}
	slog.Info("setting changed", "setting", t.sel.ObjectID, "using", t.sel.Get())
}

// apply settles on a palette and says so. A name this build does not have keeps the default rather
// than leaving the panel drawing with nothing.
func (t *Theme) apply(name string) {
	found, ok := theme.ByName(name)
	if !ok {
		if name != "" {
			slog.Warn("no such theme, using the default", "asked", name, "using", theme.DefaultName)
		}
		found = theme.Default()
	}

	t.mu.Lock()
	t.now = found
	t.mu.Unlock()

	t.sel.Set(found.Name)
	Changed.Emit(found)
	display.Get().Repaint()
}

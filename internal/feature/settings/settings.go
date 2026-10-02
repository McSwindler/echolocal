// Package settings is what the device can be set to, on the device itself.
//
// A screen is a title and a list of rows, and a row that leads somewhere pushes another screen.
// Drilling down rather than showing categories beside their contents: this panel is read from
// across a room, and two panes would halve everything on it.
package settings

import (
	"slices"
	"sync"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/drawer"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
)

func init() {
	component.Register(component.Device, Get, component.Order(36))
}

// Settings holds nothing: every screen reads the config when it draws.
type Settings struct{}

var (
	once   sync.Once
	shared *Settings
)

func Get() *Settings {
	once.Do(func() {
		shared = &Settings{}

		drawer.Get().Add(drawer.Entry{
			Name:  func() string { return say.T("settings.title") },
			Order: drawer.OrderSettings,
			Glyph: func() string { return gogui.IconGear },
			Open:  Open,
		})
	})
	return shared
}

func (s *Settings) Name() string { return "settings" }

// Open puts the top of the settings up.
func Open() { shell.Get().Push(root()) }

var pages = map[string]func() *shell.Page{"settings": root}

// OpenPage puts up a page by name, and reports whether there is one.
func OpenPage(name string) bool {
	page, ok := pages[name]
	if ok {
		shell.Get().Push(page())
	}
	return ok
}

// Pages is the names OpenPage knows.
func Pages() []string {
	out := make([]string, 0, len(pages))
	for name := range pages {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

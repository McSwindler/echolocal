// Package settings is what the device can be set to, on the device itself.
//
// A screen is a title and a list of rows, and a row that leads somewhere pushes another screen.
// Drilling down rather than showing categories beside their contents: this panel is read from
// across a room, and two panes would halve everything on it.
package settings

import (
	"sync"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/drawer"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui"
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
			Icon:  func() ui.Icon { return icons.ActionSettings },
			Open:  Open,
		})
	})
	return shared
}

func (s *Settings) Name() string { return "settings" }

// Open puts the top of the settings up.
func Open() { shell.Get().Push(root()) }

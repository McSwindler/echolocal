package screen

import (
	"context"
	"log/slog"
	"sync"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
)

func init() {
	// Before anything that draws on it or reads a touch off it.
	component.Register(component.Hardware, Get, component.Order(4), component.Needs(board.Panel))
}

// Screen owns the panel. One handle: the mapping and the page being drawn into belong to it, so two
// would fight over which page is showing.
type Screen struct {
	mu    sync.Mutex
	panel *Panel
}

var (
	once   sync.Once
	shared *Screen
)

func Get() *Screen { once.Do(func() { shared = &Screen{} }); return shared }

func (s *Screen) Name() string { return "screen" }

func (s *Screen) Start(context.Context) error {
	p, err := Open(DefaultPath, Orientation(component.Board().PanelRotation))
	if err != nil {
		// A device that cannot draw still answers.
		slog.Error("the panel would not open", "err", err)
		return nil
	}

	s.mu.Lock()
	s.panel = p
	s.mu.Unlock()

	slog.Info("panel", "info", p.Info())
	return nil
}

func (s *Screen) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.panel == nil {
		return nil
	}
	err := s.panel.Close()
	s.panel = nil
	return err
}

// Panel is the screen, or nil where there is none.
func (s *Screen) Panel() *Panel {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.panel
}

func (s *Screen) Startup() component.Progress {
	if s.Panel() == nil {
		return component.Progress{Failed: true, Doing: "no panel"}
	}
	return component.Progress{Done: true}
}

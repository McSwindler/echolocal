// Package web is the onboarding offer: what the screen shows a device nobody has added yet.
package web

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/api"
	"github.com/ygelfand/echolocal/internal/hardware/metrics"
	"github.com/ygelfand/echolocal/internal/layout"
	"github.com/ygelfand/echolocal/internal/lib/hook"
)

func init() {
	component.Register(component.Network, Get, component.Order(50), component.Needs(board.Panel))
}

const watch = 5 * time.Second

const AdoptURL = "https://my.home-assistant.io/redirect/config_flow_start/?domain=esphome"

type Server struct {
	mu    sync.Mutex
	shown string

	Offered hook.Hook[string]
}

func (s *Server) Offering() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shown
}

var (
	once   sync.Once
	shared *Server
)

func Get() *Server {
	once.Do(func() { shared = &Server{} })
	return shared
}

func (s *Server) Name() string { return "web" }

// Run holds the offer up until the device has been adopted.
func (s *Server) Run(ctx context.Context) error {
	t := time.NewTicker(watch)
	defer t.Stop()

	for {
		s.offer()

		select {
		case <-ctx.Done():
			s.hide()
			return nil
		case <-t.C:
		}
	}
}

func (s *Server) offer() {
	if Adopted() {
		s.hide()
		return
	}

	ip := address()
	if ip == "" {
		return
	}
	addr := net.JoinHostPort(ip, strconv.Itoa(layout.Port))

	s.mu.Lock()
	if s.shown == addr {
		s.mu.Unlock()
		return
	}
	s.shown = addr
	s.mu.Unlock()
	slog.Info("waiting to be adopted", "at", addr)
	s.Offered.Emit(addr)
}

func (s *Server) hide() {
	s.mu.Lock()
	if s.shown == "" {
		s.mu.Unlock()
		return
	}
	s.shown = ""
	s.mu.Unlock()
	s.Offered.Emit("")
}

// Adopted reports whether Home Assistant has ever subscribed.
func Adopted() bool { return config.Get().API.Adopted }

// Pairing is what the code to scan carries: the device's key when it has one, the add flow otherwise.
func Pairing() string {
	if key := api.Key(); key != "" {
		return key
	}
	return AdoptURL
}

func address() string {
	for _, ip := range metrics.Addresses() {
		if ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}

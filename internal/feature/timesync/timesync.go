// Package timesync takes the time and the time zone from Home Assistant.
package timesync

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/go-esphome-device/api"
	"google.golang.org/protobuf/proto"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/layout"
	"github.com/ygelfand/echolocal/internal/lib/hook"
	"github.com/ygelfand/echolocal/internal/lib/tz"
)

// Home Assistant knows the time and is already talking to this device, so it is the source: it
// works on a network with no route out, and it is what an ESPHome device with
// `time: platform: homeassistant` uses.
//
// The protocol has the device ask and the client answer, so this both sends the request and takes
// the reply.

func init() {
	component.Register(component.Device, Get, component.Order(1))
}

// step is the drift worth stepping the clock for.
const step = time.Second

// Sync is the clock before and after one answer from Home Assistant.
type Sync struct {
	Before, After time.Time
}

// Stepped reports whether the clock was moved.
func (s Sync) Stepped() bool { return !s.After.Equal(s.Before) }

type Time struct {
	// Synced carries every answer from Home Assistant.
	Synced hook.Hook[Sync]

	mu    sync.Mutex
	asked *esphome.Conn
	tz    string

	set atomic.Bool
}

var (
	once   sync.Once
	shared *Time
)

func Get() *Time {
	once.Do(func() { shared = &Time{} })
	return shared
}

func (t *Time) Name() string { return "time" }

func (t *Time) Startup() component.Progress {
	if t.set.Load() {
		return component.Progress{Done: true, Background: true}
	}
	return component.Progress{Doing: "waiting for Home Assistant", Background: true}
}

// Handle takes the answer, and asks the question the first time a client says anything.
func (t *Time) Handle(_ context.Context, conn *esphome.Conn, msg proto.Message) error {
	if reply, ok := msg.(*api.GetTimeResponse); ok {
		t.fromHome(reply.GetEpochSeconds())
		t.zoneFromHome(reply.GetTimezone())
		return nil
	}

	// Any message means a client is there to ask. Once per connection: one that reconnects is
	// worth asking again, and after a long disconnection it is the better source.
	if t.shouldAsk(conn) {
		if err := conn.Send(&api.GetTimeRequest{}); err != nil {
			slog.Debug("asking home assistant the time failed", "err", err)
		}
	}
	return nil
}

// shouldAsk reports whether this connection has not been asked yet, and records that it is about to
// be.
func (t *Time) shouldAsk(conn *esphome.Conn) bool {
	if conn == nil {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if conn == t.asked {
		return false
	}
	t.asked = conn
	return true
}

// fromHome takes an epoch Home Assistant reported.
func (t *Time) fromHome(epoch uint32) {
	if epoch == 0 {
		return
	}

	at := time.Unix(int64(epoch), 0)
	before := time.Now()
	if offset := at.Sub(before); offset < step && offset > -step {
		t.set.Store(true)
		t.Synced.Emit(Sync{Before: before, After: before})
		return
	}

	if err := setClock(at); err != nil {
		slog.Warn("could not take the time from home assistant", "err", err)
		return
	}
	t.set.Store(true)
	slog.Info("clock set", "source", "home assistant", "now", at, "was", before.Round(time.Second))
	t.Synced.Emit(Sync{Before: before, After: at})
}

// zoneFromHome takes the zone Home Assistant reported, which is what turns a clock face from UTC
// into the time on the wall.
func (t *Time) zoneFromHome(said string) {
	if said == "" || said == t.Zone() {
		return
	}

	if err := tz.Use(said); err != nil {
		slog.Warn("could not take the time zone from home assistant", "zone", said, "err", err)
		return
	}

	t.mu.Lock()
	t.tz = said
	t.mu.Unlock()

	slog.Info("time zone set", "zone", said)
}

// Zone is the time zone Home Assistant last reported, empty before it has.
func (t *Time) Zone() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tz
}

// Valid reports whether the time is worth showing: Home Assistant has set it, or it is already later
// than this build.
func (t *Time) Valid() bool {
	if t.set.Load() {
		return true
	}
	built, err := time.Parse(time.RFC3339, layout.BuildDate)
	return err != nil || time.Now().After(built)
}

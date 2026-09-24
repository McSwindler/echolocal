package clock

import (
	"context"
	"log/slog"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/go-esphome-device/api"
	"google.golang.org/protobuf/proto"

	"github.com/ygelfand/echolocal/internal/lib/tz"
)

// Home Assistant knows the time and is already talking to this device, so it is the source: it
// works on a network with no route out, and it is what an ESPHome device with
// `time: platform: homeassistant` uses.
//
// The protocol has the device ask and the client answer, so this both sends the request and takes
// the reply.

// step is the drift worth stepping the clock for.
const step = time.Second

// Handle takes the answer, and asks the question the first time a client says anything.
func (c *Clock) Handle(_ context.Context, conn *esphome.Conn, msg proto.Message) error {
	if reply, ok := msg.(*api.GetTimeResponse); ok {
		c.fromHome(reply.GetEpochSeconds())
		c.zoneFromHome(reply.GetTimezone())
		return nil
	}

	// Any message means a client is there to ask. Once per connection: one that reconnects is
	// worth asking again, and after a long disconnection it is the better source.
	if c.shouldAsk(conn) {
		if err := conn.Send(&api.GetTimeRequest{}); err != nil {
			slog.Debug("asking home assistant the time failed", "err", err)
		}
	}
	return nil
}

// shouldAsk reports whether this connection has not been asked yet, and records that it is about to
// be.
func (c *Clock) shouldAsk(conn *esphome.Conn) bool {
	if conn == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if conn == c.asked {
		return false
	}
	c.asked = conn
	return true
}

// fromHome takes an epoch Home Assistant reported.
func (c *Clock) fromHome(epoch uint32) {
	if epoch == 0 {
		return
	}

	at := time.Unix(int64(epoch), 0)
	if offset := time.Until(at); offset < step && offset > -step {
		return
	}

	if err := setClock(at); err != nil {
		slog.Warn("could not take the time from home assistant", "err", err)
		return
	}
	slog.Info("clock set", "source", "home assistant", "now", at)
}

// zoneFromHome takes the zone Home Assistant reported, which is what turns the clock face from UTC
// into the time on the wall.
func (c *Clock) zoneFromHome(said string) {
	if said == "" || said == c.zone() {
		return
	}

	if err := tz.Use(said); err != nil {
		slog.Warn("could not take the time zone from home assistant", "zone", said, "err", err)
		return
	}

	c.mu.Lock()
	c.tz = said
	c.mu.Unlock()

	slog.Info("time zone set", "zone", said)
}

func (c *Clock) zone() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tz
}

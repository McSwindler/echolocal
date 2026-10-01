package metrics

import (
	"context"
	"net"
	"time"
)

// Settled waits until the device's addresses have stayed the same for a whole interval, and
// returns them, or nil once ctx ends.
func Settled(ctx context.Context, within time.Duration) []net.IP {
	return settled(ctx, within, Addresses)
}

func settled(ctx context.Context, within time.Duration, read func() []net.IP) []net.IP {
	was := read()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(within):
		}

		now := read()
		if len(now) > 0 && AddressKey(now) == AddressKey(was) {
			return now
		}
		was = now
	}
}

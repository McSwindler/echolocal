package board

import (
	"log/slog"
	"sync"

	"github.com/ygelfand/echolocal/internal/android/prop"
)

// deviceProp is what a device calls itself.
const deviceProp = "ro.product.device"

var (
	once      sync.Once
	detected  Board
	askDevice = func() (string, error) { return prop.Get(deviceProp) }
)

// Detect is the board this process is running on, read once.
//
// An unreadable or unrecognised property gives Biscuit. Every device in the field is one, and this
// runs on a binary that can arrive by self-update with no installer involved — a board it cannot
// identify has to keep working as the thing it almost certainly is, rather than come up with no
// hardware because a property was missing.
func Detect() Board {
	once.Do(func() {
		detected = Biscuit

		device, err := askDevice()
		if err != nil {
			slog.Warn("reading the board failed, assuming biscuit", "prop", deviceProp, "err", err)
			return
		}

		b, ok := For(device)
		if !ok {
			slog.Warn("unknown board, assuming biscuit", "device", device)
			return
		}
		detected = b
		slog.Info("board", "codename", b.Codename, "device", b.Device)
	})
	return detected
}

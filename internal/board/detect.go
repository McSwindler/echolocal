package board

import (
	"log/slog"
	"sync"

	"github.com/ygelfand/echolocal/internal/android/prop"
)

// deviceProp is what a device calls itself.
const deviceProp = "ro.product.device"

// pinned is the codename a binary was built for, set with -ldflags -X by `make build-echod BOARD=`.
// Empty asks the device, which is what an all-boards build does.
var pinned string

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
	once.Do(func() { detected = detect() })
	return detected
}

func detect() Board {
	said, err := askDevice()
	if err != nil {
		said = ""
	}

	if pinned != "" {
		b, ok := ByCodename(pinned)
		if !ok {
			slog.Error("built for a board this binary has no entry for, assuming biscuit", "pinned", pinned)
			return Biscuit
		}
		// A per-board build installed on the wrong device. The pin wins, since it is what decides
		// which packages are even linked, but this is worth shouting about.
		if said != "" && said != b.Device {
			slog.Warn("built for one board and running on another", "pinned", b.Codename, "device", said)
		}
		slog.Info("board", "codename", b.Codename, "pinned", true)
		return b
	}

	if said == "" {
		slog.Warn("reading the board failed, assuming biscuit", "prop", deviceProp, "err", err)
		return Biscuit
	}

	b, ok := For(said)
	if !ok {
		slog.Warn("unknown board, assuming biscuit", "device", said)
		return Biscuit
	}
	slog.Info("board", "codename", b.Codename, "device", b.Device)
	return b
}

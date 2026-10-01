// Package board is which device this is, and what that device has.
//
// Everything that differs between models and is not worth discovering at runtime lives here: what
// Home Assistant should call it, which Amazon service echod takes over, and the coarse capabilities
// that decide whether a whole component exists.
//
// It deliberately holds no hardware detail that the device can be asked about directly. The light
// sensor, the beamformer coefficients, the headphone jack and the privacy line are all found by
// looking, in the packages that drive them, and a board that duplicated those answers would be a
// second place for them to be wrong.
package board

import "strings"

// Cap is hardware a board either has or does not, where the difference decides whether a component
// exists at all rather than how one behaves.
type Cap uint32

const (
	// Ring is the twelve-segment LED ring. It gates the ring driver and the light and room
	// components, which have nothing to present without it.
	Ring Cap = 1 << iota

	// Panel is a screen reached through the kernel framebuffer.
	Panel

	// Wifi is a board where nothing brings the radio up on its own.
	Wifi

	// Shutter is a physical cover over the camera, reported as an EV_SW switch.
	Shutter
)

// Key is a physical control. The same button is a different code on different boards.
type Key string

const (
	Mute       Key = "mute"
	VolumeDown Key = "volume_down"
	VolumeUp   Key = "volume_up"
	Action     Key = "action"
	Power      Key = "power"
)

// Board is a model of device.
type Board struct {
	// Device is ro.product.device, which is what the device calls itself and the only thing a
	// lookup may match on. It is not always the codename: a biscuit that took the Fire OS 6 OTA
	// reports biscuit_puffin.
	Device string

	// Codename is Amazon's own name for the board, which is what people use to talk about it and
	// what the hardware_board sensor reports.
	Codename string

	// Slotted is whether the board is A/B.
	Slotted bool

	// SystemAsRoot is whether / is the system partition rather than the boot ramdisk.
	SystemAsRoot bool

	// SignalsBoot is whether the device ever sets sys.boot_completed. A board whose framework echod
	// keeps switched off never does, and waiting on it there is waiting for a timeout.
	SignalsBoot bool

	// BootHooks are the stock /system scripts echod takes over, the first of which init runs on the
	// way up and so carries the rollback. Empty on a board where nothing suitable has been found.
	BootHooks []string

	// Model is what Home Assistant shows in the device panel.
	Model string

	// DefaultName is the fallback display name for a device that has none recorded.
	DefaultName string

	// Caps is what this board has.
	Caps Cap

	// Keys is what this board's buttons report, where they are not the usual codes.
	Keys map[uint16]Key

	// MuteDir is the driver that cuts the microphones, which publishes state and enable.
	MuteDir string

	// PanelRotation turns the picture the right way up, in degrees from the framebuffer's own
	// orientation. Only somebody looking at the device can say: a capture reads back through the
	// same rotation it was drawn with, so it looks correct either way.
	PanelRotation int

	// Service is the Amazon init service echod is installed as, StockLabel the SELinux label its
	// binary carries before we replace it. Taking over a service is how echod gets init's
	// supervision, so a board with nothing suitable to take over cannot be installed to.
	Service     string
	ServiceName string
	StockLabel  string

	// color decodes the shell colour from this board's idme fields.
	color func(idme func(string) string) string
}

// Has reports whether the board carries a capability.
func (b Board) Has(c Cap) bool { return b.Caps&c != 0 }

// String is the board in logs and errors.
func (b Board) String() string {
	if b.Codename == "" {
		return "unknown"
	}
	return b.Codename
}

// NameFromMAC builds the fallback display name, unique per device.
func (b Board) NameFromMAC(mac string) string {
	var hex strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(mac)) {
		if r >= '0' && r <= '9' || r >= 'A' && r <= 'F' {
			hex.WriteRune(r)
		}
	}
	s := hex.String()
	if len(s) < 6 {
		return b.DefaultName
	}
	return b.DefaultName + " " + s[len(s)-6:]
}

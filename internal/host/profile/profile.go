// Package profile is what echoctl needs to know to install onto a board.
//
// It is host-only on purpose. Boot image hashes, partition offsets and the bytes that make an adbd
// give up root have no business in the binary that runs on the device, and echod is the one thing
// here that has to stay small.
//
// A board echoctl has a name for but no profile is refused differently from a device nobody has
// heard of.
package profile

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ygelfand/echolocal/internal/android/services"
	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/bootimg"
	"github.com/ygelfand/echolocal/internal/host/sysimg"
)

// profiles is the boards this build can install onto: those whose boot image, partition layout and
// Amazon services have been worked out.
var profiles = map[string]Profile{
	board.Biscuit.Device:  biscuit,
	board.Checkers.Device: checkers,
	board.Cronos.Device:   cronos,
	board.Rook.Device:     rook,
}

// Profile is one board's install.
type Profile struct {
	Board board.Board

	// Boot is the image the flash stage writes, described rather than carried: echoctl holds the hash
	// and fetches the bytes.
	Boot bootimg.Image

	// System is what the install changes on the system partition.
	System sysimg.Layout

	// Disable and Enable are the Amazon init services an install turns off and on.
	Disable, Enable []string

	// Hide are the Amazon packages an install hides, empty on a board with no package manager.
	Hide []services.Package

	// SupportedSDK is the ro.build.version.sdk this install was worked out against, and
	// UnsupportedSDKReason what a device on another one is told. Both default to Fire OS 6.
	SupportedSDK         string
	UnsupportedSDKReason string
}

// Fire OS 6 is Android 7.1, which is what a profile that names no SDK of its own installs to.
const (
	fireOS6    = "25"
	fireOS6Why = "want 25 (Fire OS 6); Fire OS 5 needs echoctl 0.0.6 or earlier"
)

// SDK is the ro.build.version.sdk this profile installs to.
func (p Profile) SDK() string {
	if p.SupportedSDK == "" {
		return fireOS6
	}
	return p.SupportedSDK
}

// SDKRefusal is what a device on another one is told.
func (p Profile) SDKRefusal() string {
	if p.UnsupportedSDKReason == "" {
		return fireOS6Why
	}
	return p.UnsupportedSDKReason
}

// For is the profile for what a device says it is.
//
// It returns the board alongside the error, so a caller can say what the device was and whether it
// was recognised at all.
func For(device string) (Profile, error) {
	b, known := board.For(device)
	if !known {
		return Profile{}, fmt.Errorf("profile: this device calls itself %q, which is not one this build installs to. Boards meant to be supported but never met: %s — none has had its ro.product.device recorded, and %q is what one of those entries needs if this is one of them",
			device, strings.Join(board.Intended(), ", "), device)
	}

	p, ok := profiles[b.Device]
	if !ok {
		return Profile{Board: b}, fmt.Errorf("EchoLocal does not support %s yet", b)
	}
	// Without one, the install has nothing to become and would link echod over an empty path.
	if b.ServiceName == "" {
		return Profile{Board: b}, fmt.Errorf("EchoLocal does not support %s yet", b)
	}
	return p, nil
}

// Boards is every board this build installs onto, which is the set a release publishes builds for.
func Boards() []board.Board {
	out := make([]board.Board, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, p.Board)
	}
	slices.SortFunc(out, func(a, b board.Board) int { return strings.Compare(a.Codename, b.Codename) })
	return out
}

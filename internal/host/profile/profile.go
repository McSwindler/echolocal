// Package profile is what echoctl needs to know to install onto a board.
//
// It is host-only on purpose. Boot image hashes, partition offsets and the bytes that make an adbd
// give up root have no business in the binary that runs on the device, and echod is the one thing
// here that has to stay small.
//
// A board echoctl has a name for but no profile is refused differently from a device nobody has
// heard of: "not yet" and "not one of ours" are different things to be told, and the first one is a
// pointer at what would have to be measured.
package profile

import (
	"fmt"
	"strings"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/bootimg"
	"github.com/ygelfand/echolocal/internal/host/sysimg"
)

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
		return Profile{Board: b}, fmt.Errorf("profile: %s is known but not yet supported — nobody has had one to work out its boot image or partition layout", b)
	}
	return p, nil
}

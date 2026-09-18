package profile

import (
	"github.com/ygelfand/echolocal/internal/android/services"
	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/bootimg"
	"github.com/ygelfand/echolocal/internal/host/sysimg"
)

// profiles is the boards this build can install onto. A board is in here once somebody has had one
// and worked out its boot image, its partition layout and what Amazon runs on it.
var profiles = map[string]Profile{
	board.Biscuit.Device: biscuit,
}

// biscuit is the 2nd-generation Echo Dot.
var biscuit = Profile{
	Board:  board.Biscuit,
	Boot:   bootimg.Biscuit,
	System: sysimg.Biscuit,

	Disable: services.Disabled,
	Enable:  services.Enabled,
}

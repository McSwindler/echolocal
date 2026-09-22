package profile

import (
	"github.com/ygelfand/echolocal/internal/android/services"
	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/bootimg"
	"github.com/ygelfand/echolocal/internal/host/sysimg"
)

// biscuit is the 2nd-generation Echo Dot.
var biscuit = Profile{
	Board:  board.Biscuit,
	Boot:   bootimg.Biscuit,
	System: sysimg.Biscuit,

	Disable: services.Disabled,
	Enable:  services.Enabled,
}

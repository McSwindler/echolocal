package profile

import (
	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/bootimg"
	"github.com/ygelfand/echolocal/internal/host/sysimg"
)

// cronos is the 2nd-generation Echo Show 5. Nobody has read its init services or its packages off
// one, so it takes checkers' lists: a service no rc defines and a package pm does not list are both
// left alone, which makes the shared entries cost nothing where the two differ.
var cronos = Profile{
	Board:  board.Cronos,
	Boot:   bootimg.Cronos,
	System: sysimg.Cronos,

	Disable: checkers.Disable,
	Enable:  checkers.Enable,
}

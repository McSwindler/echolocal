package profile

import (
	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/bootimg"
	"github.com/ygelfand/echolocal/internal/host/sysimg"
)

var cronos = Profile{
	Board:  board.Cronos,
	Boot:   bootimg.Cronos,
	System: sysimg.Cronos,

	Disable: checkers.Disable,
	Enable:  checkers.Enable,
}

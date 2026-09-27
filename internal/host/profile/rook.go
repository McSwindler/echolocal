package profile

import (
	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/sysimg"
)

// rook is the 1st-generation Echo Spot, which runs LineageOS. Its own boot image is kept: what an
// install changes is all on the system partition.
var rook = Profile{
	Board:  board.Rook,
	System: sysimg.Rook,

	SupportedSDK:         "30",
	UnsupportedSDKReason: "only LineageOS running Android 11 (SDK 30) is supported",
}

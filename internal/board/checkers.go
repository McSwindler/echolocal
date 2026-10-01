package board

import "github.com/ygelfand/echolocal/internal/layout"

// Checkers is the 1st-generation Echo Show 5.
var Checkers = Board{
	Device:        "checkers",
	Codename:      "checkers",
	Model:         "Echo Show 5 1st gen (checkers)",
	DefaultName:   "Echo Show 5",
	Caps:          Panel | Wifi | Shutter | Camera,
	PanelRotation: 270,

	CameraWidth: 1280, CameraHeight: 720,

	// The mute button reports KEY_POWER, on a node of its own called "gating".
	Keys:    map[uint16]Key{116: Mute},
	MuteDir: "/sys/devices/platform/amazon-gating",

	Service:     layout.Binary,
	ServiceName: layout.Service,
}

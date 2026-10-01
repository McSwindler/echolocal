package board

import "github.com/ygelfand/echolocal/internal/layout"

// Cronos is the 2nd-generation Echo Show 5.
var Cronos = Board{
	Device:        "cronos",
	Codename:      "cronos",
	Model:         "Echo Show 5 2nd gen (cronos)",
	DefaultName:   "Echo Show 5",
	Caps:          Panel | Wifi | Shutter | Camera,
	PanelRotation: 270,
	Keys:          map[uint16]Key{116: Mute},
	MuteDir:       "/sys/devices/platform/gpio-privacy",

	CameraWidth: 1280, CameraHeight: 720,

	Service:     layout.Binary,
	ServiceName: layout.Service,
}

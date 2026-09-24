package board

// Checkers is the 1st-generation Echo Show 5.
var Checkers = Board{
	Device:        "checkers",
	Codename:      "checkers",
	Model:         "Echo Show 5 1st gen (checkers)",
	DefaultName:   "Echo Show 5",
	Caps:          Panel | Wifi | Shutter,
	PanelRotation: 270,

	// The mute button reports KEY_POWER, on a node of its own called "gating".
	Keys: map[uint16]Key{116: Mute},

	// surfaceflinger holds /dev/graphics/fb0.
	Service:     "/system/bin/surfaceflinger",
	ServiceName: "surfaceflinger",
	StockLabel:  "u:object_r:surfaceflinger_exec:s0",
}

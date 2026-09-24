package board

// Cronos is the 2nd-generation Echo Show 5.
var Cronos = Board{
	Device:        "cronos",
	Codename:      "cronos",
	Model:         "Echo Show 5 2nd gen (cronos)",
	DefaultName:   "Echo Show 5",
	Caps:          Panel | Wifi | Shutter,
	PanelRotation: 270,
	Keys:          map[uint16]Key{116: Mute},

	Service:     "/system/bin/surfaceflinger",
	ServiceName: "surfaceflinger",
	StockLabel:  "u:object_r:surfaceflinger_exec:s0",
}

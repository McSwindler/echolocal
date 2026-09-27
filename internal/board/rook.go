package board

// Rook is the 1st-generation Echo Spot, which runs LineageOS rather than Fire OS.
var Rook = Board{
	Device:       "rook",
	Codename:     "rook",
	SystemAsRoot: true,
	Model:        "Echo Spot (rook)",
	DefaultName:  "Echo Spot",
	Caps:         Panel | Wifi,

	// zygote is the service echod is installed as: taking its definition is what stops the framework
	// and gets init's supervision in one change.
	Service:     "/system/app/echod/echod",
	ServiceName: "zygote",
}

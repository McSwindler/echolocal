package board

// Doppler is the 1st-generation Echo.
//
// echod is installed as a service on a custom Linux kernel.
var Doppler = Board{
	Device:       "doppler",
	Codename:     "doppler",
	Slotted:      false,
	SystemAsRoot: true,
	SignalsBoot:  false,
	Model:        "Echo (doppler)",
	DefaultName:  "Echo",
	Caps:         Ring,
}

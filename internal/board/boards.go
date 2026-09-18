package board

import (
	"strconv"
	"strings"
)

// Biscuit is the 2nd-generation Echo Dot, and the only board anyone has run this on.
//
// echod is installed as Amazon's ledcontroller: taking over the definition removes the only other
// writer of the LED ring and gets us init's supervision.
var Biscuit = Board{
	Device:      "biscuit_puffin",
	Codename:    "biscuit",
	Model:       "Echo Dot 2 (biscuit)",
	DefaultName: "Echo Dot",
	Caps:        Ring,

	Service:     "/system/bin/ledcontroller",
	ServiceName: "ledcontroller",
	StockLabel:  "u:object_r:ledd_exec:s0",

	color: func(idme func(string) string) string {
		return biscuitColor(idme("productid2"), idme("serial"))
	},
}

// The boards we mean to support and have not met. Each carries the codename and the model it goes
// with, and nothing else.
//
// Device is empty on purpose. It is ro.product.device, which is not the codename — biscuit reports
// biscuit_puffin — and there is no way to work out what one of these says without one to ask. A
// board with no Device never matches, so an unrecognised device is reported with the name it gave
// itself, and that name is what fills this in.
//
// Everything else is the same: capabilities, the service to take over, the idme encoding. None of
// them can be reasoned out from here.
var (
	Crown    = Board{Codename: "crown", Model: "Echo Show 8 (crown)", DefaultName: "Echo Show 8"}
	Checkers = Board{Codename: "checkers", Model: "Echo Show 5 (checkers)", DefaultName: "Echo Show 5"}
	Cronos   = Board{Codename: "cronos", Model: "Echo Show 5 2nd gen (cronos)", DefaultName: "Echo Show 5"}
	Donut    = Board{Codename: "donut", Model: "Echo Dot 3 (donut)", DefaultName: "Echo Dot"}
	Rook     = Board{Codename: "rook", Model: "Echo Spot (rook)", DefaultName: "Echo Spot"}
	Radar    = Board{Codename: "radar", Model: "Echo 2nd gen (radar)", DefaultName: "Echo"}
)

// All is every board this build knows by name, whether or not it can be installed to.
var All = []Board{Biscuit, Crown, Checkers, Cronos, Donut, Rook, Radar}

// For is the board a ro.product.device belongs to, and false for a device we have no name for.
//
// Only a board whose Device somebody has read off real hardware can match, which today is biscuit
// alone. The rest are named so there is somewhere obvious to put that value.
func For(device string) (Board, bool) {
	device = strings.TrimSpace(device)
	if device == "" {
		return Board{}, false
	}
	for _, b := range All {
		if b.Device == device {
			return b, true
		}
	}
	return Board{}, false
}

// Intended is the codenames this build means to support but has never met, for an error that has to
// say what a device could have been.
func Intended() []string {
	var out []string
	for _, b := range All {
		if b.Device == "" {
			out = append(out, b.Codename)
		}
	}
	return out
}

// biscuitColor works the shell out from two idme fields: productid2 is 0 on a black unit and a
// nonzero code on a white one, and the serial runs G090LF… on black against G090L9… on white. They
// confirm each other; a disagreement, or nothing to read, is left unknown rather than guessed.
func biscuitColor(productID2, serial string) string {
	prod := ColorUnknown
	if n, err := strconv.Atoi(strings.TrimSpace(productID2)); err == nil {
		if n == 0 {
			prod = ColorBlack
		} else {
			prod = ColorWhite
		}
	}

	ser := ColorUnknown
	if s := strings.TrimSpace(serial); len(s) > 5 {
		switch s[5] {
		case 'F', 'f':
			ser = ColorBlack
		case '9':
			ser = ColorWhite
		}
	}

	switch {
	case prod == ColorUnknown:
		return ser
	case ser == ColorUnknown:
		return prod
	case prod == ser:
		return prod
	default:
		return ColorUnknown
	}
}

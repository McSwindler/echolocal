package board

import (
	"strconv"
	"strings"

	"github.com/ygelfand/echolocal/internal/layout"
)

// Biscuit is the 2nd-generation Echo Dot.
//
// echod is installed as Amazon's ledcontroller: taking over the definition removes the only other
// writer of the LED ring and gets us init's supervision.
var Biscuit = Board{
	Device:       "biscuit_puffin",
	Codename:     "biscuit",
	Slotted:      true,
	SystemAsRoot: true,
	SignalsBoot:  true,
	Model:        "Echo Dot 2 (biscuit)",
	DefaultName:  "Echo Dot",
	Caps:         Ring,

	Service:     "/system/bin/ledcontroller",
	ServiceName: "ledcontroller",
	StockLabel:  "u:object_r:ledd_exec:s0",
	BootHooks:   []string{layout.StartAnimation, layout.StopAnimation},

	color: func(idme func(string) string) string {
		return biscuitColor(idme("productid2"), idme("serial"))
	},
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

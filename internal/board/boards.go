package board

import "strings"

// All is every board this build knows by name, whether or not it can be installed to.
var All = []Board{Biscuit, Crown, Checkers, Cronos, Donut, Doppler, Rook, Radar}

// For is the board a ro.product.device belongs to. A board with no Device never matches.
func For(device string) (Board, bool) {
	device = strings.TrimSpace(device)
	if device == "" {
		return Board{}, false
	}
	for _, b := range All {
		if b.Device == device || (b.Device != "" && b.Codename == device) {
			return b, true
		}
	}
	return Board{}, false
}

// ByCodename is the board Amazon calls this. It also matches a board with no Device recorded.
func ByCodename(name string) (Board, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Board{}, false
	}
	for _, b := range All {
		if b.Codename == name {
			return b, true
		}
	}
	return Board{}, false
}

// Intended is the codenames this build means to support and has no Device for.
func Intended() []string {
	var out []string
	for _, b := range All {
		if b.Device == "" {
			out = append(out, b.Codename)
		}
	}
	return out
}

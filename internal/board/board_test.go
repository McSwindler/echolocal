package board

import "testing"

func TestBiscuitColor(t *testing.T) {
	// Real serial format, fabricated tails: black units read G090LF…, white G090L9…, and the tail is
	// the part that identifies a unit, so it is zeroed rather than real.
	const (
		blackSerial = "G090LF0000000000"
		whiteSerial = "G090L90000000000"
	)

	tests := []struct {
		name                     string
		productID2, serial, want string
	}{
		{"both say black", "0", blackSerial, ColorBlack},
		{"both say white", "20", whiteSerial, ColorWhite},
		{"white high code", "2020", whiteSerial, ColorWhite},
		{"trailing space", "20 ", whiteSerial, ColorWhite},

		{"fields disagree", "0", whiteSerial, ColorUnknown},
		{"nothing to read", "", "", ColorUnknown},
		{"productid2 only", "0", "", ColorBlack},
		{"serial only", "", whiteSerial, ColorWhite},
		{"serial too short", "", "G090L", ColorUnknown},
	}
	for _, tt := range tests {
		idme := func(field string) string {
			if field == "productid2" {
				return tt.productID2
			}
			return tt.serial
		}
		if got := Biscuit.Color(idme); got != tt.want {
			t.Errorf("%s: Color(%q, %q) = %q, want %q", tt.name, tt.productID2, tt.serial, got, tt.want)
		}
	}
}

// A board nobody has decoded says so rather than guessing at black or white, which is biscuit's
// answer and not necessarily anybody else's set of answers at all.
func TestAnUndecodedBoardReportsUnknown(t *testing.T) {
	idme := func(string) string { return "whatever some other board writes here" }
	if got := Crown.Color(idme); got != ColorUnknown {
		t.Errorf("Crown.Color() = %q, want %q", got, ColorUnknown)
	}
}

// A board nobody has met must never match a device, or an install would be aimed at it on the
// strength of a codename somebody typed.
func TestOnlyBoardsReadOffHardwareCanMatch(t *testing.T) {
	for _, b := range All {
		if b.Device == "" {
			continue
		}
		if b.Device != Biscuit.Device {
			t.Errorf("%s carries a ro.product.device of %q; has one actually been read off one?", b, b.Device)
		}
	}

	// The codename is not the property, so it must not match as though it were.
	for _, name := range Intended() {
		if b, ok := For(name); ok {
			t.Errorf("For(%q) matched %s on a codename rather than a ro.product.device", name, b)
		}
	}
}

func TestFor(t *testing.T) {
	// biscuit reports biscuit_puffin once it has taken the Fire OS 6 OTA, which is why the lookup is
	// on ro.product.device rather than the codename.
	if b, ok := For("biscuit_puffin"); !ok || b.Codename != "biscuit" {
		t.Errorf(`For("biscuit_puffin") = %v, %v; want biscuit`, b, ok)
	}
	if b, ok := For("  biscuit_puffin\n"); !ok || b.Codename != "biscuit" {
		t.Errorf("For did not trim what a device said: %v, %v", b, ok)
	}
	if _, ok := For("sailfish"); ok {
		t.Error("For matched a device this build has never heard of")
	}
}

// Only biscuit has been run on. A board nobody has met must claim nothing: no service to take over,
// and no capabilities, because both are things you find out by having one in your hands.
func TestABoardNobodyHasMetClaimsNothing(t *testing.T) {
	if Biscuit.ServiceName == "" {
		t.Error("biscuit names no service for echod to be installed as")
	}
	for _, b := range All {
		if b.Device == Biscuit.Device {
			continue
		}
		if b.Service != "" || b.ServiceName != "" || b.StockLabel != "" {
			t.Errorf("%s names a service to take over, which nobody has looked at one to find", b)
		}
		if b.Caps != 0 {
			t.Errorf("%s claims capabilities %b that nobody has checked", b, b.Caps)
		}
	}
}

func TestNameFromMAC(t *testing.T) {
	tests := []struct{ in, want string }{
		{"0a:1b:2c:3d:4e:5f", "Echo Dot 3D4E5F"},
		{"0A1B2C3D4E5F", "Echo Dot 3D4E5F"},
		{"", "Echo Dot"},
		{"0a:1b", "Echo Dot"},
	}
	for _, tt := range tests {
		if got := Biscuit.NameFromMAC(tt.in); got != tt.want {
			t.Errorf("NameFromMAC(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

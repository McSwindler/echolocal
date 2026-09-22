package board

import "testing"

func withDevice(t *testing.T, said string, err error) {
	t.Helper()
	restore := askDevice
	askDevice = func() (string, error) { return said, err }
	t.Cleanup(func() { askDevice = restore })
}

func withPin(t *testing.T, name string) {
	t.Helper()
	restore := pinned
	pinned = name
	t.Cleanup(func() { pinned = restore })
}

func TestDetectReadsTheDevice(t *testing.T) {
	withDevice(t, "checkers", nil)

	if got := detect(); got.Codename != "checkers" {
		t.Errorf("detect() = %s, want checkers", got)
	}
}

// A board-specific binary knows what it is without asking, since the pin is what decided which
// packages were linked into it.
func TestAPinnedBuildDoesNotAskTheDevice(t *testing.T) {
	withPin(t, "checkers")
	withDevice(t, "biscuit_puffin", nil)

	if got := detect(); got.Codename != "checkers" {
		t.Errorf("detect() = %s, want the pinned checkers", got)
	}
}

// Pinning to a codename with no entry is a build mistake, and biscuit is the safe landing.
func TestAPinNothingMatchesFallsBackToBiscuit(t *testing.T) {
	withPin(t, "nonesuch")
	withDevice(t, "", errNoProp)

	if got := detect(); got.Codename != Biscuit.Codename {
		t.Errorf("detect() = %s, want biscuit", got)
	}
}

// Every device in the field is a biscuit, so a binary that cannot identify itself carries on as one
// rather than coming up with no hardware.
func TestDetectFallsBackToBiscuit(t *testing.T) {
	for _, tc := range []struct {
		name, said string
		err        error
	}{
		{"prop unreadable", "", errNoProp},
		{"unknown device", "sailfish", nil},
		{"no device recorded for it", "crown", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withDevice(t, tc.said, tc.err)

			if got := detect(); got.Codename != Biscuit.Codename {
				t.Errorf("detect() = %s, want biscuit", got)
			}
		})
	}
}

var errNoProp = errNotThere{}

type errNotThere struct{}

func (errNotThere) Error() string { return "no such file" }

package profile

import "testing"

// Fire OS 6 on biscuit has no package manager, so an install there must never reach for one.
func TestBiscuitHidesNothing(t *testing.T) {
	if len(biscuit.Hide) != 0 {
		t.Errorf("biscuit hides %d packages: %v", len(biscuit.Hide), biscuit.Hide)
	}
}

// Every board with a profile has to name a service, or the install links echod over an empty path.
// For refuses one that does not, which is what keeps a half-filled profile from being installable.
func TestEveryProfileNamesAService(t *testing.T) {
	for device, p := range profiles {
		if p.Board.ServiceName == "" || p.Board.Service == "" {
			t.Errorf("%s names no service for echod to take over", device)
		}
		if _, err := For(device); err != nil {
			t.Errorf("%s has a complete profile and was still refused: %v", device, err)
		}
	}

	// A device this build has no name for is refused by name rather than by profile.
	if _, err := For("sailfish"); err == nil {
		t.Error("a device nobody has heard of resolved a profile")
	}
}

// Hiding the same package twice is harmless but means two lists drifted, which is worth knowing.
func TestHiddenListsAreDistinct(t *testing.T) {
	for device, p := range profiles {
		seen := make(map[string]bool, len(p.Hide))
		for _, pkg := range p.Hide {
			if seen[pkg.Name] {
				t.Errorf("%s hides %s twice", device, pkg.Name)
			}
			seen[pkg.Name] = true

			if pkg.Reason == "" {
				t.Errorf("%s hides %s with no reason", device, pkg.Name)
			}
		}
	}
}

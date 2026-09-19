package update

import (
	"strings"
	"testing"
)

// A release cut before Boards existed has nothing to offer a device that reads Boards. The shared
// fields are still there and still valid — they are how the devices in the field update — but taking
// one here would mean installing whatever build happened to be shared.
func TestForRefusesAReleaseThatNamesNoBoards(t *testing.T) {
	m := Manifest{
		Version: "0.0.6",
		URL:     "https://example/echod",
		SHA256:  strings.Repeat("a", 64),
		Size:    24 << 20,
	}
	if err := m.Valid(); err != nil {
		t.Fatal(err)
	}

	if _, err := m.For("biscuit", "arm64"); err == nil {
		t.Error("offered a build from a release that names no boards")
	}
}

// A build meant for one board is never handed to another, which is the whole reason a board takes
// only what names it.
func TestForNeverOffersAnotherBoardsBuild(t *testing.T) {
	m := Manifest{
		Version:  "0.1.0",
		URL:      "https://example/echod-arm64",
		SHA256:   strings.Repeat("a", 64),
		Size:     24 << 20,
		Binaries: map[string]Binary{"arm": {URL: "https://example/echod-arm", SHA256: strings.Repeat("b", 64), Size: 22 << 20}},
		Boards: map[string]map[string]Binary{
			"biscuit": {"arm": {URL: "https://example/echod-biscuit-arm", SHA256: strings.Repeat("c", 64), Size: 22 << 20}},
		},
	}
	if err := m.Valid(); err != nil {
		t.Fatal(err)
	}

	b, err := m.For("biscuit", "arm")
	if err != nil {
		t.Fatal(err)
	}
	if b.URL != "https://example/echod-biscuit-arm" {
		t.Errorf("biscuit was offered %s", b.URL)
	}

	// crown is not in this release. The shared arm build exists and must not be what it gets.
	if _, err := m.For("crown", "arm"); err == nil {
		t.Error("a board the release does not carry was offered the shared build")
	}
}

// Every device takes the build for what it is running, and a release carrying both must not hand
// either one the other's.
func TestForTakesTheArchitectureTheDeviceRuns(t *testing.T) {
	m := Manifest{
		Version: "0.0.7",
		URL:     "https://example/echod-arm64",
		SHA256:  strings.Repeat("a", 64),
		Size:    24 << 20,
		Binaries: map[string]Binary{
			"arm64": {URL: "https://example/echod-arm64", SHA256: strings.Repeat("a", 64), Size: 24 << 20},
			"arm":   {URL: "https://example/echod-arm", SHA256: strings.Repeat("b", 64), Size: 22 << 20},
		},
		Boards: map[string]map[string]Binary{
			"biscuit": {
				"arm64": {URL: "https://example/echod-arm64", SHA256: strings.Repeat("a", 64), Size: 24 << 20},
				"arm":   {URL: "https://example/echod-arm", SHA256: strings.Repeat("b", 64), Size: 22 << 20},
			},
		},
	}
	if err := m.Valid(); err != nil {
		t.Fatal(err)
	}

	for arch, want := range map[string]string{
		"arm64": "https://example/echod-arm64",
		"arm":   "https://example/echod-arm",
	} {
		b, err := m.For("biscuit", arch)
		if err != nil {
			t.Errorf("%s: %v", arch, err)
			continue
		}
		if b.URL != want {
			t.Errorf("%s: offered %s, want %s", arch, b.URL, want)
		}
	}
}

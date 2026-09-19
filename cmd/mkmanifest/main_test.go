package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ygelfand/echolocal/internal/host/profile"
	"github.com/ygelfand/echolocal/internal/update"
)

// deployed is the manifest as every device released before the binaries map existed parses it. Those
// devices update through these four fields and nothing else, so a release that stops filling them in
// leaves every one of them stuck on the build it is running.
type deployed struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

func write(t *testing.T) (deployed, update.Manifest) {
	t.Helper()
	dir := t.TempDir()

	for name, body := range map[string]string{"echod-arm64": "sixty four", "echod-arm": "thirty two"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	out := filepath.Join(dir, "manifest.json")
	err := run(update.Manifest{Version: "0.0.7"}, "https://example/download/0.0.7", map[string]string{
		"arm64": filepath.Join(dir, "echod-arm64"),
		"arm":   filepath.Join(dir, "echod-arm"),
	}, nil, out)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	var old deployed
	var now update.Manifest
	if err := json.Unmarshal(encoded, &old); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &now); err != nil {
		t.Fatal(err)
	}
	return old, now
}

func TestADeployedDeviceCanStillReadIt(t *testing.T) {
	old, now := write(t)

	if old.Version == "" || old.URL == "" || len(old.SHA256) != 64 || old.Size <= 0 {
		t.Fatalf("a device on the old manifest reads %+v, which it will refuse", old)
	}

	// Those fields have to be the arm64 build: every deployed device is one.
	arm64 := now.Binaries["arm64"]
	if old.URL != arm64.URL || old.SHA256 != arm64.SHA256 || old.Size != arm64.Size {
		t.Errorf("the top-level fields describe %+v, want the arm64 build %+v", old, arm64)
	}
}

func TestEachArchitectureGetsItsOwnBuild(t *testing.T) {
	_, now := write(t)

	if err := now.Valid(); err != nil {
		t.Fatal(err)
	}

	for arch, want := range map[string]string{
		"arm64": "https://example/download/0.0.7/echod-arm64",
		"arm":   "https://example/download/0.0.7/echod-arm",
	} {
		b, err := now.For("biscuit", arch)
		if err != nil {
			t.Errorf("%s: %v", arch, err)
			continue
		}
		if b.URL != want {
			t.Errorf("%s: offered %s, want %s", arch, b.URL, want)
		}
	}

	if now.Binaries["arm64"].SHA256 == now.Binaries["arm"].SHA256 {
		t.Error("both architectures were measured as the same file")
	}
}

// A board missing from the manifest is a board that stops seeing updates, so every board echoctl
// installs onto has to be in there — without anyone remembering to add it.
func TestEverySupportedBoardGetsABuild(t *testing.T) {
	_, now := write(t)

	for _, b := range profile.Boards() {
		builds, ok := now.Boards[b.Codename]
		if !ok {
			t.Errorf("%s is installable but the manifest carries no build for it", b.Codename)
			continue
		}
		for _, arch := range []string{"arm64", "arm"} {
			if _, err := now.For(b.Codename, arch); err != nil {
				t.Errorf("%s/%s: %v", b.Codename, arch, err)
			}
		}
		if len(builds) != len(now.Binaries) {
			t.Errorf("%s carries %d builds, want the %d the release made", b.Codename, len(builds), len(now.Binaries))
		}
	}
}

// With no build of its own a board names the shared file, so introducing one later is a change to
// that board and to nothing else.
func TestABoardWithNoBuildOfItsOwnNamesTheSharedFile(t *testing.T) {
	_, now := write(t)

	for arch, shared := range now.Binaries {
		got, err := now.For("biscuit", arch)
		if err != nil {
			t.Fatal(err)
		}
		if got != shared {
			t.Errorf("%s: biscuit is offered %+v, want the shared %+v", arch, got, shared)
		}
	}
}

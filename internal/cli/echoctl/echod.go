package echoctl

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ygelfand/echolocal/internal/host/device"
	"github.com/ygelfand/echolocal/internal/host/profile"
	"github.com/ygelfand/echolocal/internal/update"
)

// resolveEchod is the binary to install: a file when one was named, otherwise the build this
// device's board is served by, fetched from a manifest.
//
// Per board because what is compiled in differs — a board with a panel carries the screen and the
// artwork, and a board without it must not.
func resolveEchod(ctx context.Context, out io.Writer, d *device.Device, p profile.Profile, manifest, path string) ([]byte, string, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		return data, path, err
	}

	if manifest == "" {
		manifest = update.Stable.URL()
	}
	m, err := update.At(ctx, manifest)
	if err != nil {
		return nil, "", fmt.Errorf("%w\n\nA local binary can be given with --echod", err)
	}

	arch, err := deviceArch(d)
	if err != nil {
		return nil, "", err
	}

	b, err := m.For(p.Board.Codename, arch)
	if err != nil {
		return nil, "", fmt.Errorf("%w\n\nA local binary can be given with --echod", err)
	}

	title := fmt.Sprintf("Fetching echod %s for %s (%.1f MB)", m.Version, p.Board, float64(b.Size)/(1<<20))
	return fetchImage(ctx, out, title, b.Resolve)
}

// deviceArch is GOARCH for the device, from the ABI it reports.
func deviceArch(d *device.Device) (string, error) {
	abi, err := d.Getprop("ro.product.cpu.abi")
	if err != nil {
		return "", err
	}

	switch {
	case strings.HasPrefix(abi, "arm64"):
		return "arm64", nil
	case strings.HasPrefix(abi, "armeabi"):
		return "arm", nil
	}
	return "", fmt.Errorf("this device reports abi %q, which echod has no build for", abi)
}

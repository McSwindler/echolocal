package echoctl

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ygelfand/echolocal/internal/host/device"
	"github.com/ygelfand/echolocal/internal/host/profile"
)

// profileFor is the install this device needs, or why there is not one.
func profileFor(d *device.Device) (profile.Profile, error) {
	name, err := d.Getprop("ro.product.device")
	if err != nil {
		return profile.Profile{}, fmt.Errorf("asking the device what it is: %w", err)
	}
	return profile.For(name)
}

// resolveBootImage is the image to write: the one named on the command line, or the one published
// for this board, from the cache or fetched.
//
// A fetch is the only part of an install that needs the network, and it happens once per machine
// per board — the cache is named by the hash, so a second device or a second run finds it there.
func resolveBootImage(ctx context.Context, out io.Writer, p profile.Profile, path string) ([]byte, string, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		return data, path, err
	}

	cached, _ := p.Boot.CachePath()
	if _, err := os.Stat(cached); err != nil {
		fmt.Fprintf(out, "Fetching the %s boot image (%.1f MB)\n", p.Board, float64(p.Boot.Size)/(1<<20))
	}

	var last int
	data, from, err := p.Boot.Resolve(ctx, func(f float64) {
		// Whole percent only, and only when it moves: this writes above the progress display that is
		// about to start, and a line per chunk would scroll it away.
		if pct := int(f * 100); pct >= last+10 {
			last = pct
			fmt.Fprintf(out, "  %d%%\n", pct)
		}
	})
	if err != nil {
		return nil, "", fmt.Errorf("%w\n\nA local image can be given with --boot-image", err)
	}
	return data, from, nil
}

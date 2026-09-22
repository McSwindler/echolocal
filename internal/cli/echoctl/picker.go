package echoctl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ygelfand/echolocal/internal/host/device"
)

// ErrCancelled means the user dismissed a prompt.
var ErrCancelled = errors.New("cancelled")

// connect resolves which device a command acts on and opens it. Every command goes through
// this, so device selection behaves the same everywhere.
func connect(ctx context.Context, out io.Writer, serial string) (*device.Device, error) {
	target, err := resolveSerial(ctx, out, serial, device.List, "")
	if err != nil {
		return nil, err
	}
	return device.Connect(target)
}

// attach is connect for the install, which begins by writing the boot image that grants root. It
// cannot demand root up front for the same reason.
//
// A device adb cannot drive is offered a retry rather than refused. An install stopped part way leaves
// one there, and the way back is the device's own buttons; the retry then picks up wherever that lands,
// recovery included.
func attach(ctx context.Context, out io.Writer, serial string) (*device.Device, error) {
	for {
		d, err := attachOnce(ctx, out, serial)
		if !errors.Is(err, device.ErrUnreachable) || !isTerminal() {
			return d, err
		}
		if err := offerRecovery(ctx, out); err != nil {
			return nil, err
		}
	}
}

// noneOfThese is the row for a picker that listed devices but not the one being installed to.
const noneOfThese = "None of these"

func attachOnce(ctx context.Context, out io.Writer, serial string) (*device.Device, error) {
	target, err := resolveSerial(ctx, out, serial, device.ListAny, noneOfThese)
	if err != nil {
		return nil, err
	}
	return device.AttachAny(target)
}

// offerRecovery says how to get a device back and waits for one to turn up.
//
// What is connected when this starts is what was just turned down, so waiting means waiting for a
// serial that is not already there rather than for any device at all.
func offerRecovery(ctx context.Context, out io.Writer) error {
	fmt.Fprintf(out, "\n%s\n", styleTitle.Render("Cannot reach the device"))
	fmt.Fprintf(out, "%s\n", styleDetail.Render(
		"  If the device is up and connected but the adb patches are not installed yet,\n"+
			"  hold volume up while it powers on. On a device with a light, it is in\n"+
			"  recovery once that light is solid white."))

	already := make(map[string]bool)
	for _, d := range listAny() {
		already[d.Serial] = true
	}

	fmt.Fprintf(out, "%s\n", styleDetail.Render("  Waiting for it to appear; ctrl+c to give up."))

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		for _, d := range listAny() {
			if !already[d.Serial] {
				fmt.Fprintf(out, "%s\n", styleDone.Render("✓ "+d.Serial+" appeared"))
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ErrCancelled
		case <-tick.C:
		}
	}
}

// listAny is the connected devices, or none when adb cannot say.
func listAny() []device.Info {
	devices, err := device.ListAny()
	if err != nil {
		return nil
	}
	return devices
}

// resolveSerial decides which device to act on. An explicit serial always wins. On a terminal
// the user picks, even when only one device is connected, so it is clear what is about to be
// written to. Off a terminal there is nobody to ask, so selection is left to device.Connect.
// other, when set, adds a row for none of them being the one, which is reported as a device that
// could not be reached: that is the same answer as nothing being connected, and the caller that
// retries treats it the same way.
func resolveSerial(ctx context.Context, out io.Writer, serial string, list func() ([]device.Info, error), other string) (string, error) {
	if serial != "" {
		return serial, nil
	}
	if !isTerminal() {
		return "", nil
	}

	devices, err := list()
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", device.ErrUnreachable
	}
	return pickDevice(ctx, out, devices, other)
}

func pickDevice(ctx context.Context, out io.Writer, devices []device.Info, other string) (string, error) {
	chosen, err := choose(ctx, out, "Select a device", devices,
		func(d device.Info) string { return fmt.Sprintf("%s  %s", d, d.Serial) }, other)
	if err != nil {
		return "", err
	}
	if chosen.Serial == "" {
		return "", device.ErrUnreachable
	}
	return chosen.Serial, nil
}

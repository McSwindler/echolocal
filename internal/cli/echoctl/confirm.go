package echoctl

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ygelfand/echolocal/internal/host/device"
	"github.com/ygelfand/echolocal/internal/host/installer"
)

// approveFlash asks before the progress display starts, since that owns the terminal.
//
// The answer is typed out rather than a keypress: this is the one thing echoctl does that cannot be
// undone from here.
func approveFlash(ctx context.Context, out io.Writer, d *device.Device, state installer.BootState, image string, assumeYes bool) (bool, error) {
	if assumeYes {
		return true, nil
	}
	if !isTerminal() {
		return false, errors.New("writing the boot partition needs confirmation: run this on a terminal, or pass --yes")
	}

	fmt.Fprintf(out, "\n%s\n", styleTitle.Render("This takes the device over, and undoing it needs a reflash"))
	fmt.Fprintf(out, "  device     %s (%s)\n", d.Serial(), state.Summary)
	if image != "" {
		fmt.Fprintf(out, "  partition  %s\n", state.Partition)
		fmt.Fprintf(out, "  image      %s\n", image)
	}

	return typed(ctx, out, "Type yes to continue", "yes")
}

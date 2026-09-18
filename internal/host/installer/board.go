package installer

import (
	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/device"
)

// BoardOf is what a connected device says it is, and Biscuit when it will not say or says something
// this build has no name for.
//
// Assuming rather than refusing is right for everything that acts on a device echod is already
// installed on — a restart, a key rotation, reading state. Every device in the field is a biscuit,
// and the cost of being wrong here is naming the wrong init service, which fails loudly and changes
// nothing. Installing does not come through here: it resolves a profile and refuses what it does not
// recognise, because that one writes to the boot partition.
func BoardOf(d *device.Device) board.Board {
	device, err := d.Getprop("ro.product.device")
	if err != nil {
		return board.Biscuit
	}
	if b, ok := board.For(device); ok {
		return b
	}
	return board.Biscuit
}

// board is the board this run is acting on, read once and held.
func (r *run) board() board.Board {
	if r.on.Device == "" {
		r.on = BoardOf(r.d)
	}
	return r.on
}

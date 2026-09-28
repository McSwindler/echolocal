// Package defaults is the settings values that are really statements about a board.
//
// config owns the shape of what the device remembers. This owns the numbers a fresh device starts
// from where those came from measuring one — the gain the vendor ran the array at, the noise floor
// of a quiet room as these microphones hear it, how many cores are worth holding online. Read as a
// constant in config, each of those would be a biscuit fact the next board silently inherited.
//
// Only what actually varies lives here. Half the volume range is where a device nobody has turned up
// should sit whatever it is, and a default like that stays in config where it is read.
package defaults

import (
	"sync"

	"github.com/ygelfand/echolocal/internal/board"
)

// Set is what one board starts from.
type Set struct {
	// MicGain is analog gain on the array in dB.
	MicGain int

	// Sensitivity is how far above the room's own noise a sound has to be to count, in dB.
	Sensitivity int

	// RingTrouble and RingMuted are animations by name, empty for none. They are settings about a
	// ring, and a board without one is never asked: the entities are not offered there.
	RingTrouble string
	RingMuted   string

	// MinCores is how many cores are held online that the governor would otherwise park.
	MinCores int
}

// biscuit is every value as it was measured on a 2nd-generation Echo Dot, which is the only board
// any of this has been measured on.
var biscuit = Set{
	// Where the vendor ran it.
	MicGain: 20,

	// Measured on a quiet room: 0.8 dB at the 99th percentile of frames, so this is well clear of the
	// room itself and is really about brief small sounds — a chair, a keyboard.
	Sensitivity: 8,

	// A failure says so, because a request that silently did nothing is the worst of the options.
	RingTrouble: "Alert",

	// A cut microphone does not, because the button has its own LED for exactly this and a ring held
	// lit for as long as someone leaves the device muted is both a light nobody asked for and, on this
	// hardware, an audible one.
	RingMuted: "",

	MinCores: 2,
}

// sets is the boards somebody has measured. A board gets an entry when there is a device to measure
// it on, and not before.
var sets = map[string]Set{
	board.Biscuit.Device: biscuit,
}

// For is what a board starts from, biscuit's where it has no set of its own.
func For(b board.Board) Set {
	if s, ok := sets[b.Device]; ok {
		return s
	}
	return biscuit
}

var (
	mu      sync.RWMutex
	current = biscuit
)

// Use says which board's values to start from. Called once at boot, before anything reads a setting.
func Use(b board.Board) {
	mu.Lock()
	defer mu.Unlock()
	current = For(b)
}

// Current is the set in force, biscuit's if nobody has said otherwise.
func Current() Set {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

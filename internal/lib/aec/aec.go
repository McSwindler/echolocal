// Package aec removes the echo of what a device is playing from what its microphone hears, given a
// reference of the playback sample-aligned with the microphone.
package aec

import "errors"

const (
	// full is int16 full scale.
	full = 32768

	// erleTau is the averaging length of the reported ERLE, in samples.
	erleTau = 8000
)

// ErrLength is returned when the microphone and reference frames are not the same length.
var ErrLength = errors.New("aec: mic and ref must be the same length")

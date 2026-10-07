//go:build !board_checkers && !board_cronos && !board_doppler

package mic

import (
	"github.com/ygelfand/echolocal/internal/lib/alsa"
)

// The capture codec accepts one format only: 16 kHz, S24_3LE, 9 channels.
const (
	Channels      = 9
	CaptureDevice = 24
	Format        = alsa.FormatS24_3LE
	Bits          = 24
)

// Mics is how many of the nine channels are microphones. ch7 and ch8 are the playback loopback.
const Mics = 7

// CenterMic is the middle microphone: no arrival delay relative to the array, and usable with no
// beamformer at all.
const CenterMic = 6

// adcs are the four converters the seven microphones arrive on.
var adcs = []string{"A", "B", "C", "D"}

const beamforming = true

const lowCutHz = 0

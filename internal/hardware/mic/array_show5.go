//go:build board_checkers || board_cronos

package mic

// Device 22 offers 16 kHz, S24_3LE, 4 channels; ch0 and ch1 are the microphones, ch2 and ch3 read silent with nothing playing.
const (
	Channels      = 4
	CaptureDevice = 22
)

const Mics = 2

const CenterMic = 0

var adcs = []string{"A"}

const beamforming = false

// Measured in a silent room beside a Dot: 40-100 Hz sits about 9 dB above it, the rest 3-5 dB.
const lowCutHz = 100

package privacy

import (
	"errors"
	"time"
)

// Writing 1 to enable cuts, writing 0 does nothing, and only the button releases. Measured on
// checkers.
type gating struct {
	dir string
	sw  func() (cut, known bool)
}

func (g gating) state() string { return g.dir + "/state" }
func (g gating) cut() string   { return g.dir + "/enable" }

// On cronos the state file reports a cut about 1.1s after SW_MUTE_DEVICE does; a release lands in both at once.
func (g gating) Get() (bool, error) {
	if g.sw != nil {
		if cut, known := g.sw(); known {
			return cut, nil
		}
	}
	return reads(g.state(), "1")
}

func (gating) HardwareToggles() bool { return true }

func (gating) Lag() time.Duration { return 500 * time.Millisecond }

// Checkers' kernel learns the latch is set only from a write to enable; cronos ignores a write while cut.
func (g gating) Set(muted bool) error {
	if muted {
		return write(g.cut(), "1")
	}
	switch is, err := g.Get(); {
	case err != nil:
		return err
	case !is:
		return nil
	}
	return errors.New("releasing the microphones needs the mute button")
}

func (g gating) Toggle() (bool, error) {
	is, err := g.Get()
	if err != nil {
		return false, err
	}
	return !is, g.Set(!is)
}

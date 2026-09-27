package privacy

import (
	"errors"
	"time"
)

// Writing 1 to enable cuts, writing 0 does nothing, and only the button releases. Measured on
// checkers.
type gating struct{ dir string }

func (g gating) state() string { return g.dir + "/state" }
func (g gating) cut() string   { return g.dir + "/enable" }

func (g gating) Get() (bool, error) { return reads(g.state(), "1") }

func (gating) HardwareToggles() bool { return true }

func (gating) Lag() time.Duration { return 500 * time.Millisecond }

func (g gating) Set(muted bool) error {
	switch is, err := g.Get(); {
	case err != nil:
		return err
	case is == muted:
		return nil
	case muted:
		return write(g.cut(), "1")
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

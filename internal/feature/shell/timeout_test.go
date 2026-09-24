package shell

import (
	"testing"
	"time"
)

// lingering is a view that asks for the settings timeout, the way *Page does.
type lingering struct{ blank }

func (lingering) Timeout() time.Duration { return SettingsTimeout }

// fast goes away quickly, so a countdown can be watched inside a test.
type fast struct{ blank }

func (fast) Timeout() time.Duration { return 50 * time.Millisecond }

// Drawing is not touching. The countdown used to restart on every paint, so a view that repaints
// itself held up the whole stack, including the screens under it that nobody was looking at.
func TestRepaintingDoesNotHoldTheScreenOpen(t *testing.T) {
	s := &Shell{}
	s.Push(fast{blank{"ticker"}})

	for end := time.Now().Add(150 * time.Millisecond); time.Now().Before(end); {
		s.Redraw()
		time.Sleep(10 * time.Millisecond)
	}

	if s.Open() {
		t.Error("a view that kept redrawing never timed out")
	}
}

func TestTimeoutIsTheDockOneByDefault(t *testing.T) {
	s := &Shell{}
	s.stack = []View{blank{"rail"}}

	if got := s.timeout(); got != DockTimeout {
		t.Errorf("a rail times out after %v, want %v", got, DockTimeout)
	}
}

// The rail is under a settings screen, not replaced by it, so a stack holding both has to take the
// longer of the two. Reading the top view alone would answer this one right and the next one wrong.
func TestASettingsScreenOverTheRailTakesTheLongerTimeout(t *testing.T) {
	s := &Shell{}
	s.stack = []View{blank{"rail"}, lingering{blank{"display"}}}

	if got := s.timeout(); got != SettingsTimeout {
		t.Errorf("a settings screen times out after %v, want %v", got, SettingsTimeout)
	}
}

// A notice arriving over a settings screen must not cut the screen's time short.
func TestSomethingOverASettingsScreenDoesNotShortenIt(t *testing.T) {
	s := &Shell{}
	s.stack = []View{lingering{blank{"display"}}, blank{"volume"}}

	if got := s.timeout(); got != SettingsTimeout {
		t.Errorf("a notice over a settings screen times out after %v, want %v", got, SettingsTimeout)
	}
}

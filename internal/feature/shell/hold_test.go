package shell

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// blank is a view that draws nothing, for testing the stack rather than the drawing.
type blank struct{ name string }

func (blank) Draw(ui.Surface, theme.Theme) {}
func (blank) Tap(int, int) bool            { return true }
func (blank) Covers() bool                 { return true }

func (s *Shell) showing() []View {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]View(nil), s.stack...)
}

func TestReleasingTakesAwayTheHeldView(t *testing.T) {
	s := &Shell{}
	held := s.Hold(blank{"player"})

	if got := s.showing(); len(got) != 1 {
		t.Fatalf("holding put up %d views, want 1", len(got))
	}
	if !held.Held() {
		t.Error("a hold that was just taken does not report itself held")
	}

	held.Release()

	if got := s.showing(); len(got) != 0 {
		t.Errorf("releasing left %d views up, want none", len(got))
	}
	if held.Held() {
		t.Error("a released hold still reports itself held")
	}
}

// Keep(false) is what a player does when it stops: the card stays on the screen and goes with the
// idle timeout, rather than vanishing the moment the music does.
func TestNoLongerHoldingLeavesTheViewUp(t *testing.T) {
	s := &Shell{}
	held := s.Hold(blank{"player"})

	held.Keep(false)

	if got := s.showing(); len(got) != 1 {
		t.Fatalf("no longer holding left %d views up, want the view still there", len(got))
	}
	if held.Held() {
		t.Error("it still reports itself held")
	}

	// And the timeout takes it, where it would have kept a held one.
	s.idled()
	if got := s.showing(); len(got) != 0 {
		t.Errorf("the idle timeout kept %d views that nothing is holding", len(got))
	}
}

// Playing again inside the timeout has to take the hold back rather than put up a second copy: the
// player screen is one instance, and Push appends whatever it is given.
func TestHoldingAViewAlreadyUpDoesNotStackIt(t *testing.T) {
	s := &Shell{}
	player := blank{"player"}

	first := s.Hold(player)
	first.Keep(false)

	again := s.Hold(player)

	if got := s.showing(); len(got) != 1 {
		t.Fatalf("holding it again put up %d views, want 1", len(got))
	}
	if !again.Held() {
		t.Error("taking the hold back does not report itself held")
	}

	s.idled()
	if got := s.showing(); len(got) != 1 {
		t.Errorf("the idle timeout took a held view away, leaving %d", len(got))
	}
}

// The point of a hold: it takes away its own view and leaves whatever somebody opened over it.
func TestReleasingLeavesAScreenOpenedOverIt(t *testing.T) {
	s := &Shell{}
	held := s.Hold(blank{"player"})

	settings := blank{"settings"}
	s.Push(settings)

	held.Release()

	got := s.showing()
	if len(got) != 1 {
		t.Fatalf("%d views are up, want 1", len(got))
	}
	if got[0] != View(settings) {
		t.Errorf("the view left up is %v, want the one that was opened over the hold", got[0])
	}
}

// Releasing twice is what happens when something stops, is asked to stop again, and neither call
// knows about the other.
func TestReleasingTwiceTakesNothingElse(t *testing.T) {
	s := &Shell{}
	held := s.Hold(blank{"player"})
	held.Release()

	settings := blank{"settings"}
	s.Push(settings)
	held.Release()

	if got := s.showing(); len(got) != 1 {
		t.Fatalf("%d views are up, want the one opened after releasing", len(got))
	}
}

// A nil hold is what a caller has before anything has been held.
func TestANilHoldIsSafe(t *testing.T) {
	var held *Hold

	if held.Held() {
		t.Error("a nil hold reports itself held")
	}
	held.Release()
}

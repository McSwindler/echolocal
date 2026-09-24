package shell

import "testing"

func TestPopUncoversWhatWasUnderIt(t *testing.T) {
	s := &Shell{}
	under := blank{"settings"}
	s.Push(under)
	s.Push(blank{"theme"})

	s.Pop()

	got := s.showing()
	if len(got) != 1 {
		t.Fatalf("%d views are up, want 1", len(got))
	}
	if got[0] != View(under) {
		t.Errorf("the view left up is %v, want the one that was under", got[0])
	}
}

// Back from the bottom of the stack is the dashboard. Every page's header goes back, so the one
// that got there first has to close rather than pop nothing and sit there.
func TestPopAtTheRootCloses(t *testing.T) {
	s := &Shell{}
	s.Push(blank{"settings"})

	s.Pop()

	if s.Open() {
		t.Error("popping the last view left the shell open")
	}
}

func TestPopWithNothingUpIsSafe(t *testing.T) {
	s := &Shell{}
	s.Pop()

	if s.Open() {
		t.Error("popping an empty stack put something up")
	}
}

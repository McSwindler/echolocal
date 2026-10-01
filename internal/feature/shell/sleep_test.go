package shell

import "testing"

type sleeping struct{ blank }

func (sleeping) Asleep() bool { return true }

type waking struct {
	blank
	wakes bool
}

func (w *waking) Wakes() bool { return w.wakes }

type overlay struct{ blank }

func (overlay) Covers() bool { return false }

func TestAViewThatDoesNotCoverComesUpOverSleepAndLeavesItUnder(t *testing.T) {
	s := &Shell{}
	idle := sleeping{blank{"idle"}}
	s.Hold(idle)
	card := overlay{blank{"volume"}}
	h := s.Hold(card)
	if !h.Held() {
		t.Fatal("a overlay card was refused over sleep")
	}
	h.Release()
	if got := s.showing(); len(got) != 1 || got[0] != View(idle) {
		t.Fatalf("after the card went, the stack is %v, want the sleeper alone", got)
	}
}

func TestOnlyAWakerComesUpOverSleep(t *testing.T) {
	s := &Shell{}
	idle := sleeping{blank{"idle"}}
	s.Hold(idle)

	if h := s.Hold(blank{"volume"}); h != nil || len(s.showing()) != 1 {
		t.Fatalf("a plain view came up over sleep: %v", s.showing())
	}
	s.Push(blank{"card"})
	if len(s.showing()) != 1 {
		t.Fatalf("a pushed view came up over sleep: %v", s.showing())
	}

	video := &waking{blank: blank{"video"}}
	if h := s.Hold(video); h != nil {
		t.Fatal("a waker without a picture came up over sleep")
	}
	video.wakes = true
	if h := s.Hold(video); !h.Held() {
		t.Fatal("a waker with a picture was refused")
	}

	if h := s.Hold(blank{"volume"}); !h.Held() {
		t.Fatal("once awake, a plain view was refused")
	}
}

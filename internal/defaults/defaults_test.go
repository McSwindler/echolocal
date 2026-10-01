package defaults

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/board"
)

// The values moved here out of config were all measured on a biscuit, and moving them must not have
// changed any of them: a device that has never been configured has to come up the way it did before.
func TestBiscuitIsWhatItAlwaysWas(t *testing.T) {
	got := For(board.Biscuit)

	want := Set{
		MicGain:     20,
		Sensitivity: 8,
		RingTrouble: "Alert",
		RingMuted:   "",
		MinCores:    2,

		VisualizerLift: 10,
	}
	if got != want {
		t.Errorf("biscuit's defaults are %+v, want %+v", got, want)
	}
}

func TestTheShowsStartTheVisualsUnlifted(t *testing.T) {
	for _, b := range []board.Board{board.Checkers, board.Cronos} {
		if got := For(b).VisualizerLift; got != 0 {
			t.Errorf("%s lifts the visuals %d dB, want 0", b.Codename, got)
		}
	}
}

func TestAnUnmeasuredBoardBorrowsBiscuits(t *testing.T) {
	if got := For(board.Crown); got != For(board.Biscuit) {
		t.Errorf("crown starts from %+v, want biscuit's %+v", got, For(board.Biscuit))
	}
}

// Nobody has to call Use for the values to be sane, because plenty of things that read a setting are
// not the agent: the offline tools, and every test in the tree.
func TestCurrentIsBiscuitUntilToldOtherwise(t *testing.T) {
	if got := Current(); got != For(board.Biscuit) {
		t.Errorf("Current() = %+v before Use, want biscuit's %+v", got, For(board.Biscuit))
	}
}

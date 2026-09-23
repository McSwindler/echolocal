package touch

import (
	"testing"
	"time"

	"github.com/ygelfand/echolocal/internal/lib/input"
)

func abs(code uint16, v int32) input.Event {
	return input.Event{Type: input.EvAbs, Code: code, Value: v}
}

var syn = input.Event{Type: input.EvSyn, Code: synReport}

// feed runs a batch and returns what the closing SYN_REPORT produced.
func feed(d *decoder, events ...input.Event) []Contact {
	var out []Contact
	now := time.Now()
	for _, e := range events {
		out = append(out, d.event(e, now)...)
	}
	return out
}

func TestAFingerDownMovesAndLifts(t *testing.T) {
	var d decoder

	got := feed(&d,
		abs(absMTSlot, 0), abs(absMTTrackingID, 7),
		abs(absMTPositionX, 100), abs(absMTPositionY, 200), syn)
	if len(got) != 1 || got[0].Phase != Down {
		t.Fatalf("first report %+v, want one Down", got)
	}
	if got[0].X != 100 || got[0].Y != 200 || got[0].ID != 7 {
		t.Errorf("Down at (%d,%d) id=%d, want (100,200) id=7", got[0].X, got[0].Y, got[0].ID)
	}

	// Only X changed; the kernel says nothing about Y and the finger keeps the position it had.
	got = feed(&d, abs(absMTPositionX, 140), syn)
	if len(got) != 1 || got[0].Phase != Move {
		t.Fatalf("second report %+v, want one Move", got)
	}
	if got[0].X != 140 || got[0].Y != 200 {
		t.Errorf("Move at (%d,%d), want (140,200)", got[0].X, got[0].Y)
	}

	got = feed(&d, abs(absMTTrackingID, released), syn)
	if len(got) != 1 || got[0].Phase != Up {
		t.Fatalf("third report %+v, want one Up", got)
	}
	if got[0].ID != 7 {
		t.Errorf("Up carried id %d, want the 7 it went down with", got[0].ID)
	}
}

// A report with nothing changed says nothing, or every SYN would look like a finger moving.
func TestAnIdleReportSaysNothing(t *testing.T) {
	var d decoder

	feed(&d, abs(absMTSlot, 0), abs(absMTTrackingID, 1), abs(absMTPositionX, 10), syn)
	if got := feed(&d, syn); len(got) != 0 {
		t.Errorf("an idle report produced %+v", got)
	}
}

// A slot released without ever having been reported down must not produce an Up, or a caller sees
// a finger lift that never landed.
func TestALiftWithoutADownIsSilent(t *testing.T) {
	var d decoder

	if got := feed(&d, abs(absMTSlot, 0), abs(absMTTrackingID, released), syn); len(got) != 0 {
		t.Errorf("a lift with no down produced %+v", got)
	}
}

func TestTwoFingersAreTrackedApart(t *testing.T) {
	var d decoder

	got := feed(&d,
		abs(absMTSlot, 0), abs(absMTTrackingID, 1), abs(absMTPositionX, 10), abs(absMTPositionY, 11),
		abs(absMTSlot, 1), abs(absMTTrackingID, 2), abs(absMTPositionX, 90), abs(absMTPositionY, 91),
		syn)
	if len(got) != 2 {
		t.Fatalf("got %d contacts, want 2", len(got))
	}

	byID := map[int]Contact{}
	for _, c := range got {
		byID[c.ID] = c
	}
	if c := byID[1]; c.X != 10 || c.Y != 11 || c.Slot != 0 {
		t.Errorf("id 1 is %+v, want slot 0 at (10,11)", c)
	}
	if c := byID[2]; c.X != 90 || c.Y != 91 || c.Slot != 1 {
		t.Errorf("id 2 is %+v, want slot 1 at (90,91)", c)
	}
}

// One finger lifting must not disturb the other, which is what the slot bookkeeping is for.
func TestOneFingerLiftingLeavesTheOther(t *testing.T) {
	var d decoder

	feed(&d,
		abs(absMTSlot, 0), abs(absMTTrackingID, 1), abs(absMTPositionX, 10),
		abs(absMTSlot, 1), abs(absMTTrackingID, 2), abs(absMTPositionX, 90),
		syn)

	got := feed(&d, abs(absMTSlot, 0), abs(absMTTrackingID, released), syn)
	if len(got) != 1 || got[0].Phase != Up || got[0].ID != 1 {
		t.Fatalf("got %+v, want only id 1 lifting", got)
	}

	got = feed(&d, abs(absMTSlot, 1), abs(absMTPositionX, 95), syn)
	if len(got) != 1 || got[0].Phase != Move || got[0].ID != 2 {
		t.Fatalf("got %+v, want id 2 still moving", got)
	}
}

// Positions arrive in the panel's own coordinates and are reported in the ones drawing uses.
func TestContactsArePlacedInViewedCoordinates(t *testing.T) {
	d := decoder{place: func(x, y int) (int, int) { return y, x }}

	got := feed(&d, abs(absMTSlot, 0), abs(absMTTrackingID, 1),
		abs(absMTPositionX, 3), abs(absMTPositionY, 400), syn)
	if len(got) != 1 {
		t.Fatalf("got %d contacts, want 1", len(got))
	}
	if got[0].X != 400 || got[0].Y != 3 {
		t.Errorf("reported (%d,%d), want the placed (400,3)", got[0].X, got[0].Y)
	}
}

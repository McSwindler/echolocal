package timesync

import (
	"testing"
	"time"
)

func TestAnAnswerThatAgreesLeavesTheClockAndSaysSo(t *testing.T) {
	ts := &Time{}
	var got []Sync
	ts.Synced.Listen(func(s Sync) { got = append(got, s) })

	ts.fromHome(uint32(time.Now().Unix()))

	if len(got) != 1 {
		t.Fatalf("%d syncs, want 1", len(got))
	}
	if got[0].Stepped() {
		t.Errorf("an answer within a second stepped the clock by %v", got[0].After.Sub(got[0].Before))
	}
	if !ts.Valid() {
		t.Error("the time is not valid after Home Assistant agreed with it")
	}
}

func TestOnlyAMovedClockIsAStep(t *testing.T) {
	at := time.Now()
	if (Sync{Before: at, After: at}).Stepped() {
		t.Error("the same time before and after is a step")
	}
	if !(Sync{Before: at, After: at.Add(5 * time.Minute)}).Stepped() {
		t.Error("five minutes is not a step")
	}
}

func TestNoAnswerIsIgnored(t *testing.T) {
	ts := &Time{}
	called := false
	ts.Synced.Listen(func(Sync) { called = true })
	ts.fromHome(0)
	if called {
		t.Error("an empty answer was reported")
	}
}

package indicate

import (
	"testing"
	"time"
)

// A board with nothing to show on still answers, so a feature says what happened without first
// asking whether anyone is listening.
func TestAClaimIsAlwaysUsable(t *testing.T) {
	s := nowhere{}

	if s.Present() {
		t.Error("nowhere reports a surface")
	}

	c := s.Claim(PriorityMute)
	if c == nil {
		t.Fatal("no claim")
	}

	// None of these may panic, which is the whole point of answering with something.
	c.Play("Pulse", Color{R: 0xFF})
	c.PlayReversed("Pulse", Color{})
	c.React("Pulse", Color{}, Room{})
	c.ShowFor("Alert", Color{}, time.Second)
	c.Clear()
	c.Release()

	s.HoldOnStop(Color{})
}

// Get always answers, even before any surface has registered.
func TestGetAlwaysAnswers(t *testing.T) {
	if got := Get(); got == nil {
		t.Fatal("Get returned nothing")
	}
	// Twice, since it is resolved once and held.
	if Get() != Get() {
		t.Error("Get answered with a different surface the second time")
	}
}

// The priorities are an order, and the ones that must outrank each other do. A surface resolves
// competing claims by comparing these, so the order is the contract.
func TestPrioritiesAreOrdered(t *testing.T) {
	for _, tc := range []struct {
		lower, higher Priority
		why           string
	}{
		{PriorityBase, PriorityRoom, "the room reaction sits over the resting colour"},
		{PriorityRoom, PriorityMute, "a cut microphone outranks reacting to a room it cannot hear"},
		{PriorityBusy, PriorityTurn, "answering outranks getting ready to"},
		{PriorityTurn, PriorityAlarm, "a ringing timer outranks the turn told to stop it"},
		{PriorityTurn, PriorityTrouble, "ending a failed turn must not take the failure with it"},
		{PriorityTrouble, PriorityBoot, "nothing may draw over the start-up indication"},
	} {
		if tc.lower >= tc.higher {
			t.Errorf("%v is not under %v: %s", tc.lower, tc.higher, tc.why)
		}
	}
}

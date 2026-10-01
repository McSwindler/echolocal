package buttons

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/lib/input"
)

func press(c *Controller, down map[uint16]*held, code uint16, at, until uint64) {
	c.key(input.Event{Type: input.EvKey, Code: code, Value: 1, Sec: at / 1e6, Usec: at % 1e6}, down)
	c.key(input.Event{Type: input.EvKey, Code: code, Value: 0, Sec: until / 1e6, Usec: until % 1e6}, down)
}

func TestAnInstantMutePressIsPower(t *testing.T) {
	c := &Controller{}
	var got []Event
	c.Events.Listen(func(e Event) { got = append(got, e) })
	down := map[uint16]*held{}

	press(c, down, 113, 2_559_640_067, 2_559_830_078)
	press(c, down, 113, 2_560_950_087, 2_560_950_127)

	want := []Event{{Name: Mute, Kind: Tap}, {Name: Power, Kind: Tap}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d is %v, want %v", i, got[i], want[i])
		}
	}
}

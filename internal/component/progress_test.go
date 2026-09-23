package component

import "testing"

type coming struct {
	name string
	p    Progress
}

func (c *coming) Name() string      { return c.name }
func (c *coming) Startup() Progress { return c.p }

// quiet is a component with nothing to say about coming up, which is most of them.
type quiet struct{ name string }

func (q *quiet) Name() string { return q.name }

func TestProgressOnlyCollectsWhatSpeaks(t *testing.T) {
	r := New()
	r.Add(Hardware, func() Component { return &coming{name: "wifi"} })
	r.Add(Hardware, func() Component { return &quiet{name: "buttons"} })

	got := r.Progress()
	if len(got) != 1 || got[0].Name != "wifi" {
		t.Errorf("Progress() = %+v, want only wifi", got)
	}
}

// The registry fills the name in, so a component says what it is doing without repeating what it is.
func TestProgressCarriesTheName(t *testing.T) {
	r := New()
	r.Add(Hardware, func() Component { return &coming{name: "screen", p: Progress{Doing: "taking the panel"}} })

	got := r.Progress()
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	if got[0].Name != "screen" || got[0].Doing != "taking the panel" {
		t.Errorf("got %+v, want screen/taking the panel", got[0])
	}
}

func TestReadyWaitsForWhatHolds(t *testing.T) {
	r := New()
	r.Add(Hardware, func() Component { return &coming{name: "wifi", p: Progress{Doing: "associating"}} })

	if r.Ready() {
		t.Error("ready while something is still coming up")
	}
}

func TestReadyOnceEverythingIsDone(t *testing.T) {
	r := New()
	r.Add(Hardware, func() Component { return &coming{name: "wifi", p: Progress{Done: true}} })

	if !r.Ready() {
		t.Error("not ready with everything done")
	}
}

// A component that will not be coming up must not hold the device on the boot screen for ever.
func TestAFailureStopsTheWait(t *testing.T) {
	r := New()
	r.Add(Hardware, func() Component {
		return &coming{name: "mic", p: Progress{Failed: true, Doing: "no capture device"}}
	})

	if !r.Ready() {
		t.Error("a failed component is still being waited for")
	}
}

func TestReadyWithNothingToWaitFor(t *testing.T) {
	r := New()
	r.Add(Hardware, func() Component { return &quiet{name: "buttons"} })

	if !r.Ready() {
		t.Error("not ready with nothing that reports progress")
	}
}

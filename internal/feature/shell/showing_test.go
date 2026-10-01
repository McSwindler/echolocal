package shell

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// page is a view that either covers the screen or hangs over it.
type page struct {
	name   string
	covers bool
	urgent bool
}

func (p *page) Draw(ui.Surface, theme.Theme) {}
func (p *page) Tap(int, int) bool            { return true }
func (p *page) Covers() bool                 { return p.covers }
func (p *page) Urgent() bool                 { return p.urgent }

func names(views []View) []string {
	out := make([]string, 0, len(views))
	for _, v := range views {
		out = append(out, v.(*page).name)
	}
	return out
}

func same(t *testing.T, got []View, want ...string) {
	t.Helper()

	have := names(got)
	if len(have) != len(want) {
		t.Fatalf("drawing %v, want %v", have, want)
	}
	for i := range want {
		if have[i] != want[i] {
			t.Errorf("drawing %v, want %v", have, want)
			return
		}
	}
}

// The one that shipped. The volume card hangs over the screen rather than covering it, so with only
// the top view drawn it replaced whatever was under it and the dashboard showed through — the
// settings disappeared while somebody was reading them.
func TestAnOverlayDoesNotReplaceTheScreenUnderIt(t *testing.T) {
	settings := &page{name: "settings", covers: true}
	volume := &page{name: "volume"}

	same(t, showing([]View{settings, volume}), "settings", "volume")
}

// Only back to the highest one that covers: anything under that is hidden and painting it is work
// for nothing.
func TestNothingUnderTheTopCoveringViewIsDrawn(t *testing.T) {
	first := &page{name: "first", covers: true}
	second := &page{name: "second", covers: true}
	over := &page{name: "over"}

	same(t, showing([]View{first, second, over}), "second", "over")
}

// A rail on its own lets the dashboard through, which is the whole point of it.
func TestOverlaysWithNothingUnderThemAreAllDrawn(t *testing.T) {
	rail := &page{name: "rail"}
	volume := &page{name: "volume"}

	same(t, showing([]View{rail, volume}), "rail", "volume")
}

func TestOneCoveringViewIsJustItself(t *testing.T) {
	same(t, showing([]View{&page{name: "settings", covers: true}}), "settings")
}

// An alarm going off is still an alarm with a volume notice over it, and the claim has to stay
// where the alarm put it or both drop under whatever else is showing.
func TestUrgencyIsTakenFromAnywhereInTheStack(t *testing.T) {
	ringing := &page{name: "ringing", covers: true, urgent: true}
	volume := &page{name: "volume"}

	if got := rank([]View{ringing, volume}); got != display.PriorityAlert {
		t.Errorf("a notice over an alarm is drawn at %v, want the alarm's priority", got)
	}
}

func TestAnOrdinaryStackIsNotUrgent(t *testing.T) {
	settings := &page{name: "settings", covers: true}
	volume := &page{name: "volume"}

	if got := rank([]View{settings, volume}); got != display.PriorityUI {
		t.Errorf("an ordinary screen is drawn at %v, want %v", got, display.PriorityUI)
	}
}

// A view that is not urgent and does not say so at all are the same thing, and most views do not
// implement it.
func TestAViewThatSaysNothingIsNotUrgent(t *testing.T) {
	if urgent(&quiet{}) {
		t.Error("a view with no opinion counted as urgent")
	}
	if urgent(&page{urgent: false}) {
		t.Error("a view that said no counted as urgent")
	}
}

type quiet struct{}

func (quiet) Draw(ui.Surface, theme.Theme) {}
func (quiet) Tap(int, int) bool            { return true }
func (quiet) Covers() bool                 { return true }

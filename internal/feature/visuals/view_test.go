package visuals

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

func TestWithoutTheHelperTheViewStaysDark(t *testing.T) {
	r, v := newRig()
	w := v.View()
	img := ui.NewImage(120, 80, theme.Color{R: 9})
	w.Draw(img, theme.Default())
	w.mu.Lock()
	running, broken := w.running, w.broken
	w.mu.Unlock()
	if running || !broken || r.tapped() != nil {
		t.Fatalf("running %v, broken %v, tapped %v", running, broken, r.tapped() != nil)
	}
	if img.At(60, 40) == (theme.Color{R: 9}) {
		t.Fatal("the view did not clear its hole")
	}
}

func TestATapPutsTheViewAway(t *testing.T) {
	_, v := newRig()
	if v.View().Tap(10, 10) {
		t.Fatal("a tap was kept")
	}
}

package videoplayer

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

type clearing struct {
	*ui.Image
	cleared []ui.Rect
}

func (c *clearing) ClearRect(r ui.Rect) { c.cleared = append(c.cleared, r) }

func TestAPlayingPictureShowsThroughThePage(t *testing.T) {
	p := NewPage(playing())
	p.SetPicture(true)
	s := &clearing{Image: ui.NewImage(1920, 1200, theme.Color{R: 1})}
	p.Draw(s, theme.All[0])
	if len(s.cleared) != 1 || s.cleared[0] != (ui.Rect{W: 1920, H: 1200}) {
		t.Errorf("cleared %v, want the whole page", s.cleared)
	}
}

func TestWithoutAPictureThePageIsOpaque(t *testing.T) {
	p := NewPage(playing())
	p.SetNote("refused")
	s := &clearing{Image: ui.NewImage(1920, 1200, theme.Color{R: 1})}
	p.Draw(s, theme.All[0])
	if len(s.cleared) != 0 {
		t.Errorf("cleared %v with no picture to show", s.cleared)
	}
}

func TestThePageKeepsEveryTapAndItsButtonsAct(t *testing.T) {
	f := playing()
	p := NewPage(f)
	p.SetPicture(true)
	if !p.Tap(600, 100) {
		t.Fatal("a tap on the video closed the shell")
	}
	p.Draw(ui.NewImage(1920, 1200, theme.Color{}), theme.All[0])
	p.mu.Lock()
	hits := p.hits
	p.mu.Unlock()
	var pause hit
	for _, h := range hits {
		if h.at.W < 400 && h.at.W > pause.at.W {
			pause = h
		}
	}
	x, y := pause.at.Center()
	if !p.Tap(x, y) || f.paused != 1 {
		t.Errorf("tapping the biggest button paused %d times", f.paused)
	}
	if !p.Tap(600, 100) {
		t.Error("a tap away from the controls closed the shell")
	}
}

func TestFollowPointsTheControlsAtTheNextTrack(t *testing.T) {
	first, next := playing(), playing()
	p := NewPage(first)
	p.Follow(next)
	p.SetPicture(true)
	p.Tap(600, 100)
	p.Draw(ui.NewImage(1920, 1200, theme.Color{}), theme.All[0])
	p.mu.Lock()
	hits := p.hits
	p.mu.Unlock()
	var pause hit
	for _, h := range hits {
		if h.at.W < 400 && h.at.W > pause.at.W {
			pause = h
		}
	}
	x, y := pause.at.Center()
	p.Tap(x, y)
	if first.paused != 0 || next.paused != 1 {
		t.Errorf("paused the first track %d times and the next %d", first.paused, next.paused)
	}
}

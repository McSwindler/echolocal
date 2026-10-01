package idle

import (
	"fmt"
	"sync"

	"github.com/ygelfand/echolocal/internal/feature/clock/face"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

type span struct {
	x, y int
	px   []byte
}

type Stamps struct {
	mu   sync.Mutex
	made map[string][]span
}

func (st *Stamps) get(key string, make func() []span) []span {
	st.mu.Lock()
	defer st.mu.Unlock()
	if s, ok := st.made[key]; ok {
		return s
	}
	if st.made == nil || len(st.made) > 4 {
		st.made = map[string][]span{}
	}
	s := make()
	st.made[key] = s
	return s
}

func stampKey(reading string, box ui.Rect, p theme.Theme, f face.Face) string {
	return fmt.Sprintf("%s|%v|%v|%v|%v|%v|%T", reading, box, p.Background, p.Text, p.Muted, p.Accent, f)
}

func makeStamp(f face.Face, box ui.Rect, r face.Reading, palette theme.Theme) []span {
	w, h := box.W, box.H
	onBlack := ui.NewImage(w, h, theme.Color{})
	onWhite := ui.NewImage(w, h, theme.Color{R: 255, G: 255, B: 255})
	local := ui.Rect{W: w, H: h}
	f.Draw(onBlack, local, r, palette)
	f.Draw(onWhite, local, r, palette)

	var out []span
	for y := range h {
		var run *span
		for x := range w {
			b, wh := onBlack.At(x, y), onWhite.At(x, y)
			gap := max(int(wh.R)-int(b.R), int(wh.G)-int(b.G), int(wh.B)-int(b.B), 0)
			a := byte(255 - min(gap, 255))
			if a == 0 {
				run = nil
				continue
			}
			if run == nil {
				out = append(out, span{x: box.X + x, y: box.Y + y})
				run = &out[len(out)-1]
			}
			run.px = append(run.px, b.R, b.G, b.B, a)
		}
	}
	return out
}

func paintStamp(s ui.Surface, spans []span) {
	for _, sp := range spans {
		for i := 0; i < len(sp.px); i += 4 {
			ui.Over(s, sp.x+i/4, sp.y, theme.Color{R: sp.px[i], G: sp.px[i+1], B: sp.px[i+2]}, sp.px[i+3])
		}
	}
}

package screen

import "testing"

// checkers' panel, portrait.
const (
	fbW = 480
	fbH = 960
)

func TestSizeSwapsOnlySideways(t *testing.T) {
	for _, tc := range []struct {
		o    Orientation
		w, h int
	}{
		{Rotate0, fbW, fbH},
		{Rotate90, fbH, fbW},
		{Rotate180, fbW, fbH},
		{Rotate270, fbH, fbW},
	} {
		if w, h := tc.o.Size(fbW, fbH); w != tc.w || h != tc.h {
			t.Errorf("%s: %dx%d, want %dx%d", tc.o, w, h, tc.w, tc.h)
		}
	}
}

// Every viewed pixel has to land inside the framebuffer and no two may land on the same one, or a
// rotated draw writes outside the mapping or leaves holes.
func TestPanelCoversTheFramebufferExactly(t *testing.T) {
	// Small, so every pixel can be walked.
	const w, h = 4, 6

	for _, o := range []Orientation{Rotate0, Rotate90, Rotate180, Rotate270} {
		vw, vh := o.Size(w, h)
		seen := make(map[[2]int]bool, w*h)

		for y := range vh {
			for x := range vw {
				px, py := o.Panel(x, y, w, h)

				if px < 0 || px >= w || py < 0 || py >= h {
					t.Fatalf("%s: viewed (%d,%d) maps to (%d,%d), outside %dx%d", o, x, y, px, py, w, h)
				}
				if seen[[2]int{px, py}] {
					t.Fatalf("%s: two viewed pixels both map to (%d,%d)", o, px, py)
				}
				seen[[2]int{px, py}] = true
			}
		}

		if len(seen) != w*h {
			t.Errorf("%s: covered %d of %d framebuffer pixels", o, len(seen), w*h)
		}
	}
}

// The viewed origin is the top-left of what a person sees, whichever way the panel is mounted.
func TestTheViewedOriginIsACorner(t *testing.T) {
	corners := map[Orientation][2]int{
		Rotate0:   {0, 0},
		Rotate90:  {0, fbH - 1},
		Rotate180: {fbW - 1, fbH - 1},
		Rotate270: {fbW - 1, 0},
	}
	for o, want := range corners {
		px, py := o.Panel(0, 0, fbW, fbH)
		if px != want[0] || py != want[1] {
			t.Errorf("%s: viewed origin is panel (%d,%d), want (%d,%d)", o, px, py, want[0], want[1])
		}
	}
}

// A touch has to land where the pixel was drawn, so Viewed has to undo Panel exactly.
func TestViewedUndoesPanel(t *testing.T) {
	const w, h = 4, 6

	for _, o := range []Orientation{Rotate0, Rotate90, Rotate180, Rotate270} {
		vw, vh := o.Size(w, h)

		for y := range vh {
			for x := range vw {
				px, py := o.Panel(x, y, w, h)
				gx, gy := o.Viewed(px, py, w, h)

				if gx != x || gy != y {
					t.Fatalf("%s: (%d,%d) -> panel (%d,%d) -> (%d,%d)", o, x, y, px, py, gx, gy)
				}
			}
		}
	}
}

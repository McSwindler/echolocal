package mic

import (
	"math"
	"testing"
)

func gainAt(t *testing.T, hz float64) float64 {
	t.Helper()
	f := newLowCut(100)
	in := make([]int16, Rate)
	for i := range in {
		in[i] = int16(10000 * math.Sin(2*math.Pi*hz*float64(i)/Rate))
	}
	out := append([]int16(nil), in...)
	f.apply(out)

	var ein, eout float64
	for i := Rate / 2; i < Rate; i++ {
		ein += float64(in[i]) * float64(in[i])
		eout += float64(out[i]) * float64(out[i])
	}
	return 10 * math.Log10(eout/ein)
}

func TestLowCutTakesOutHumAndKeepsSpeech(t *testing.T) {
	for _, c := range []struct {
		hz       float64
		low, top float64
	}{
		{50, -15, -10},
		{100, -3.5, -2.5},
		{300, -0.6, 0.1},
		{1000, -0.1, 0.1},
		{4000, -0.1, 0.1},
	} {
		if g := gainAt(t, c.hz); g < c.low || g > c.top {
			t.Errorf("%.0f Hz comes through at %.1f dB, want %.1f to %.1f", c.hz, g, c.low, c.top)
		}
	}
}

func TestNoLowCutLeavesTheFrameAlone(t *testing.T) {
	frame := []int16{1, -2, 3}
	newLowCut(0).apply(frame)
	if frame[0] != 1 || frame[1] != -2 || frame[2] != 3 {
		t.Errorf("an absent filter changed the frame to %v", frame)
	}
}

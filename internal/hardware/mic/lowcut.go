package mic

import "math"

// lowCut is a second-order Butterworth high-pass, run in place over the mixed frame.
type lowCut struct {
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
}

func newLowCut(hz float64) *lowCut {
	if hz <= 0 {
		return nil
	}
	w := 2 * math.Pi * hz / Rate
	cos, alpha := math.Cos(w), math.Sin(w)/math.Sqrt2
	a0 := 1 + alpha
	return &lowCut{
		b0: (1 + cos) / 2 / a0,
		b1: -(1 + cos) / a0,
		b2: (1 + cos) / 2 / a0,
		a1: -2 * cos / a0,
		a2: (1 - alpha) / a0,
	}
}

func (f *lowCut) apply(frame []int16) {
	if f == nil {
		return
	}
	for i, v := range frame {
		x := float64(v)
		y := f.b0*x + f.b1*f.x1 + f.b2*f.x2 - f.a1*f.y1 - f.a2*f.y2
		f.x2, f.x1 = f.x1, x
		f.y2, f.y1 = f.y1, y
		frame[i] = int16(min(max(math.Round(y), math.MinInt16), math.MaxInt16))
	}
}

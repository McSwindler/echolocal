package mic

import (
	"math"
	"testing"
)

// A wake engine is routinely several frames behind: measured on the Dot at worst 120 ms a frame, six
// frames, against a channel eight deep.
const behind = 8

type reusedBuffer struct {
	buf []int16
	nth int16
}

func (m *reusedBuffer) Mix(mics [][]int16) []int16 {
	if cap(m.buf) < len(mics[0]) {
		m.buf = make([]int16, len(mics[0]))
	}
	m.buf = m.buf[:len(mics[0])]

	m.nth++
	for i := range m.buf {
		m.buf[i] = m.nth
	}
	return m.buf
}

func quietSource() *Source {
	s := New()
	s.cancelling = false
	s.leveling.Store(false)
	s.suppress = nil
	s.lowCut = nil
	return s
}

func TestListenerBehindKeepsEveryFrame(t *testing.T) {
	s := quietSource()
	s.mixer = &reusedBuffer{}

	frames, stop := s.Listen("behind")
	defer stop()

	for i := range behind {
		s.broadcast(captured(i, false))
	}

	for i := range behind {
		frame := <-frames
		for _, v := range frame {
			if v != int16(i+1) {
				t.Fatalf("frame %d carries frame %d: a frame a listener still held was overwritten", i+1, v)
			}
		}
	}
}

func TestCancelledFramesAreNotOneBuffer(t *testing.T) {
	s := quietSource()
	s.cancelling = true

	frames, stop := s.Listen("behind")
	defer stop()

	for i := range behind {
		s.broadcast(captured(i, true))
	}
	if !s.Cancelling() {
		t.Fatal("the canceller did not engage")
	}

	seen := map[*int16]int{}
	for i := range behind {
		frame := <-frames
		if first, ok := seen[&frame[0]]; ok {
			t.Fatalf("frames %d and %d are the same buffer", first+1, i+1)
		}
		seen[&frame[0]] = i
	}
}

func captured(nth int, playing bool) []byte {
	const frameBytes = Channels * Bits / 8

	raw := make([]byte, FrameSamples*frameBytes)
	for f := range FrameSamples {
		off := f * frameBytes
		for c := range Mics {
			put24(raw[off+c*3:], int32(nth+1)<<8)
		}
		if !playing {
			continue
		}
		v := int32(8000 * math.Sin(2*math.Pi*440*float64(nth*FrameSamples+f)/Rate))
		for c := range Refs {
			put24(raw[off+(Mics+c)*3:], v<<8)
		}
	}
	return raw
}

func put24(b []byte, v int32) { b[0], b[1], b[2] = byte(v), byte(v>>8), byte(v>>16) }

package aec

import (
	"math"
	"testing"
)

func TestSuppressionSurvivesExactSilence(t *testing.T) {
	const frame, rate = 64, 16000
	for _, withEcho := range []bool{false, true} {
		var echo *MDF
		if withEcho {
			m, err := NewMDF(frame, 1024, rate)
			if err != nil {
				t.Fatal(err)
			}
			echo = m
		}
		p, err := NewPreprocessor(frame, rate, echo)
		if err != nil {
			t.Fatal(err)
		}

		block := make([]int16, frame)
		for range 3000 {
			clear(block)
			if echo != nil {
				out, err := echo.Process(block, block)
				if err != nil {
					t.Fatal(err)
				}
				copy(block, out)
			}
			if err := p.Run(block); err != nil {
				t.Fatal(err)
			}
		}

		speech := noiseAt(frame*400, 6000, 7)
		var loudest float64
		for at := 0; at+frame <= len(speech); at += frame {
			copy(block, speech[at:at+frame])
			if err := p.Run(block); err != nil {
				t.Fatal(err)
			}
			for _, v := range block {
				loudest = math.Max(loudest, math.Abs(float64(v)))
			}
		}
		if loudest == 0 {
			t.Errorf("echo %t: sound after a run of exact zeros came out silent", withEcho)
		}
		for i, v := range p.gain {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				t.Fatalf("echo %t: gain %d is %v after exact zeros", withEcho, i, v)
			}
		}
	}
}

package speaker

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/ygelfand/echolocal/internal/config"
)

// levels is how loud each kind of sound is against the level the hardware carries.
//
// In software because more than one kind sounds at once: a beep over a reply, a reply ducking
// music. The hardware has one gain, and it belongs to the device rather than to any of them.
type levels struct {
	once sync.Once
	at   map[config.Stream]*atomic.Uint32
}

func (l *levels) build() {
	l.once.Do(func() {
		l.at = map[config.Stream]*atomic.Uint32{}
		for _, s := range config.Streams() {
			v := &atomic.Uint32{}
			v.Store(math.Float32bits(1))
			l.at[s] = v
		}
	})
}

// SetLevel is how loud one kind of sound plays, as a percentage.
func (p *Player) SetLevel(s config.Stream, level int) {
	p.levels.build()

	at, ok := p.levels.at[s]
	if !ok {
		return
	}
	at.Store(math.Float32bits(float32(max(0, min(level, 100))) / 100))
}

// Level is the multiplier one kind of sound plays at.
func (p *Player) Level(s config.Stream) float32 {
	p.levels.build()

	at, ok := p.levels.at[s]
	if !ok {
		return 1
	}
	return math.Float32frombits(at.Load())
}

// quieter is samples at a level, and the samples themselves at full.
func quieter(samples []int16, level float32) []int16 {
	if level >= 1 || len(samples) == 0 {
		return samples
	}

	out := make([]int16, len(samples))
	for i, s := range samples {
		out[i] = int16(float32(s) * level)
	}
	return out
}

// attenuate turns samples down in place.
func attenuate(samples []int16, level float32) {
	if level >= 1 {
		return
	}
	for i, s := range samples {
		samples[i] = int16(float32(s) * level)
	}
}

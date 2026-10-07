package mic

import (
	"log/slog"
	"math"
	"sync/atomic"

	"github.com/ygelfand/echolocal/internal/lib/aec"
)

// cancelTaps is the echo tail the filter reaches, 64 ms. Measured on a Dot: 13.7% of a core.
const cancelTaps = 1024

// speexFrame is the speex block, which divides FrameSamples.
const speexFrame = 64

// LANovo's.
const echoSuppressActive = -40

func newSuppressor(c *canceller) *aec.Preprocessor {
	var echo *aec.MDF
	if c != nil {
		echo = c.filter
	}
	p, err := aec.NewPreprocessor(speexFrame, Rate, echo)
	if err != nil {
		slog.Error("noise suppression unavailable", "err", err)
		return nil
	}
	p.EchoSuppressActive = echoSuppressActive
	return p
}

func (s *Source) suppressBlock(block []int16) {
	if err := s.suppress.Run(block); err != nil {
		slog.Error("noise suppression failed", "err", err)
	}
}

// refQuiet is the mean square per sample, at int16 scale, below which the loopback counts as silence.
// About -60 dBFS. Below it there is no echo to remove, so the filter is skipped entirely and the frame
// costs nothing — which is what keeps this free on an idle device.
const refQuiet = 1e-6 * 32768 * 32768

// refHold is how long the filter keeps running after the loopback goes quiet, in samples.
//
// Speech is full of gaps, and without this the gate flaps between every word. That is not only untidy:
// the room is still ringing with the echo of the word that just played, and the measured tail here runs
// past 100 ms, so disengaging on the gap stops cancelling exactly as that tail arrives. 250 ms covers
// it with room to spare.
const refHold = Rate / 4

// canceller subtracts the playback loopback from one fixed microphone.
//
// It runs on the center microphone rather than the mix, and replaces the mix while it runs. Two reasons,
// and simplicity is the lesser of them: the filter has to learn one acoustic path, so the microphone it
// reads cannot move — and the beamformer steers at the loudest sound, which during playback is our own
// speaker. Steering into the echo is the opposite of useful when the echo is what is being removed.
type canceller struct {
	filter *aec.MDF

	// erle is published for diagnostics, as thousandths of a dB so it fits an integer. best is the most
	// it reached during the current run: ERLE is averaged over the last half second, so reading it as
	// playback ends catches the reference fading out and says the filter did worse than it did.
	erle atomic.Int64
	best atomic.Int64

	// active says whether the last frame had playback in it, which is when any of this happened.
	active atomic.Bool

	refE, micE float64
	blocks     int

	out []int16
}

func newCanceller() *canceller {
	f, err := aec.NewMDF(speexFrame, cancelTaps, Rate)
	if err != nil {
		slog.Error("echo cancellation unavailable", "err", err)
		return nil
	}
	return &canceller{filter: f}
}

// idle is the frame having no playback in it. The filter keeps what it learned: the room has not
// changed just because the reply ended, so the next one starts converged rather than from nothing.
func (c *canceller) idle() {
	if c.active.Swap(false) {
		// ERLE cannot exceed how far the echo stood above the rest of the microphone.
		slog.Info("echo cancellation idle",
			"best_db", float64(c.best.Load())/1000, "last_db", float64(c.erle.Load())/1000,
			"ref_dbfs", meanDBFS(c.refE, c.blocks), "mic_dbfs", meanDBFS(c.micE, c.blocks),
			"seconds", math.Round(float64(c.blocks)*float64(FrameSamples)/Rate*10)/10)
		c.erle.Store(0)
		c.best.Store(0)
		c.refE, c.micE, c.blocks = 0, 0, 0
	}
}

// apply returns the center microphone with the echo removed, for a frame the caller has already found
// playback in.
func (c *canceller) apply(ref []int16, mics [][]int16, after func(block []int16)) []int16 {
	if !c.active.Swap(true) {
		slog.Info("echo cancellation running", "engine", "speex", "taps", cancelTaps)
	}
	mic := mics[CenterMic]
	c.refE += power(ref)
	c.micE += power(mic)
	c.blocks++

	n := min(len(mic), len(ref))
	if cap(c.out) < n {
		c.out = make([]int16, n)
	}
	out := c.out[:n]
	for k := 0; k+speexFrame <= n; k += speexFrame {
		got, err := c.filter.Process(mic[k:k+speexFrame], ref[k:k+speexFrame])
		if err != nil {
			slog.Error("echo cancellation failed", "err", err)
			return nil
		}
		copy(out[k:k+speexFrame], got)
		if after != nil {
			after(out[k : k+speexFrame])
		}
	}
	erle := int64(c.filter.ERLE() * 1000)
	c.erle.Store(erle)
	if erle > c.best.Load() {
		c.best.Store(erle)
	}
	return out
}

// sounding decodes the loopback and reports whether the device is making a sound, counting the tail
// after it goes quiet. Everything the microphones learn from the room is held still while it is true:
// the echo is not the room, and neither the filter nor the leveler has any way to tell from the audio.
func (s *Source) sounding(raw []byte, n int) bool {
	if cap(s.ref) < n {
		s.ref = make([]int16, n)
	}
	s.ref = s.ref[:n]
	referenceInto(raw, s.ref)

	switch {
	case playing(s.ref):
		s.hold = refHold
	case s.hold > 0:
		s.hold -= n
	default:
		return false
	}
	return true
}

// playing reports whether the loopback carries anything.
func playing(ref []int16) bool {
	return len(ref) > 0 && power(ref) > refQuiet
}

func meanDBFS(energy float64, blocks int) float64 {
	if blocks == 0 || energy <= 0 {
		return -120
	}
	return math.Round(10*math.Log10(energy/float64(blocks)/(32768*32768))*10) / 10
}

// power is the mean square per sample, at int16 scale.
func power(s []int16) float64 {
	var sum float64
	for _, v := range s {
		sum += float64(v) * float64(v)
	}
	if len(s) == 0 {
		return 0
	}
	return sum / float64(len(s))
}

// referenceInto decodes ch7, the left half of the playback loopback, into dst. ch8 is left alone: on
// music the two measure within 12-17 dB of each other, which caps cancellation far above anything the
// filter reaches, so a stereo reference would buy nothing.
func referenceInto(raw []byte, dst []int16) {
	bytesPerSample := Bits / 8
	frameBytes := Channels * bytesPerSample

	frames := min(len(raw)/frameBytes, len(dst))
	for f := range frames {
		o := f*frameBytes + Mics*bytesPerSample
		dst[f] = DecodeSample(raw[o : o+bytesPerSample])
	}
	for f := frames; f < len(dst); f++ {
		dst[f] = 0
	}
}

// Cancelling reports whether the canceller is currently running, which it does only while the loopback
// is carrying audio.
func (s *Source) Cancelling() bool {
	return s.cancel != nil && s.cancel.active.Load()
}

// ERLE is how much echo the canceller is removing, in dB, or zero when it is not running.
func (s *Source) ERLE() float64 {
	if s.cancel == nil {
		return 0
	}
	return float64(s.cancel.erle.Load()) / 1000
}

// SetCancelling turns echo cancellation on or off. Off is the same signal path the device had before it
// existed, which is what makes it worth having as a switch: it is the comparison.
func (s *Source) SetCancelling(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelling = on
}

// SetAdapting stops or resumes the canceller learning, while it goes on cancelling with what it has.
//
// Turn it off while somebody is being listened to. The filter cannot tell a voice it was never given a
// reference for from an echo it predicted badly, so it treats the voice as its own error and fits itself
// to it — at exactly the moment cancelling matters. What it already learned stays correct: the room did
// not change because somebody spoke.
func (s *Source) SetAdapting(on bool) { s.adapt.Store(on) }

// Freeze holds the canceller still whatever the conversation asks, for measuring it.
func (s *Source) Freeze(on bool) { s.frozen.Store(on) }

// Adapting reports whether the canceller is learning.
func (s *Source) Adapting() bool { return s.adapt.Load() && !s.frozen.Load() }

// Frozen reports whether the control socket is holding it still.
func (s *Source) Frozen() bool { return s.frozen.Load() }

// ResetCancel forgets what the canceller learned, on the next frame.
func (s *Source) ResetCancel() { s.reset.Store(true) }

// Package mic owns the microphone array: one capture stream, held for the life of the process,
// with 16 kHz mono frames fanned out to whoever is listening.
package mic

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ygelfand/echolocal/internal/android/prop"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/lib/aec"
	"github.com/ygelfand/echolocal/internal/lib/alsa"
	"github.com/ygelfand/echolocal/internal/lib/audio"
	"github.com/ygelfand/echolocal/internal/service"
)

const (
	Rate = 16000

	Card = 0

	period  = FrameSamples
	periods = 8
)

// Refs is the loopback pair that follows the microphones, left then right.
const Refs = Channels - Mics

// FrameSamples is the frame size handed to listeners, 20 ms at 16 kHz.
const FrameSamples = Rate / 50

// MediaService is the init service that holds the capture device on a fresh boot.
const MediaService = "media"

const (
	acquireRetry    = 250 * time.Millisecond
	acquireAttempts = 8
)

// Source is the capture stream. Listeners receive mono frames; a listener that cannot keep up
// misses frames rather than stalling the reader.
type Source struct {
	// The hardware, taken by Start and let go by Close. Nil in between: the handle outlives the
	// device so that a restart can take it again without every listener being rebuilt.
	devMu sync.Mutex
	pcm   *alsa.Capture

	mu        sync.Mutex
	listeners map[int]*listener
	raw       map[int]chan []byte
	next      int

	// dropped counts frames a listener was too slow to take. It matters more than it looks: the
	// wake model is streaming, so a missing frame corrupts its state rather than just losing audio.
	dropped atomic.Uint64

	history history

	// mixer is read by the reader and replaced from Home Assistant, both under mu.
	mixer  Mixer
	mixing config.Mixing

	// cancel belongs to the reader alone. cancelling is the switch, which anything may set, and is read
	// under mu with the mixer.
	cancel     *canceller
	cancelling bool

	// adapt is what the conversation wants, frozen what the control socket wants, and reset a request
	// to forget the room. Anything may set them; only the reader hands them to the filter.
	adapt, frozen, reset atomic.Bool

	echoes map[int]chan Echo

	// ref is the decoded loopback and hold the samples still counted as sounding after it goes quiet.
	// Reader-only, and read whatever the cancelling switch says.
	ref  []int16
	hold int

	// leveler and wasLeveling belong to the reader alone; leveling is the switch, which anything may
	// set.
	leveler     *leveler
	wasLeveling bool
	leveling    atomic.Bool

	lowCut *lowCut

	suppress *aec.Preprocessor

	// Which way the loudest sound is, as a beam, or -1 until something asks. finder belongs to the
	// reader; wantFacing is how anything else asks it to look.
	finder     *Beamformer
	facing     atomic.Int32
	wantFacing atomic.Bool
}

// SetMixing chooses how the microphones are combined. It takes effect on the next frame, and reports
// what it settled on, which differs from the request only when this build cannot do it.
func (s *Source) SetMixing(m config.Mixing) config.Mixing {
	mixer, settled := NewMixer(m)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.mixer, s.mixing = mixer, settled
	return settled
}

// Mixing is the combination in use.
func (s *Source) Mixing() config.Mixing {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mixing
}

// New makes the array without taking the hardware, so listeners can subscribe before there is
// anything to hear. Listen, Recent and the mixing setting work throughout; frames start at Start.
func New() *Source {
	mixer, mixing := NewMixer(config.Get().Microphone.Mixing)
	s := &Source{
		listeners:  map[int]*listener{},
		raw:        map[int]chan []byte{},
		echoes:     map[int]chan Echo{},
		mixer:      mixer,
		mixing:     mixing,
		leveler:    newLeveler(),
		finder:     NewBeamformer(),
		cancel:     newCanceller(),
		cancelling: config.Get().Microphone.Cancel,
		lowCut:     newLowCut(lowCutHz),
	}
	s.finder.hold = 1
	s.facing.Store(-1)
	s.adapt.Store(true)
	s.leveling.Store(config.Get().Microphone.Leveling)
	s.suppress = newSuppressor(s.cancel)
	return s
}

var (
	once   sync.Once
	shared *Source
)

func init() {
	// After the speaker: both are held for the life of the process, and the playback path is the one
	// the vendor's own services fight over.
	component.Register(component.Hardware, Get, component.Order(7),
		component.Supervise(service.Restart(time.Second, 30*time.Second)))
}

// Get is the array. There is one, and everything that wants frames subscribes to it.
func Get() *Source {
	once.Do(func() { shared = New() })
	return shared
}

func (s *Source) Name() string { return "capture" }

// Start takes the capture device, off Android if it got there first, the same way the speaker does.
func (s *Source) Start(context.Context) error {
	err := s.open()
	if err == nil || !errors.Is(err, alsa.ErrBusy) {
		return err
	}

	slog.Warn("capture device busy, stopping "+MediaService+" to take it", "err", err)
	if err := prop.Stop(MediaService); err != nil {
		return err
	}
	defer func() {
		if err := prop.Start(MediaService); err != nil {
			slog.Error("restarting "+MediaService+" failed", "err", err)
		}
	}()

	for range acquireAttempts {
		time.Sleep(acquireRetry)

		err = s.open()
		if err == nil {
			slog.Info("capture device acquired")
			return nil
		}
		if !errors.Is(err, alsa.ErrBusy) {
			return err
		}
	}
	return err
}

// Acquire is New and Start together, for a tool that wants the array for the length of one command.
func Acquire() (*Source, error) {
	s := New()
	if err := s.Start(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Source) open() error {
	pcm, err := alsa.Open(Card, CaptureDevice, alsa.Config{
		Channels:   Channels,
		Rate:       Rate,
		Format:     Format,
		Bits:       Bits,
		PeriodSize: period,
		Periods:    periods,
	})
	if err != nil {
		return fmt.Errorf("mic: opening capture: %w", err)
	}

	s.devMu.Lock()
	s.pcm = pcm
	s.devMu.Unlock()

	routeInputs()
	applyGain(config.Get().Microphone.Gain)
	return nil
}

// device is the hardware, or nil when it is not held.
func (s *Source) device() *alsa.Capture {
	s.devMu.Lock()
	defer s.devMu.Unlock()
	return s.pcm
}

type listener struct {
	name      string
	ch        chan []int16
	dropped   uint64
	told      uint64
	unleveled bool
}

// Listen returns a channel of mono frames and a function that stops the subscription. The name is
// what a dropped frame is reported against.
func (s *Source) Listen(name string) (<-chan []int16, func()) { return s.listen(name, false) }

// ListenUnleveled is Listen before leveling, for something that shows how loud the room is.
func (s *Source) ListenUnleveled(name string) (<-chan []int16, func()) { return s.listen(name, true) }

func (s *Source) listen(name string, unleveled bool) (<-chan []int16, func()) {
	l := &listener{name: name, ch: make(chan []int16, 8), unleveled: unleveled}

	s.mu.Lock()
	id := s.next
	s.next++
	s.listeners[id] = l
	s.mu.Unlock()

	return l.ch, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if was, ok := s.listeners[id]; ok {
			delete(s.listeners, id)
			close(was.ch)
			// A turn lasts seconds, so anything it lost would otherwise go unsaid.
			if lost := was.dropped - was.told; lost > 0 {
				slog.Warn("listener behind", "who", was.name, "frames", lost, "total", was.dropped)
			}
		}
	}
}

// ListenRaw returns a channel of interleaved frames, all nine channels as the hardware gives them.
// For characterising the array; the mono path is what detection and the pipeline use.
func (s *Source) ListenRaw() (<-chan []byte, func()) {
	ch := make(chan []byte, 8)

	s.mu.Lock()
	id := s.next
	s.next++
	s.raw[id] = ch
	s.mu.Unlock()

	return ch, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if c, ok := s.raw[id]; ok {
			delete(s.raw, id)
			close(c)
		}
	}
}

// Decode splits an interleaved frame into one slice per microphone, narrowed to 16 bits.
func Decode(raw []byte) [][]int16 { return decode(raw, 0, Mics) }

// Reference is the playback loopback, left and right. It is what was sent to the DAC rather than
// anything the microphones heard: bit exact, and decimated by the same hardware that decimates them,
// so it arrives already aligned with the microphones at their rate.
func Reference(raw []byte) [][]int16 { return decode(raw, Mics, Refs) }

func decode(raw []byte, first, n int) [][]int16 {
	bytesPerSample := Bits / 8
	frameBytes := Channels * bytesPerSample

	frames := len(raw) / frameBytes
	out := make([][]int16, n)
	for c := range out {
		out[c] = make([]int16, frames)
	}
	for f := range frames {
		off := f * frameBytes
		for c := range out {
			at := off + (first+c)*bytesPerSample
			out[c][f] = DecodeSample(raw[at : at+bytesPerSample])
		}
	}
	return out
}

// Run reads until ctx is cancelled. It reads whether or not anyone is listening, because a stream
// left unread overruns and the hardware ring is only 160 ms deep.
func (s *Source) Run(ctx context.Context) error {
	pcm := s.device()
	if pcm == nil {
		return errors.New("mic: the capture device is not held")
	}
	raw := make([]byte, FrameSamples*Channels*Bits/8)

	var overruns uint64
	report := time.NewTicker(dropReport)
	defer report.Stop()

	// An ALSA read can return short of a period, and the next read carries the rest.
	filled := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		select {
		case <-report.C:
			s.reportDrops()
		default:
		}

		n, err := pcm.Read(raw[filled:])
		if err != nil {
			if errors.Is(err, alsa.ErrOverrun) {
				if overruns++; overruns == 1 || overruns%100 == 0 {
					slog.Warn("capture overrun", "times", overruns)
				}
				filled = 0
				continue
			}
			return err
		}
		if filled += n; filled < len(raw) {
			continue
		}
		s.broadcast(raw)
		filled = 0
	}
}

// dropReport is how often a listener losing frames is said out loud. Nothing is logged while nothing
// is being lost.
const dropReport = 30 * time.Second

func (s *Source) reportDrops() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, l := range s.listeners {
		since := l.dropped - l.told
		if since == 0 {
			continue
		}
		l.told = l.dropped
		slog.Warn("listener behind", "who", l.name, "frames", since,
			"per_second", float64(since)/dropReport.Seconds(), "total", l.dropped)
	}
}

// broadcast hands the frame to every listener, dropping it for any that is behind. The mono mix is
// only computed when something wants it.
func (s *Source) broadcast(raw []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Everything downstream reads the same single channel, whichever way it was made: wake detection
	// and what Home Assistant transcribes should never disagree about what was heard.
	//
	// The recent history is kept whether or not anyone is listening: a wake word is only recognised
	// once it has been said, so by the time a turn starts, the words after it are already past.
	mics := Decode(raw)
	frame := s.mixer.Mix(mics)

	// While something is playing, the echo cancelled center microphone replaces the mix. It has to be
	// one fixed microphone: the filter learns a single acoustic path, and the beamformer would steer at
	// the loudest thing in the room, which during playback is the speaker being cancelled.
	sounding := false
	if Refs > 0 {
		sounding = s.sounding(raw, len(mics[CenterMic]))
	}
	s.leveler.atPlayback(sounding)

	quiet := s.suppress != nil
	suppressed := false

	if s.cancelling && s.cancel != nil {
		if s.reset.Swap(false) {
			s.cancel.filter.Reset()
		}
		s.cancel.filter.SetAdapting(s.Adapting())

		switch {
		case sounding:
			var after func([]int16)
			if quiet {
				s.suppress.SetEcho(s.cancel.filter)
				after = s.suppressBlock
			}
			if cancelled := s.cancel.apply(s.ref, mics, after); cancelled != nil {
				frame, suppressed = cancelled, quiet
			}
		default:
			s.cancel.idle()
		}
	}

	// The canceller and the mixers hand back a buffer they overwrite next frame.
	frame = append([]int16(nil), frame...)

	if quiet && !suppressed {
		s.suppress.SetEcho(nil)
		s.suppressBlock(frame)
	}
	s.lowCut.apply(frame)

	s.findFacing(mics)

	for _, l := range s.listeners {
		if l.unleveled {
			s.offer(l, append([]int16(nil), frame...))
		}
	}

	// Turning leveling off throws away what it learned, so a room it has adapted badly to is
	// recovered by switching it off and on rather than by restarting anything.
	on := s.leveling.Load()
	switch {
	case on:
		s.leveler.apply(frame)
	default:
		// Measured even with the gain switched off, because how loud the room is has watchers of its
		// own: the ring can be set to react to it, and that should not depend on a setting about what
		// Home Assistant hears.
		if s.wasLeveling {
			s.leveler.forget()
		}
		if len(frame) > 0 {
			s.leveler.observe(frame)
		}
	}
	s.wasLeveling = on
	s.remember(frame)

	for _, l := range s.listeners {
		if !l.unleveled {
			s.offer(l, frame)
		}
	}

	if len(s.echoes) > 0 {
		e := Echo{Mic: mics[CenterMic], Ref: append([]int16(nil), s.ref...), Out: frame, Sounding: sounding}
		for _, ch := range s.echoes {
			select {
			case ch <- e:
			default:
			}
		}
	}

	if len(s.raw) == 0 {
		return
	}

	// The reader reuses its buffer, so raw listeners get their own copy.
	interleaved := make([]byte, len(raw))
	copy(interleaved, raw)
	for _, ch := range s.raw {
		select {
		case ch <- interleaved:
		default:
		}
	}
}

func (s *Source) offer(l *listener, frame []int16) {
	select {
	case l.ch <- frame:
	default:
		l.dropped++
		s.dropped.Add(1)
	}
}

// Dropped is how many frames a listener has missed.
func (s *Source) Dropped() uint64 { return s.dropped.Load() }

// Close lets the device go. The Source stays usable and its listeners stay subscribed: Start can take
// the hardware again, which is how a restart works.
func (s *Source) Close() error {
	s.devMu.Lock()
	pcm := s.pcm
	s.pcm = nil
	s.devMu.Unlock()

	if pcm == nil {
		return nil
	}
	return pcm.Close()
}

// Mono takes the center microphone alone, narrowed from 24 bits to 16. The beamformed mix is what
// listeners get; this is for tools that need one microphone as it comes off the hardware.
func Mono(raw []byte) []int16 {
	bytesPerSample := Bits / 8
	frameBytes := Channels * bytesPerSample

	out := make([]int16, len(raw)/frameBytes)
	for i := range out {
		o := i*frameBytes + CenterMic*bytesPerSample
		out[i] = DecodeSample(raw[o : o+bytesPerSample])
	}
	return out
}

func DecodeSample(raw []byte) int16 {
	switch Format {
	case alsa.FormatS16_LE:
		return int16(audio.DecodeS16LE(raw[:2]))

	case alsa.FormatS24_3LE:
		// Convert signed 24-bit PCM to signed 16-bit PCM.
		return int16(audio.DecodeS24LE3(raw[:3]) >> 8)

	default:
		panic("unknown PCM sample format")
	}
}

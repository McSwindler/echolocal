package media

import (
	"fmt"
	"log/slog"
	"time"
)

// Now is what is playing, for the entity and for the screen.
type Now struct {
	Playing bool
	Paused  bool

	// Hold keeps the card up through a gap, a cast session between tracks.
	Hold bool

	// Empty where the source does not know, which is anything played from a url.
	Title  string
	Artist string
	Album  string

	// Art is the album art as it arrived, nil where there is none, and ArtID changes with it.
	// Bytes rather than a picture: this package is on every board and the decoder is not.
	Art   []byte
	ArtID uint64

	// Mark is the app's own icon, where the source is an app, as bytes like Art.
	Mark   []byte
	MarkID uint64

	// Elapsed is how far into the track playback has reached, and Length how long it runs for. A
	// zero Length is a source that does not say.
	Elapsed time.Duration
	Length  time.Duration

	// LiveWithin is how far behind live a live stream may be and still read as live, zero for one
	// that is not live.
	LiveWithin time.Duration

	// Queue is what plays next, where the source knows.
	Queue []Track

	// Can is the transport to offer.
	Can Controls
}

// Track is one thing in a queue.
type Track struct {
	Title  string
	Artist string
	Length time.Duration

	Art   []byte
	ArtID uint64

	// Play jumps to it, nil where the source cannot.
	Play func()
}

// Seeker is a source that can move within what it is playing.
type Seeker interface {
	Seek(to time.Duration)
	CanSeek() bool
}

// Opener is a source with a screen of its own, which it shows instead of the player's card.
type Opener interface {
	Open() bool
}

// Controls is which transport a source answers to.
type Controls uint8

const (
	CanPause Controls = 1 << iota
	CanStop
	CanNext
	CanPrevious
)

// Has reports whether every control in want is offered.
func (c Controls) Has(want Controls) bool { return c&want == want }

// Kind is where the audio came from.
type Kind uint8

const (
	FromQueue Kind = iota
	FromBluetooth
	FromGroup
	FromCast
)

// Details is a source that says what it is playing and answers the transport for it.
type Details interface {
	Now() Now
	Next()
	Previous()

	// Kind is which of them this is, for the mark on the card.
	Kind() Kind

	// Label is what to call it: the group's own name. Empty where it has none.
	Label() string
}

// Now is what the device is playing, whoever started it.
func (p *Player) Now() Now {
	if d, ok := p.source().(Details); ok {
		return d.Now()
	}

	playing, paused := p.Playing()
	return Now{Playing: playing, Paused: paused, Can: CanStop}
}

// From is the kind holding the card, which is the queue when nothing else has it.
func (p *Player) From() Kind {
	if d, ok := p.source().(Details); ok {
		return d.Kind()
	}
	return FromQueue
}

// Named is what to call whoever holds the card, and empty where it has no name.
func (p *Player) Named() string {
	if d, ok := p.source().(Details); ok {
		return d.Label()
	}
	return ""
}

// Next and Previous reach whoever is playing, and do nothing where it cannot skip.
func (p *Player) Next() {
	if d, ok := p.source().(Details); ok {
		d.Next()
	}
}

func (p *Player) Previous() {
	if d, ok := p.source().(Details); ok {
		d.Previous()
	}
}

// Began hands the card to a source that has started playing. One of a different kind stops whoever
// held it before.
func (p *Player) Began(s Source) {
	var replaced Source
	p.claimMu.Lock()
	if p.live == nil || kindOf(p.live) != kindOf(s) {
		if held := p.source(); held != nil && kindOf(held) != kindOf(s) {
			replaced = held
		}
	}
	p.live = s
	p.claimMu.Unlock()

	p.External(s)
	p.Begun.Emit(s)
	if replaced != nil {
		slog.Info("the player was replaced", "was", fmt.Sprintf("%T", replaced), "by", fmt.Sprintf("%T", s))
		replaced.Stop()
	}
}

// Ended says a source of this kind has stopped, so the next one of its kind counts as new.
func (p *Player) Ended(s Source) {
	p.claimMu.Lock()
	if p.live != nil && kindOf(p.live) == kindOf(s) {
		p.live = nil
	}
	p.claimMu.Unlock()
}

// Release gives the card back, if s still holds it.
func (p *Player) Release(s Source) {
	cur := p.external.Load()
	if cur == nil || *cur != s {
		return
	}
	p.external.Store(nil)
	slog.Info("the player card was given up", "by", fmt.Sprintf("%T", s))
	p.refresh()
}

// Holder is whoever holds the card, nil when the local queue does.
func (p *Player) Holder() Source { return p.source() }

// Sourced names whoever holds the card, for a log, and whether anything does.
func (p *Player) Sourced() (source string, external bool) {
	if s := p.source(); s != nil {
		return fmt.Sprintf("%T", s), true
	}
	return "the local queue", false
}

func kindOf(s Source) Kind {
	if d, ok := s.(Details); ok {
		return d.Kind()
	}
	return FromQueue
}

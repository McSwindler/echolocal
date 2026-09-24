package media

import "time"

// Now is what is playing, for the entity and for the screen.
type Now struct {
	Playing bool
	Paused  bool

	// Empty where the source does not know, which is anything played from a url.
	Title  string
	Artist string
	Album  string

	// Art is the album art as it arrived, nil where there is none, and ArtID changes with it.
	// Bytes rather than a picture: this package is on every board and the decoder is not.
	Art   []byte
	ArtID uint64

	// Elapsed is how far into the track playback has reached, and Length how long it runs for. A
	// zero Length is a source that does not say.
	Elapsed time.Duration
	Length  time.Duration

	// Can is the transport to offer.
	Can Controls
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

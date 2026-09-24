package sendspin

import (
	"hash/maphash"

	"github.com/ygelfand/echolocal/internal/feature/media"
)

// Next and Previous implement media.Details. The server says what it accepts when it joins, and a
// command it did not offer is dropped there.
func (p *Player) Next()     { p.tell("next") }
func (p *Player) Previous() { p.tell("previous") }

func (p *Player) Kind() media.Kind { return media.FromGroup }

// Label is the group's name, which nothing reports yet.
func (p *Player) Label() string { return "" }

// Now is what the group is playing.
func (p *Player) Now() media.Now {
	playing, paused := p.Playing()
	t := p.track()

	p.mu.Lock()
	art := p.artwork
	p.mu.Unlock()

	return media.Now{
		Playing: playing || paused,
		Paused:  paused,
		Title:   t.Title,
		Artist:  t.Artist,
		Album:   t.Album,
		Art:     art,
		ArtID:   artID(art),
		Elapsed: t.at(playing),
		Length:  t.Length,
		Can:     media.CanPause | media.CanStop | media.CanNext | media.CanPrevious,
	}
}

var artSeed = maphash.MakeSeed()

// artID names the picture, so whatever draws it can tell one from the next without keeping a copy.
func artID(art []byte) uint64 {
	if len(art) == 0 {
		return 0
	}
	return maphash.Bytes(artSeed, art)
}

package sendspin

import (
	"time"

	"github.com/Sendspin/sendspin-go/pkg/protocol"
)

// track is what the group is playing, as the metadata role reports it.
type track struct {
	Title  string
	Artist string
	Album  string

	// Elapsed is where the track had reached when the server last said, and At when that was.
	// Progress arrives every few seconds, so a bar that did not run on from it would step.
	Elapsed time.Duration
	Length  time.Duration
	At      time.Time
}

// merge applies one metadata update. The wire is tristate and all three cases mean something
// different: a key that is absent leaves what was there, a key that is null clears it, and a key with
// a value sets it. Treating absent as empty would blank the title on every progress update.
func (t track) merge(m *protocol.MetadataState) track {
	t.Title = field(m, "title", m.Title, t.Title)
	t.Artist = field(m, "artist", m.Artist, t.Artist)
	t.Album = field(m, "album", m.Album, t.Album)

	if m.HasField("progress") && m.Progress != nil {
		t.Elapsed = time.Duration(m.Progress.TrackProgress) * time.Millisecond
		t.Length = time.Duration(m.Progress.TrackDuration) * time.Millisecond
		t.At = time.Now()
	}
	return t
}

func (t track) empty() bool { return t.Title == "" && t.Artist == "" && t.Album == "" }

// at is where the track has reached now, run on from the last report.
func (t track) at(playing bool) time.Duration {
	if t.At.IsZero() || !playing {
		return t.Elapsed
	}

	on := t.Elapsed + time.Since(t.At)
	if t.Length > 0 && on > t.Length {
		return t.Length
	}
	return on
}

func field(m *protocol.MetadataState, key string, now *string, was string) string {
	if !m.HasField(key) {
		return was
	}
	if now == nil {
		return ""
	}
	return *now
}

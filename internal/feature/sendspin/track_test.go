package sendspin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Sendspin/sendspin-go/pkg/protocol"
)

// Decoded rather than built: absent and null are the same nil pointer in Go, and only the decoder
// records which of the two arrived.
func decode(t *testing.T, raw string) *protocol.MetadataState {
	t.Helper()

	var m protocol.MetadataState
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	return &m
}

func TestTrackMerge(t *testing.T) {
	was := track{Title: "Teardrop", Artist: "Massive Attack", Album: "Mezzanine"}

	for _, c := range []struct {
		name string
		raw  string
		want track
	}{
		{
			name: "an absent key leaves what was there",
			raw:  `{"year":1998}`,
			want: was,
		},
		{
			name: "a null clears",
			raw:  `{"artist":null}`,
			want: track{Title: "Teardrop", Album: "Mezzanine"},
		},
		{
			name: "a value sets",
			raw:  `{"title":"Angel"}`,
			want: track{Title: "Angel", Artist: "Massive Attack", Album: "Mezzanine"},
		},
		{
			name: "a whole track at once",
			raw:  `{"title":"Rez","artist":"Underworld","album":"Second Toughest"}`,
			want: track{Title: "Rez", Artist: "Underworld", Album: "Second Toughest"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := was.merge(decode(t, c.raw)); got != c.want {
				t.Errorf("merge(%s) = %+v, want %+v", c.raw, got, c.want)
			}
		})
	}
}

func TestTrackEmpty(t *testing.T) {
	if !(track{}).empty() {
		t.Error("the zero track is not empty")
	}
	if (track{Title: "Rez"}).empty() {
		t.Error("a track with a title is empty")
	}
}

func TestTrackProgress(t *testing.T) {
	got := track{Title: "Rez"}.merge(decode(t, `{"progress":{"track_progress":1000,"track_duration":5000}}`))

	if got.Elapsed != time.Second || got.Length != 5*time.Second {
		t.Errorf("merge progress = %v of %v, want 1s of 5s", got.Elapsed, got.Length)
	}
	if got.At.IsZero() {
		t.Error("progress arrived with no time on it")
	}
	if got.Title != "Rez" {
		t.Errorf("progress cleared the title, got %q", got.Title)
	}
}

// A bar drawn from the last report alone steps every few seconds, and one that runs on past the
// end of the track reads as a track that has not finished.
func TestTrackRunsOn(t *testing.T) {
	was := track{Elapsed: time.Second, Length: 2 * time.Second, At: time.Now().Add(-10 * time.Second)}

	if at := was.at(false); at != time.Second {
		t.Errorf("paused at %v, want 1s", at)
	}
	if at := was.at(true); at != 2*time.Second {
		t.Errorf("playing at %v, want the 2s length", at)
	}
}

package player

import (
	"log/slog"
	"sync"

	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/ui"
)

// The album art, decoded once and kept until a different picture arrives. It is drawn every
// redraw and arrives once.
var art struct {
	mu   sync.Mutex
	id   uint64
	img  *ui.Image
	said bool
}

// cover is the artwork to draw, and nil where there is none or it would not decode.
func cover(now media.Now) *ui.Image {
	art.mu.Lock()
	defer art.mu.Unlock()

	if now.ArtID == art.id {
		return art.img
	}
	art.id, art.img, art.said = now.ArtID, nil, false

	if len(now.Art) == 0 {
		return nil
	}

	img, err := ui.Decode(now.Art)
	if err != nil {
		if !art.said {
			slog.Warn("the album art would not decode", "bytes", len(now.Art), "err", err)
			art.said = true
		}
		return nil
	}

	art.img = img
	return art.img
}

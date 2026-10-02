package videoplayer

import (
	"time"

	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

type Controls interface {
	media.Source
	media.Details
	Seek(to time.Duration)
	CanSeek() bool
}

type Mark struct {
	From, To time.Duration
	Color    theme.Color
}

type Marked interface {
	Marks() []Mark
}

const linger = 4 * time.Second

package widget

import (
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

type Kind int

const (
	Plain Kind = iota
	Chevron
	Toggle
	Slider
)

type Row struct {
	Glyph string
	Label string
	Hint  string

	Snap func(level int) int

	Value string

	Kind  Kind
	On    bool
	Level int

	Chosen bool
	Dim    bool

	Preview func(s ui.Surface, at ui.Rect, palette theme.Theme)
}

type Cell struct {
	Label  string
	Chosen bool

	Paint func(s ui.Surface, at ui.Rect, palette theme.Theme)

	Palette *theme.Theme

	Face config.Face
}

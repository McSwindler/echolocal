package config

import "github.com/ygelfand/echolocal/internal/lib/say"

// Clock is the face the panel shows when nothing else is on it.
type Clock struct {
	Face     Face     `json:"face"`
	Date     bool     `json:"date"`
	Position Position `json:"position"`
	Size     Size     `json:"size"`
	Ink      Ink      `json:"ink"`
}

func defaultClock() Clock {
	return Clock{
		Face:     DefaultFace,
		Date:     true,
		Position: DefaultPosition,
		Size:     DefaultSize,
		Ink:      DefaultInk,
	}
}

type Position string

const (
	PositionTop    Position = "top"
	PositionCenter Position = "center"
	PositionBottom Position = "bottom"
)

const DefaultPosition = PositionCenter

func (p Position) Label() string {
	switch p {
	case PositionTop:
		return say.T("position.top")
	case PositionCenter:
		return say.T("position.center")
	case PositionBottom:
		return say.T("position.bottom")
	}
	return string(p)
}

func Positions() []Position { return []Position{PositionTop, PositionCenter, PositionBottom} }

type Size string

const (
	SizeNano   Size = "nano"
	SizeMicro  Size = "micro"
	SizeMini   Size = "mini"
	SizeSmall  Size = "small"
	SizeMedium Size = "medium"
	SizeLarge  Size = "large"
)

const DefaultSize = SizeLarge

func (s Size) Label() string {
	switch s {
	case SizeNano:
		return say.T("size.nano")
	case SizeMicro:
		return say.T("size.micro")
	case SizeMini:
		return say.T("size.mini")
	case SizeSmall:
		return say.T("size.small")
	case SizeMedium:
		return say.T("size.medium")
	case SizeLarge:
		return say.T("size.large")
	}
	return string(s)
}

// Share is how much of the panel each way the face takes.
func (s Size) Share() float64 {
	switch s {
	case SizeNano:
		return 0.16
	case SizeMicro:
		return 0.26
	case SizeMini:
		return 0.40
	case SizeSmall:
		return 0.58
	case SizeMedium:
		return 0.78
	}
	return 1
}

func Sizes() []Size {
	return []Size{SizeNano, SizeMicro, SizeMini, SizeSmall, SizeMedium, SizeLarge}
}

type Face string

const (
	FacePlain         Face = "plain"
	FaceCards         Face = "cards"
	FaceAnalog        Face = "analog"
	FaceAnalogSeconds Face = "analog-seconds"
	FaceSegments      Face = "segments"
	FaceWords         Face = "words"
	FaceOverlap       Face = "overlap"
)

const DefaultFace = FacePlain

func (f Face) Label() string {
	switch f {
	case FaceNone:
		return say.T("face.none")
	case FacePlain:
		return say.T("face.plain")
	case FaceCards:
		return say.T("face.cards")
	case FaceAnalog:
		return say.T("face.analog")
	case FaceAnalogSeconds:
		return say.T("face.analog-seconds")
	case FaceSegments:
		return say.T("face.segments")
	case FaceWords:
		return say.T("face.words")
	case FaceOverlap:
		return say.T("face.overlap")
	}
	return string(f)
}

func Faces() []Face {
	return []Face{FacePlain, FaceCards, FaceAnalog, FaceAnalogSeconds, FaceSegments, FaceWords, FaceOverlap}
}

type ClockWriter struct{ st *Store }

func (w ClockWriter) Face(v Face) error {
	return w.st.Update(func(c *Config) { c.Clock.Face = v })
}

func (w ClockWriter) Date(v bool) error {
	return w.st.Update(func(c *Config) { c.Clock.Date = v })
}

func (w ClockWriter) Position(v Position) error {
	return w.st.Update(func(c *Config) { c.Clock.Position = v })
}

func (w ClockWriter) Size(v Size) error {
	return w.st.Update(func(c *Config) { c.Clock.Size = v })
}

func (w ClockWriter) Ink(v Ink) error {
	return w.st.Update(func(c *Config) { c.Clock.Ink = v })
}

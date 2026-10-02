package config

import "time"

// Idle is what the screen settles into when nobody has touched it for a while.
type Idle struct {
	After    Delay      `json:"after"`
	Face     Face       `json:"face"`
	Position Position   `json:"position"`
	Align    Align      `json:"align"`
	Size     Size       `json:"size"`
	First    IdleVisual `json:"first"`
	Second   IdleVisual `json:"second"`
}

type IdleVisual struct {
	Kind   string `json:"kind"`
	Source Source `json:"source"`
}

func (v IdleVisual) On() bool { return v.Kind != "" }

const FaceNone Face = "none"

func IdleFaces() []Face { return append([]Face{FaceNone}, Faces()...) }

type Delay string

const (
	DelayNever Delay = "never"
	Delay30s   Delay = "30s"
	Delay1m    Delay = "1m"
	Delay2m    Delay = "2m"
	Delay5m    Delay = "5m"
	Delay15m   Delay = "15m"
)

const DefaultDelay = Delay1m

func (d Delay) Label() string {
	switch d {
	case DelayNever:
		return say.T("delay.never")
	case Delay30s:
		return say.T("delay.30s")
	case Delay1m:
		return say.T("delay.1m")
	case Delay2m:
		return say.T("delay.2m")
	case Delay5m:
		return say.T("delay.5m")
	case Delay15m:
		return say.T("delay.15m")
	}
	return string(d)
}

// After is how long the delay is, zero for never.
func (d Delay) After() time.Duration {
	switch d {
	case Delay30s:
		return 30 * time.Second
	case Delay1m:
		return time.Minute
	case Delay2m:
		return 2 * time.Minute
	case Delay5m:
		return 5 * time.Minute
	case Delay15m:
		return 15 * time.Minute
	}
	return 0
}

func Delays() []Delay { return []Delay{Delay30s, Delay1m, Delay2m, Delay5m, Delay15m, DelayNever} }

type Align string

const (
	AlignLeft   Align = "left"
	AlignCenter Align = "center"
	AlignRight  Align = "right"
)

func (a Align) Label() string {
	switch a {
	case AlignLeft:
		return say.T("align.left")
	case AlignCenter:
		return say.T("align.center")
	case AlignRight:
		return say.T("align.right")
	}
	return string(a)
}

func Aligns() []Align { return []Align{AlignLeft, AlignCenter, AlignRight} }

type Source string

const (
	SourceBoth    Source = "both"
	SourceMic     Source = "mic"
	SourceSpeaker Source = "speaker"
)

func (s Source) Label() string {
	switch s {
	case SourceBoth:
		return say.T("source.both")
	case SourceMic:
		return say.T("source.mic")
	case SourceSpeaker:
		return say.T("source.speaker")
	}
	return string(s)
}

func Sources() []Source { return []Source{SourceBoth, SourceMic, SourceSpeaker} }

func defaultIdle() Idle {
	return Idle{
		After:    DefaultDelay,
		Face:     DefaultFace,
		Position: DefaultPosition,
		Align:    AlignCenter,
		Size:     DefaultSize,
		First:    IdleVisual{Source: SourceBoth},
		Second:   IdleVisual{Source: SourceBoth},
	}
}

type IdleWriter struct{ st *Store }

func (w IdleWriter) After(v Delay) error {
	return w.st.Update(func(c *Config) { c.Idle.After = v })
}

func (w IdleWriter) Face(v Face) error {
	return w.st.Update(func(c *Config) { c.Idle.Face = v })
}

func (w IdleWriter) Position(v Position) error {
	return w.st.Update(func(c *Config) { c.Idle.Position = v })
}

func (w IdleWriter) Align(v Align) error {
	return w.st.Update(func(c *Config) { c.Idle.Align = v })
}

func (w IdleWriter) Size(v Size) error {
	return w.st.Update(func(c *Config) { c.Idle.Size = v })
}

func (w IdleWriter) Visual(slot int, v IdleVisual) error {
	return w.st.Update(func(c *Config) {
		if slot == 0 {
			c.Idle.First = v
		} else {
			c.Idle.Second = v
		}
	})
}

package voice

import "github.com/ygelfand/echolocal/internal/lib/hook"

// Phase is what a turn is doing, for whatever shows it.
type Phase int

const (
	Idle Phase = iota
	Listening
	Thinking
	Replying
)

func (p Phase) String() string {
	switch p {
	case Listening:
		return "listening"
	case Thinking:
		return "thinking"
	case Replying:
		return "replying"
	}
	return "idle"
}

// Showing is a turn as something to look at: the phase, the slot whose assistant it belongs to, and
// the words either side of it.
type Showing struct {
	Phase Phase
	Slot  int

	Said  string
	Reply string
}

// Shown carries every phase change.
var Shown hook.Hook[Showing]

func shown(p phase) Phase {
	switch p {
	case phaseListening:
		return Listening
	case phaseThinking:
		return Thinking
	case phaseReplying:
		return Replying
	}
	return Idle
}

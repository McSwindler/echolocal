// Package indicate is how the device shows what it is doing, without naming what it shows it on.
//
// A Dot has a ring and a Show has a panel. What a conversation, a failure or a cut microphone wants
// to say is the same either way, so features claim a surface here and the board decides which one.
//
// Geometry is deliberately absent: a twelve-segment arc means nothing to a screen, so anything that
// draws one talks to the ring directly and is gated on the board having one.
package indicate

import "time"

// Priority is how a surface resolves being asked for two things at once. Higher wins, and when a
// claim goes away whatever is under it comes back on its own.
type Priority int

const (
	PriorityBase Priority = iota
	PriorityRoom
	PriorityMute
	PriorityTimer
	PriorityBusy
	PriorityTurn
	PriorityAlarm
	PriorityNotice
	PriorityTrouble
	PriorityBoot
)

type Color struct{ R, G, B byte }

// Room is what an effect that reacts to the room reads. Whoever claims the surface supplies it,
// because the surface has no business knowing where the microphones are.
type Room struct {
	// Level is 0 for quiet to 1 for someone talking close by, against the room's own noise floor.
	Level func() float64

	// Facing is where the loudest sound is, as a fraction clockwise from the front, and whether that
	// is known: it takes a frame of asking, and an empty room has no answer.
	Facing func() (float64, bool)
}

// Claim is a hold on the surface at one priority. Nothing is shown until it is given something, and
// whatever it shows lasts until it is released or something higher takes over.
type Claim interface {
	Play(effect string, base Color)
	PlayReversed(effect string, base Color)
	React(effect string, base Color, room Room)
	ShowFor(effect string, base Color, d time.Duration)
	Clear()
	Release()
}

type Surface interface {
	Claim(Priority) Claim

	// HoldOnStop is what to leave showing once the process goes away, for one that is coming back.
	HoldOnStop(Color)

	Present() bool
}

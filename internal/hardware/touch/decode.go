package touch

import (
	"time"

	"github.com/ygelfand/echolocal/internal/lib/input"
)

// Protocol B axes. A contact is a slot, and a slot holds a tracking id until it goes to -1.
const (
	absMTSlot       = 0x2f
	absMTPositionX  = 0x35
	absMTPositionY  = 0x36
	absMTTrackingID = 0x39

	synReport = 0x00
)

// released is the tracking id of a slot nobody is touching.
const released = -1

// Phase is what happened to a contact.
type Phase int

const (
	Down Phase = iota
	Move
	Up
)

func (p Phase) String() string {
	switch p {
	case Down:
		return "down"
	case Move:
		return "move"
	case Up:
		return "up"
	}
	return "unknown"
}

// Contact is one finger, in viewed coordinates.
type Contact struct {
	Slot  int
	ID    int
	X, Y  int
	Phase Phase
	At    time.Time
}

// slot is what is known about one contact between reports.
type slot struct {
	id      int
	x, y    int
	touched bool
	changed bool

	// lifting holds the id until the Up has gone out, which is what matches it to its own Down.
	lifting bool
}

// decoder turns the event stream into contacts. The kernel sends only what changed, so a finger
// that is still moving keeps whatever it last said.
type decoder struct {
	current int
	slots   map[int]*slot

	// place maps a panel position into the coordinates the drawing uses.
	place func(x, y int) (int, int)
}

func (d *decoder) at(n int) *slot {
	if d.slots == nil {
		d.slots = map[int]*slot{}
	}
	if d.slots[n] == nil {
		d.slots[n] = &slot{id: released}
	}
	return d.slots[n]
}

// event takes one event and returns the contacts to report, which is nothing until SYN_REPORT says
// the batch is complete.
func (d *decoder) event(e input.Event, now time.Time) []Contact {
	switch e.Type {
	case input.EvAbs:
		d.abs(e.Code, int(e.Value))
	case input.EvSyn:
		if e.Code == synReport {
			return d.report(now)
		}
	}
	return nil
}

func (d *decoder) abs(code uint16, value int) {
	switch code {
	case absMTSlot:
		d.current = value
	case absMTTrackingID:
		s := d.at(d.current)
		s.changed = true
		if value == released {
			s.lifting = true
		} else {
			s.id, s.lifting = value, false
			s.touched = false
		}
	case absMTPositionX:
		s := d.at(d.current)
		s.x, s.changed = value, true
	case absMTPositionY:
		s := d.at(d.current)
		s.y, s.changed = value, true
	}
}

// report drains what changed since the last one.
func (d *decoder) report(now time.Time) []Contact {
	var out []Contact

	for n, s := range d.slots {
		if !s.changed {
			continue
		}
		s.changed = false

		x, y := s.x, s.y
		if d.place != nil {
			x, y = d.place(x, y)
		}

		switch {
		case s.lifting:
			if s.touched {
				out = append(out, Contact{Slot: n, ID: s.id, X: x, Y: y, Phase: Up, At: now})
			}
			s.id, s.touched, s.lifting = released, false, false
		case s.id == released:
		case !s.touched:
			s.touched = true
			out = append(out, Contact{Slot: n, ID: s.id, X: x, Y: y, Phase: Down, At: now})
		default:
			out = append(out, Contact{Slot: n, ID: s.id, X: x, Y: y, Phase: Move, At: now})
		}
	}
	return out
}

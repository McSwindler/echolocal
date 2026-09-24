package clock

import "time"

// Reading is what the clock says at a moment, worked out apart from any drawing so it can be
// checked without a panel.
type Reading struct {
	Time   string
	Suffix string
	Date   string
}

// Read is what to show at this moment.
func Read(at time.Time, twentyFour bool) Reading {
	r := Reading{Date: at.Format("Monday, 2 January")}

	if twentyFour {
		r.Time = at.Format("15:04")
		return r
	}

	r.Time = at.Format("3:04")
	r.Suffix = at.Format("PM")
	return r
}

// String is what the panel is showing, for deciding whether to draw it again.
func (r Reading) String() string { return r.Time + r.Suffix + r.Date }

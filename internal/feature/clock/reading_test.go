package clock

import (
	"testing"
	"time"
)

func TestReadTwentyFourHasNoSuffix(t *testing.T) {
	at := time.Date(2026, 9, 23, 21, 15, 30, 0, time.UTC)

	r := Read(at, true)
	if r.Time != "21:15" {
		t.Errorf("time is %q", r.Time)
	}
	if r.Suffix != "" {
		t.Errorf("a 24 hour clock carries the suffix %q", r.Suffix)
	}
	if r.Date != "Wednesday, 23 September" {
		t.Errorf("date is %q", r.Date)
	}
}

func TestReadTwelveHourCarriesTheSuffixApart(t *testing.T) {
	at := time.Date(2026, 9, 23, 21, 15, 0, 0, time.UTC)

	r := Read(at, false)
	if r.Time != "9:15" {
		t.Errorf("time is %q", r.Time)
	}
	if r.Suffix != "PM" {
		t.Errorf("suffix is %q", r.Suffix)
	}
}

// The panel is written pixel by pixel, so it is only redrawn when the reading changed.
func TestSecondsDoNotChangeTheReading(t *testing.T) {
	at := time.Date(2026, 9, 23, 21, 15, 0, 0, time.UTC)

	for _, s := range []int{1, 17, 59} {
		if got := Read(at.Add(time.Duration(s)*time.Second), true); got != Read(at, true) {
			t.Errorf("%ds in the reading is %v", s, got)
		}
	}
	if Read(at.Add(time.Minute), true) == Read(at, true) {
		t.Error("a new minute reads the same")
	}
}

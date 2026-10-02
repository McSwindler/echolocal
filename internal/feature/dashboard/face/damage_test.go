package face

import (
	"testing"
	"time"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// boxes are the shapes a face is handed: the whole panel either way up, and the smaller bands the
// position and size settings cut out of it.
var boxes = []ui.Rect{
	{W: 1200, H: 1920},
	{W: 1920, H: 1200},
	{X: 0, Y: 0, W: 1200, H: 1382},
	{X: 180, Y: 288, W: 840, H: 1344},
}

// changedBy renders both readings and reports what actually moved.
func changedBy(t *testing.T, f Face, in ui.Rect, from, to Reading) ui.Rect {
	t.Helper()

	palette := theme.Default()
	w, h := in.X+in.W, in.Y+in.H

	a := ui.NewImage(w, h, palette.Background)
	ui.Fill(a, palette.Background)
	f.Draw(a, in, from, palette)

	b := ui.NewImage(w, h, palette.Background)
	ui.Fill(b, palette.Background)
	f.Draw(b, in, to, palette)

	return ui.Changed(a, b)
}

// ticks are the minute transitions worth rendering, at every box shape.
//
// Not all 1440 of them: that is 5760 pairs of full-screen renders and twenty-odd seconds, for
// coverage an hour of ordinary minutes already gives. What is sampled instead is every case where
// the line changes width and so moves — every hour rollover, which includes 9:59 to 10:00, and
// midnight, which is the one that also changes the date.
func ticks() []time.Time {
	day := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)

	var out []time.Time
	for m := range 60 {
		out = append(out, day.Add(time.Duration(m)*time.Minute))
	}
	for hour := range 24 {
		out = append(out, day.Add(time.Duration(hour)*time.Hour-time.Minute))
	}
	return append(out, day.Add(-time.Minute), day.Add(12*time.Hour-time.Minute))
}

// The declared rectangle has to hold everything that moved, at every shape a face is handed and in
// both clock formats.
//
// Both formats, because they fail differently. Twenty-four hour time is always five characters, so
// the line never changes width and only the digits move — that is what this device is set to.
// Twelve hour time goes from 9:59 to 10:00 and the whole centred line shifts, which is what catches
// a face that declares only where the new reading lands.
func TestPlainDeclaresEveryChangeItMakes(t *testing.T) {
	f := Of(config.FacePlain)

	if _, ok := f.(Damaging); !ok {
		t.Fatal("the plain face no longer declares its damage")
	}

	for _, twentyFour := range []bool{true, false} {
		for _, in := range boxes {
			for _, at := range ticks() {
				from, to := Read(at, twentyFour), Read(at.Add(time.Minute), twentyFour)

				if from.Time == to.Time && from.Date == to.Date && from.Suffix == to.Suffix {
					continue
				}

				declared, ok := Damage(config.FacePlain, in, from, to)
				if !ok {
					t.Fatalf("24h=%v %+v %s -> %s: the face declared nothing",
						twentyFour, in, from.Time, to.Time)
				}

				if moved := changedBy(t, f, in, from, to); !declared.Holds(moved) {
					t.Fatalf("24h=%v %+v %s -> %s: moved %+v, declared %+v",
						twentyFour, in, from.Time, to.Time, moved, declared)
				}
			}
		}
	}
}

// Midnight is the one that changes the date as well as the time, and a date changes width — Monday
// to Tuesday, or the 9th to the 10th — so the line under the clock moves too.
func TestPlainDeclaresTheDateChanging(t *testing.T) {
	f := Of(config.FacePlain)
	in := ui.Rect{W: 1200, H: 1920}

	for _, at := range []time.Time{
		time.Date(2026, 9, 21, 23, 59, 0, 0, time.UTC),
		time.Date(2026, 9, 9, 23, 59, 0, 0, time.UTC),
		time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC),
	} {
		for _, twentyFour := range []bool{true, false} {
			from, to := Read(at, twentyFour), Read(at.Add(time.Minute), twentyFour)

			declared, ok := Damage(config.FacePlain, in, from, to)
			if !ok {
				t.Fatalf("%s: the face declared nothing", from.Date)
			}
			if moved := changedBy(t, f, in, from, to); !declared.Holds(moved) {
				t.Errorf("24h=%v %q -> %q: moved %+v, declared %+v",
					twentyFour, from.Date, to.Date, moved, declared)
			}
		}
	}
}

// A device set to show only the time draws no date at all, so the layout is one line and the
// rectangle has to shrink with it rather than covering a date that is not there.
func TestPlainDeclaresWithNoDate(t *testing.T) {
	f := Of(config.FacePlain)
	in := ui.Rect{W: 1200, H: 1920}

	at := time.Date(2026, 9, 21, 9, 59, 0, 0, time.UTC)
	from, to := Read(at, false).Undated(), Read(at.Add(time.Minute), false).Undated()

	declared, ok := Damage(config.FacePlain, in, from, to)
	if !ok {
		t.Fatal("the face declared nothing")
	}
	if moved := changedBy(t, f, in, from, to); !declared.Holds(moved) {
		t.Errorf("moved %+v, declared %+v", moved, declared)
	}
}

// The point of declaring it. A whole panel is 2.3 million pixels and a minute should not cost them.
func TestPlainDeclaresFarLessThanTheWholeBox(t *testing.T) {
	in := ui.Rect{W: 1200, H: 1920}

	from := Read(time.Date(2026, 9, 21, 14, 37, 0, 0, time.UTC), false)
	to := Read(time.Date(2026, 9, 21, 14, 38, 0, 0, time.UTC), false)

	declared, ok := Damage(config.FacePlain, in, from, to)
	if !ok {
		t.Fatal("the face declared nothing")
	}

	share := float64(declared.W*declared.H) / float64(in.W*in.H)
	if share > 0.35 {
		t.Errorf("a minute declares %.0f%% of the box, which is not worth the bookkeeping", share*100)
	}
}

// declaring is every face that opts in, and the most of its box one ordinary minute may declare.
//
// A face is added here the moment it implements Damaging, and the sweep below then holds it to the
// same contract as the rest without a test of its own. The share is what makes the opt-in worth
// having: a face declaring most of its box has done the bookkeeping and saved nothing.
var declaring = []struct {
	name  config.Face
	share float64
}{
	{config.FacePlain, 0.35},

	// A whole card out of two, plus the gap between them on one axis.
	{config.FaceCards, 0.30},

	// One line of two, and the lines are sized off the height this panel has to spare.
	{config.FaceStack, 0.30},

	// The best of them: the lamps do not move, so a tick is one cell out of four.
	{config.FaceSegments, 0.08},

	// The two blocks cross, and the crossing is inside the minutes' own rectangle.
	{config.FaceOverlap, 0.35},

	// A hand sweeping six degrees, plus the hour hand creeping half of one.
	{config.FaceAnalog, 0.35},
	{config.FaceAnalogSeconds, 0.35},
}

// Every declaring face, held to the rectangle it promised, at every box shape and in both clock
// formats.
//
// This is the check the whole opt-in rests on: draw both readings, find what actually moved, and
// require the declaration to cover it. A face that declares too little leaves a stale strip.
func TestEveryDeclaringFaceDeclaresEveryChangeItMakes(t *testing.T) {
	for _, face := range declaring {
		t.Run(string(face.name), func(t *testing.T) {
			f := Of(face.name)
			if _, ok := f.(Damaging); !ok {
				t.Fatalf("%s is listed as declaring and does not implement Damaging", face.name)
			}

			for _, twentyFour := range []bool{true, false} {
				for _, in := range boxes {
					for _, at := range ticks() {
						from := Read(at, twentyFour)
						to := Read(at.Add(time.Minute), twentyFour)

						if from.Time == to.Time && from.Date == to.Date &&
							from.Suffix == to.Suffix {
							continue
						}

						declared, ok := Damage(face.name, in, from, to)
						if !ok {
							t.Fatalf("24h=%v %+v %s -> %s: declared nothing",
								twentyFour, in, from.Time, to.Time)
						}

						moved := changedBy(t, f, in, from, to)
						if !declared.Holds(moved) {
							t.Fatalf("24h=%v %+v %s -> %s: moved %+v, declared %+v",
								twentyFour, in, from.Time, to.Time, moved, declared)
						}
					}
				}
			}
		})
	}
}

// The check the whole thing actually rests on, rather than a proxy for it: paint the old frame,
// repaint only what the face declared over the top of it, and require the result to be identical to
// a full repaint. That is exactly what the driver does to the panel through Claim.ShowIn.
//
// Holds against ui.Changed says the same thing and says it faster, which is why the sweep uses it.
// This one is here because it is the claim itself, and a proxy that drifted from it would be a
// stale strip on the screen and a green test.
func TestRepaintingOnlyWhatWasDeclaredMatchesAFullRepaint(t *testing.T) {
	palette := theme.Default()
	at := time.Date(2026, 9, 21, 9, 59, 0, 0, time.UTC)

	for _, face := range declaring {
		t.Run(string(face.name), func(t *testing.T) {
			f := Of(face.name)

			for _, twentyFour := range []bool{true, false} {
				for _, in := range boxes {
					from := Read(at, twentyFour)
					to := Read(at.Add(time.Minute), twentyFour)

					declared, ok := Damage(face.name, in, from, to)
					if !ok {
						t.Fatal("declared nothing")
					}

					w, h := in.X+in.W, in.Y+in.H

					// The old frame, then the new one painted only inside what was declared.
					partial := ui.NewImage(w, h, palette.Background)
					ui.Fill(partial, palette.Background)
					f.Draw(partial, in, from, palette)

					clipped := ui.Within(partial, declared)
					ui.Fill(clipped, palette.Background)
					f.Draw(clipped, in, to, palette)

					whole := ui.NewImage(w, h, palette.Background)
					ui.Fill(whole, palette.Background)
					f.Draw(whole, in, to, palette)

					if diff := ui.Changed(partial, whole); diff.W > 0 || diff.H > 0 {
						t.Errorf("24h=%v %+v: repainting only %+v left %+v unlike a full repaint",
							twentyFour, in, declared, diff)
					}
				}
			}
		})
	}
}

// The undated case, which changes the layout rather than the reading: a device set to show only the
// time has no line under the clock, so the rectangle has to shrink with it.
func TestEveryDeclaringFaceDeclaresWithNoDate(t *testing.T) {
	in := ui.Rect{W: 1200, H: 1920}
	at := time.Date(2026, 9, 21, 9, 59, 0, 0, time.UTC)

	for _, face := range declaring {
		t.Run(string(face.name), func(t *testing.T) {
			f := Of(face.name)
			from := Read(at, false).Undated()
			to := Read(at.Add(time.Minute), false).Undated()

			declared, ok := Damage(face.name, in, from, to)
			if !ok {
				t.Fatal("declared nothing")
			}
			if moved := changedBy(t, f, in, from, to); !declared.Holds(moved) {
				t.Errorf("moved %+v, declared %+v", moved, declared)
			}
		})
	}
}

// The point of declaring it. A whole panel is 2.3 million pixels and a minute should not cost them.
func TestEveryDeclaringFaceSavesSomethingWorthHaving(t *testing.T) {
	in := ui.Rect{W: 1200, H: 1920}

	from := Read(time.Date(2026, 9, 21, 14, 37, 0, 0, time.UTC), false)
	to := Read(time.Date(2026, 9, 21, 14, 38, 0, 0, time.UTC), false)

	for _, face := range declaring {
		t.Run(string(face.name), func(t *testing.T) {
			declared, ok := Damage(face.name, in, from, to)
			if !ok {
				t.Fatal("declared nothing")
			}

			share := float64(declared.W*declared.H) / float64(in.W*in.H)
			if share > face.share {
				t.Errorf("a minute declares %.1f%% of the box, more than the %.0f%% budgeted",
					share*100, face.share*100)
			}
			t.Logf("a minute declares %.1f%% of the box", share*100)
		})
	}
}

// The variant with a second hand repaints every second, which is the frame this whole task is
// about: sixty full panels a minute where what moved is one sliver six degrees wide.
//
// Every second of a minute, including the ones that carry the minute hand with them.
func TestAnalogSecondsDeclaresEverySecondTick(t *testing.T) {
	f := Of(config.FaceAnalogSeconds)
	in := ui.Rect{W: 1200, H: 1920}

	// From just before the minute, so the roll where all three hands move is included.
	at := time.Date(2026, 9, 21, 14, 37, 30, 0, time.UTC)

	var worst float64
	for s := range 60 {
		from := Read(at.Add(time.Duration(s)*time.Second), true)
		to := Read(at.Add(time.Duration(s+1)*time.Second), true)

		declared, ok := Damage(config.FaceAnalogSeconds, in, from, to)
		if !ok {
			t.Fatalf("second %d: declared nothing", from.Second)
		}
		if moved := changedBy(t, f, in, from, to); !declared.Holds(moved) {
			t.Fatalf("second %d -> %d: moved %+v, declared %+v",
				from.Second, to.Second, moved, declared)
		}

		worst = max(worst, float64(declared.W*declared.H)/float64(in.W*in.H))
	}

	t.Logf("the worst second declares %.1f%% of the box", worst*100)
	if worst > 0.30 {
		t.Errorf("a second declares up to %.1f%% of the box, which is not worth the bookkeeping",
			worst*100)
	}
}

// The face without a second hand must not answer differently for two readings that differ only in
// their seconds: it does not draw them, so nothing changed and nothing needs repainting.
func TestAnalogWithoutSecondsIgnoresThem(t *testing.T) {
	in := ui.Rect{W: 1200, H: 1920}
	at := time.Date(2026, 9, 21, 14, 37, 10, 0, time.UTC)

	from := Read(at, true)
	to := Read(at.Add(20*time.Second), true)

	f := Of(config.FaceAnalog)
	if moved := changedBy(t, f, in, from, to); moved.W > 0 || moved.H > 0 {
		t.Errorf("the face without a second hand drew the seconds: %+v", moved)
	}

	declared, ok := Damage(config.FaceAnalog, in, from, to)
	if !ok {
		t.Fatal("declared nothing")
	}
	if moved := changedBy(t, f, in, from, to); !declared.Holds(moved) {
		t.Errorf("moved %+v, declared %+v", moved, declared)
	}
}

// The hour card does not change for fifty-nine minutes out of sixty, and that is the whole reason
// this face is worth declaring rather than repainting.
func TestCardsLeavesTheHourAloneWithinTheHour(t *testing.T) {
	in := ui.Rect{W: 1200, H: 1920}

	from := Read(time.Date(2026, 9, 21, 14, 37, 0, 0, time.UTC), true)
	to := Read(time.Date(2026, 9, 21, 14, 38, 0, 0, time.UTC), true)

	declared, ok := Damage(config.FaceCards, in, from, to)
	if !ok {
		t.Fatal("the cards face declared nothing")
	}

	layout, _ := cardsArrange(in, from)

	// The hour's card is untouched, so anything inside it away from the shared edge must be
	// outside what was declared.
	hour := layout.First.Inset(2)
	if declared.Holds(hour) {
		t.Errorf("a minute tick declared the whole hour card: declared %+v, hour %+v",
			declared, hour)
	}
	if !declared.Holds(layout.Second) {
		t.Errorf("the minutes card is not covered: declared %+v, minutes %+v",
			declared, layout.Second)
	}
}

// Both cards change on the hour, and at midnight the date does too.
func TestCardsDeclaresTheHourAndTheDateRolling(t *testing.T) {
	in := ui.Rect{W: 1200, H: 1920}
	f := Of(config.FaceCards)

	for _, at := range []time.Time{
		time.Date(2026, 9, 21, 14, 59, 0, 0, time.UTC),
		time.Date(2026, 9, 21, 23, 59, 0, 0, time.UTC),
		time.Date(2026, 9, 9, 23, 59, 0, 0, time.UTC),
		time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC),
	} {
		for _, twentyFour := range []bool{true, false} {
			from, to := Read(at, twentyFour), Read(at.Add(time.Minute), twentyFour)

			declared, ok := Damage(config.FaceCards, in, from, to)
			if !ok {
				t.Fatalf("%s: declared nothing", from.Time)
			}
			if moved := changedBy(t, f, in, from, to); !declared.Holds(moved) {
				t.Errorf("24h=%v %s -> %s: moved %+v, declared %+v",
					twentyFour, from.Time, to.Time, moved, declared)
			}
		}
	}
}

// Drawing and saying where things are come from the same arithmetic, so a card the layout reports
// has to be where the face actually painted one.
func TestTheCardsLayoutIsWhereTheCardsAre(t *testing.T) {
	for _, in := range boxes {
		r := Read(time.Date(2026, 9, 21, 14, 37, 0, 0, time.UTC), true)
		layout, _ := cardsArrange(in, r)

		if !in.Holds(layout.bounds()) {
			t.Errorf("%+v: the face reports %+v, outside its box", in, layout.bounds())
		}
		if layout.First.W != layout.Side || layout.First.H != layout.Side {
			t.Errorf("%+v: a card is %dx%d, want square at %d",
				in, layout.First.W, layout.First.H, layout.Side)
		}

		// The two cards do not overlap, or one repainted would take the other with it.
		if layout.First.Holds(layout.Second) || layout.Second.Holds(layout.First) {
			t.Errorf("%+v: the cards overlap: %+v and %+v", in, layout.First, layout.Second)
		}
	}
}

// A face that says nothing about damage is not asked to: the caller repaints the lot, which is what
// every face got before any of them answered.
func TestAFaceThatDeclaresNothing(t *testing.T) {
	in := ui.Rect{W: 1200, H: 1920}
	from := Read(time.Date(2026, 9, 21, 14, 37, 0, 0, time.UTC), false)
	to := Read(time.Date(2026, 9, 21, 14, 38, 0, 0, time.UTC), false)

	for _, name := range []config.Face{config.FaceWords, config.FaceAnalog} {
		if _, ok := Of(name).(Damaging); ok {
			continue
		}
		if _, ok := Damage(name, in, from, to); ok {
			t.Errorf("%s does not implement Damaging but answered with a rectangle", name)
		}
	}
}

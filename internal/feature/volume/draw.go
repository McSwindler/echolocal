package volume

import (
	"strconv"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
)

// The capsule, as fractions of the screen.
const (
	capsuleWidthShare = 0.13
	trackHeightShare  = 0.30
	marginShare       = 0.05
)

// What is in it, as fractions of the capsule's width.
const (
	padShare = 0.16
	gapShare = 0.16

	// The head overhangs the capsule a little.
	headShare = 1.08

	// The number wants more room under it than things want between them.
	footShare = 0.34

	buttonShare = 0.74
	buttonRound = 0.30

	valueShare = 0.30

	// pressedShare is how far something moves towards the text colour under a finger.
	pressedShare = 0.18
)

// Card is the volume control: one kind of sound at a time, and which one.
type Card struct {
	// Streams in the order they are offered, and the level of each.
	Streams []config.Stream
	Levels  []int

	// Selected indexes the one the bar moves, which is the one in the head at the top.
	Selected int

	// Edge is the side it hangs off, left or right.
	Edge config.Edge

	// Open is whether the ones it is not showing are revealed.
	Open bool
}

// Places is where the card landed, so a touch resolves against what was drawn.
type Places struct {
	// Head is the one the bar is moving, and the button that shows the rest.
	Head ui.Rect

	// Others runs in the same order as Streams, empty at the selected one and throughout while the
	// card is closed.
	Others []ui.Rect

	// Track is the bar, and Capsule the whole thing.
	Track   ui.Rect
	Capsule ui.Rect
}

// Bounds is everything the card covers, which is the capsule plus the head overhanging it.
func (p Places) Bounds() ui.Rect {
	if p.Capsule.W <= 0 || p.Capsule.H <= 0 {
		return p.Head
	}
	if p.Head.W <= 0 || p.Head.H <= 0 {
		return p.Capsule
	}

	left := min(p.Capsule.X, p.Head.X)
	top := min(p.Capsule.Y, p.Head.Y)
	right := max(p.Capsule.X+p.Capsule.W, p.Head.X+p.Head.W)
	bottom := max(p.Capsule.Y+p.Capsule.H, p.Head.Y+p.Head.H)

	return ui.Rect{X: left, Y: top, W: right - left, H: bottom - top}
}

// DrawCard paints the card over whatever is underneath and answers where it put everything.
//
// Laid out from the bottom up. Opening grows it upwards, so the bar and the number stay where they
// were.
func DrawCard(s ui.Surface, c Card, palette theme.Theme, pressed func(int) bool) Places {
	w, h := s.Size()
	side := min(w, h)

	if len(c.Streams) == 0 {
		return Places{}
	}
	sel := max(0, min(c.Selected, len(c.Streams)-1))

	var (
		width  = int(float64(side) * capsuleWidthShare)
		pad    = int(float64(width) * padShare)
		gap    = int(float64(width) * gapShare)
		head   = int(float64(width) * headShare)
		button = int(float64(width) * buttonShare)
		track  = int(float64(h) * trackHeightShare)
		foot   = int(float64(width) * footShare)
		margin = int(float64(side) * marginShare)
	)

	valueFont := ui.MustLoad(ui.Medium, int(float64(width)*valueShare))
	_, valueH := valueFont.Measure("100")

	shown := 0
	if c.Open {
		shown = len(c.Streams) - 1
	}

	closed := head + gap + track + gap + valueH + foot
	tall := closed + shown*(button+gap)

	// The bar gives up what the card cannot fit: a foot off the bottom takes the number with it.
	if over := tall - (h - 2*margin); over > 0 {
		track = max(track-over, width)
		closed = head + gap + track + gap + valueH + foot
		tall = closed + shown*(button+gap)
	}

	x := w - margin - width
	if c.Edge == config.EdgeLeft {
		x = margin
	}

	bottom := (h + closed) / 2
	if top := bottom - tall; top < margin {
		bottom += margin - top
	}

	capsule := ui.Rect{X: x, Y: bottom - tall, W: width, H: tall}
	ui.FillRounded(s, capsule, width/2, palette.Surface)

	places := Places{Capsule: capsule, Others: make([]ui.Rect, len(c.Streams))}

	places.Head = ui.Rect{X: x - (head-width)/2, Y: capsule.Y, W: head, H: head}
	drawHead(s, places.Head, c.Streams[sel], c.Levels[sel], pressed(0), palette)

	y := capsule.Y + head + gap
	at := 1
	for i, stream := range c.Streams {
		if i == sel || !c.Open {
			continue
		}

		box := ui.Rect{X: x + (width-button)/2, Y: y, W: button, H: button}
		places.Others[i] = box
		drawOther(s, box, stream, c.Levels[i], pressed(at), palette)

		y += button + gap
		at++
	}

	places.Track = ui.Rect{X: x + pad, Y: y, W: width - 2*pad, H: track}
	drawTrack(s, places.Track, c.Levels[sel], palette)

	ui.DrawTextIn(s, valueFont,
		ui.Rect{X: x, Y: places.Track.Y + track + gap, W: width, H: valueH},
		palette.Muted, palette.Surface, strconv.Itoa(clamp(c.Levels[sel])))

	return places
}

// drawHead is the one the bar is moving, filled in the accent.
func drawHead(s ui.Surface, at ui.Rect, stream config.Stream, level int, pressed bool, palette theme.Theme) {
	on := palette.Accent
	if pressed {
		on = on.Blend(palette.Text, pressedShare)
	}

	ui.FillRounded(s, at, at.W/2, on)
	ui.DrawIcon(s, glyph(stream, level), at.Inset(at.W/4), palette.Background, on)
}

// drawOther is one of the kinds the bar is not moving, which tapping selects.
func drawOther(s ui.Surface, at ui.Rect, stream config.Stream, level int, pressed bool, palette theme.Theme) {
	on := palette.Surface.Blend(palette.Text, 0.08)
	if pressed {
		on = palette.Surface.Blend(palette.Text, pressedShare)
	}

	ui.FillRounded(s, at, int(float64(at.W)*buttonRound), on)
	ui.DrawIcon(s, glyph(stream, level), at.Inset(at.W/4), palette.Text, on)
}

// drawTrack fills the bar from the bottom up.
func drawTrack(s ui.Surface, track ui.Rect, level int, palette theme.Theme) {
	if track.H <= 0 {
		return
	}
	radius := track.W / 2

	ui.FillRounded(s, track, radius, palette.Surface.Blend(palette.Text, 0.04))

	filled := track.H * clamp(level) / 100
	if filled < track.W {
		// A rounded fill shorter than it is wide reads as a dot.
		if filled < radius {
			return
		}
		filled = track.W
	}

	ui.FillRounded(s, ui.Rect{
		X: track.X, Y: track.Y + track.H - filled,
		W: track.W, H: filled,
	}, radius, palette.Accent)
}

// LevelAt is the level a finger at this height means, for a track filled from the bottom up.
func LevelAt(track ui.Rect, y int) int {
	if track.H <= 0 {
		return 0
	}
	return clamp((track.Y + track.H - y) * 100 / track.H)
}

// glyph says which kind of sound it is, and whether it is silent.
func glyph(stream config.Stream, level int) ui.Icon {
	silent := level <= 0

	switch stream {
	case config.StreamMedia:
		if silent {
			return icons.AVVolumeOff
		}
		return icons.ImageAudiotrack

	case config.StreamAlerts:
		if silent {
			return icons.SocialNotificationsOff
		}
		return icons.SocialNotifications

	case config.StreamVoice:
		if silent {
			return icons.AVMicOff
		}
		return icons.ActionRecordVoiceOver

	case config.StreamFeedback:
		if silent {
			return icons.AVVolumeMute
		}
		return icons.ActionTouchApp
	}

	switch {
	case silent:
		return icons.AVVolumeOff
	case level < 50:
		return icons.AVVolumeDown
	}
	return icons.AVVolumeUp
}

func clamp(level int) int {
	switch {
	case level < 0:
		return 0
	case level > 100:
		return 100
	}
	return level
}

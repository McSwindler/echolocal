package config

// Stream is a kind of sound, each with its own level.
type Stream string

const (
	// StreamMain is the device's loudness, which the others are set against.
	StreamMain Stream = "main"

	StreamMedia    Stream = "media"
	StreamVoice    Stream = "voice"
	StreamAlerts   Stream = "alerts"
	StreamFeedback Stream = "feedback"
)

// Streams is every level, in the order they are offered.
func Streams() []Stream {
	return []Stream{StreamMain, StreamMedia, StreamVoice, StreamAlerts, StreamFeedback}
}

// Label is how a level is named on the panel and in Home Assistant.
func (s Stream) Label() string {
	switch s {
	case StreamMain:
		return "Main"
	case StreamMedia:
		return "Music"
	case StreamVoice:
		return "Voice"
	case StreamAlerts:
		return "Alerts"
	case StreamFeedback:
		return "Beeps"
	}
	return string(s)
}

// Volume is how loud each kind of sound is, as a percentage. Main lives in Speaker.Volume, which
// is the level the hardware itself carries.
type Volume struct {
	Media    int `json:"media"`
	Voice    int `json:"voice"`
	Alerts   int `json:"alerts"`
	Feedback int `json:"feedback"`
}

// DefaultLevel is where a kind of sound sits until somebody moves it.
const DefaultLevel = 100

func defaultVolume() Volume {
	return Volume{
		Media:    DefaultLevel,
		Voice:    DefaultLevel,
		Alerts:   DefaultLevel,
		Feedback: DefaultLevel,
	}
}

// Level is how loud one kind of sound is.
func (v Volume) Level(s Stream) int {
	switch s {
	case StreamMedia:
		return v.Media
	case StreamVoice:
		return v.Voice
	case StreamAlerts:
		return v.Alerts
	case StreamFeedback:
		return v.Feedback
	}
	return DefaultLevel
}

type VolumeWriter struct{ st *Store }

func (w VolumeWriter) Level(s Stream, level int) error {
	return w.st.Update(func(c *Config) {
		switch s {
		case StreamMedia:
			c.Volume.Media = level
		case StreamVoice:
			c.Volume.Voice = level
		case StreamAlerts:
			c.Volume.Alerts = level
		case StreamFeedback:
			c.Volume.Feedback = level
		}
	})
}

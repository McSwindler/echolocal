package config

// Cast is the Chromecast receiver and what it plays.
type Cast struct {
	Receiver bool    `json:"receiver"`
	Oracle   string  `json:"oracle,omitempty"`
	YouTube  YouTube `json:"youtube,omitzero"`
}

type YouTube struct {
	Device string `json:"device,omitempty"`
	Music  string `json:"music,omitempty"`
	Video  string `json:"video,omitempty"`

	Skip []string `json:"skip"`

	OnDemand bool `json:"onDemand,omitempty"`

	LiveDelay int `json:"liveDelay"`
}

const (
	LiveDelayLeast = 4
	LiveDelayMost  = 60
)

func defaultCast() Cast {
	return Cast{Receiver: false, YouTube: YouTube{Skip: []string{"sponsor"}, LiveDelay: 10}}
}

type CastWriter struct{ st *Store }

func (w CastWriter) Receiver(v bool) error {
	return w.st.Update(func(c *Config) { c.Cast.Receiver = v })
}

func (w CastWriter) Oracle(url string) error {
	return w.st.Update(func(c *Config) { c.Cast.Oracle = url })
}

func (w CastWriter) Screens(device, music, video string) error {
	return w.st.Update(func(c *Config) {
		c.Cast.YouTube.Device, c.Cast.YouTube.Music, c.Cast.YouTube.Video = device, music, video
	})
}

func (w CastWriter) Skip(categories []string) error {
	if categories == nil {
		categories = []string{}
	}
	return w.st.Update(func(c *Config) { c.Cast.YouTube.Skip = categories })
}

func (w CastWriter) LiveDelay(seconds int) error {
	seconds = min(max(seconds, LiveDelayLeast), LiveDelayMost)
	return w.st.Update(func(c *Config) { c.Cast.YouTube.LiveDelay = seconds })
}

func (w CastWriter) LoungeOnDemand(v bool) error {
	return w.st.Update(func(c *Config) { c.Cast.YouTube.OnDemand = v })
}

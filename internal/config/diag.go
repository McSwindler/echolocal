package config

import "github.com/ygelfand/echolocal/internal/defaults"

// Diag is how the device reports on itself.
type Diag struct {
	// Interval is how often the readings that drift are collected, in seconds.
	Interval int `json:"interval"`

	RemoteADB bool `json:"remote_adb"`

	// Control opens the control socket, which drives the running device from a terminal on it.
	Control bool `json:"control"`

	// InsecureTLS stops certificates being checked on anything the device downloads.
	InsecureTLS bool `json:"insecure_tls"`

	// MinCores holds cores online that the governor would otherwise park.
	MinCores int `json:"min_cores"`
}

// DefaultInterval is five minutes. The readings move slowly and every one of them costs a read of
// sysfs or proc, so this is a compromise between a stale card and a device busying itself for
// nobody.
const DefaultInterval = 300

// How many cores are worth holding online is a question about the cores a board has, so it comes
// from defaults.
func defaultDiag(d defaults.Set) Diag {
	return Diag{Interval: DefaultInterval, MinCores: d.MinCores}
}

type DiagWriter struct{ st *Store }

func (w DiagWriter) Interval(v int) error {
	return w.st.Update(func(c *Config) { c.Diag.Interval = v })
}

func (w DiagWriter) RemoteADB(v bool) error {
	return w.st.Update(func(c *Config) { c.Diag.RemoteADB = v })
}

func (w DiagWriter) Control(v bool) error {
	return w.st.Update(func(c *Config) { c.Diag.Control = v })
}

func (w DiagWriter) InsecureTLS(v bool) error {
	return w.st.Update(func(c *Config) { c.Diag.InsecureTLS = v })
}

func (w DiagWriter) MinCores(v int) error {
	return w.st.Update(func(c *Config) { c.Diag.MinCores = v })
}

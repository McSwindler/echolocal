package config

// API is what the device knows about Home Assistant.
type API struct {
	// Adopted is whether Home Assistant has ever subscribed. It only goes one way: a device that has
	// been added stays added through a restart, a move to another network, or Home Assistant being
	// down.
	Adopted bool `json:"adopted"`
}

type APIWriter struct{ st *Store }

func (w APIWriter) Adopted(v bool) error {
	return w.st.Update(func(c *Config) { c.API.Adopted = v })
}

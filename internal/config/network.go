package config

type Network struct {
	// Address is the IPv4 address last granted, which the next boot asks for again so the device
	// keeps the one anything pointed at it by address is using.
	Address string `json:"address,omitempty"`
}

type NetworkWriter struct{ st *Store }

func (w NetworkWriter) Address(v string) error {
	return w.st.Update(func(c *Config) { c.Network.Address = v })
}

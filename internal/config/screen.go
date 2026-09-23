package config

// Screen is the panel, on a board that has one.
type Screen struct {
	// Theme is the palette by name, from internal/ui/theme.
	Theme string `json:"theme"`
}

func defaultScreen() Screen { return Screen{Theme: DefaultTheme} }

// DefaultTheme is what a device nobody has chosen for shows. It is also where an unreadable
// settings file lands, so it is the one worth being a sensible sight rather than a statement.
const DefaultTheme = "Midnight"

type ScreenWriter struct{ st *Store }

func (w ScreenWriter) Theme(v string) error {
	return w.st.Update(func(c *Config) { c.Screen.Theme = v })
}

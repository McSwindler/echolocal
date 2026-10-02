package config

// Screen is the panel, on a board that has one.
type Screen struct {
	// Theme is the palette by name, from internal/ui/theme.
	Theme string `json:"theme"`

	// Drawer is the side of the picture the rail of icons is pulled in from.
	Drawer Edge `json:"drawer"`

	// Volume is the side the volume card comes up on, left or right.
	Volume Edge `json:"volume"`

	// Marks is whether a corner of the panel says the microphones are cut or the camera covered.
	Marks bool `json:"marks"`

	Hours HourFormat `json:"hours"`
	Logo  bool       `json:"logo"`

	// Backlight is the panel's level in manual, and what the automatic curve is shifted by in auto.
	Backlight int        `json:"backlight"`
	Mode      ScreenMode `json:"mode"`

	Style string `json:"style"`
	Size  string `json:"size"`
}

const (
	StyleStandard = "standard"

	UISizeLarge   = "large"
	UISizeCompact = "compact"
)

func ScreenStyles() []string { return []string{StyleStandard} }

func ScreenSizes() []string { return []string{UISizeLarge, UISizeCompact} }

func (w ScreenWriter) Style(v string) error {
	return w.st.Update(func(c *Config) { c.Screen.Style = v })
}

func (w ScreenWriter) Size(v string) error {
	return w.st.Update(func(c *Config) { c.Screen.Size = v })
}

const (
	DefaultBacklight  = 70
	DefaultScreenMode = ModeAuto
)

func defaultScreen() Screen {
	return Screen{
		Theme: DefaultTheme, Drawer: DefaultEdge, Volume: DefaultEdge, Marks: true, Hours: TwentyFourHour, Logo: true,
		Backlight: DefaultBacklight, Mode: DefaultScreenMode,
	}
}

// ScreenMode is how the brightness is decided.
type ScreenMode string

const (
	// ModeAuto sets the brightness from the ambient light sensor.
	ModeAuto ScreenMode = "auto"

	// ModeManual holds whatever the backlight is set to.
	ModeManual ScreenMode = "manual"
)

type HourFormat string

const (
	TwentyFourHour HourFormat = "24"
	TwelveHour     HourFormat = "12"
)

func (h HourFormat) Label() string {
	switch h {
	case TwentyFourHour:
		return say.T("hours.24")
	case TwelveHour:
		return say.T("hours.12")
	}
	return string(h)
}

func HourFormats() []HourFormat { return []HourFormat{TwentyFourHour, TwelveHour} }

// DefaultTheme is what a device nobody has chosen for shows. It is also where an unreadable
// settings file lands, so it is the one worth being a sensible sight rather than a statement.
const DefaultTheme = "Midnight"

type ScreenWriter struct{ st *Store }

// All writes the panel's settings together, for a caller holding a whole one it has already changed.
func (w ScreenWriter) All(v Screen) error {
	return w.st.Update(func(c *Config) { c.Screen = v })
}

func (w ScreenWriter) Theme(v string) error {
	return w.st.Update(func(c *Config) { c.Screen.Theme = v })
}

func (w ScreenWriter) Drawer(v Edge) error {
	return w.st.Update(func(c *Config) { c.Screen.Drawer = v })
}

func (w ScreenWriter) Volume(v Edge) error {
	return w.st.Update(func(c *Config) { c.Screen.Volume = v })
}

func (w ScreenWriter) Marks(v bool) error {
	return w.st.Update(func(c *Config) { c.Screen.Marks = v })
}

func (w ScreenWriter) Hours(v HourFormat) error {
	return w.st.Update(func(c *Config) { c.Screen.Hours = v })
}

func (w ScreenWriter) Logo(v bool) error {
	return w.st.Update(func(c *Config) { c.Screen.Logo = v })
}

func (w ScreenWriter) Backlight(v int) error {
	return w.st.Update(func(c *Config) { c.Screen.Backlight = v })
}

func (w ScreenWriter) Mode(v ScreenMode) error {
	return w.st.Update(func(c *Config) { c.Screen.Mode = v })
}

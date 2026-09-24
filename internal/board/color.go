package board

// The shell colours a board may report.
const (
	ColorBlack   = "black"
	ColorWhite   = "white"
	ColorUnknown = "unknown"
)

// Color is the device's shell as Home Assistant shows it. idme reads a factory identity field.
func (b Board) Color(idme func(string) string) string {
	if b.color == nil {
		return ColorUnknown
	}
	return b.color(idme)
}

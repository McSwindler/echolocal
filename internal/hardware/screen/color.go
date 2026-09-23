package screen

import "image/color"

// Color is one pixel.
type Color struct{ R, G, B, A byte }

// Opaque is a colour with no transparency, which is what the panel shows.
func Opaque(r, g, b byte) Color { return Color{R: r, G: g, B: b, A: 0xFF} }

var (
	Black = Opaque(0, 0, 0)
	White = Opaque(0xFF, 0xFF, 0xFF)
)

// From converts any image/color, which is what a decoded image hands over.
func From(c color.Color) Color {
	r, g, b, a := c.RGBA()
	return Color{R: byte(r >> 8), G: byte(g >> 8), B: byte(b >> 8), A: byte(a >> 8)}
}

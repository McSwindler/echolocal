package screen

// Orientation is how the picture sits on the panel, as a rotation from the panel's own portrait.
//
// Drawing works in viewed coordinates and the framebuffer is laid out in the panel's, so this is
// the one place the two are reconciled.
type Orientation int

const (
	// Rotate0 is the panel's native portrait, origin top-left.
	Rotate0 Orientation = 0

	Rotate90  Orientation = 90
	Rotate180 Orientation = 180
	Rotate270 Orientation = 270
)

func (o Orientation) String() string {
	switch o {
	case Rotate0:
		return "portrait"
	case Rotate90:
		return "landscape"
	case Rotate180:
		return "portrait inverted"
	case Rotate270:
		return "landscape inverted"
	}
	return "unknown"
}

// Sideways reports whether the rotation swaps width and height.
func (o Orientation) Sideways() bool { return o == Rotate90 || o == Rotate270 }

// Size is what a panel of this size looks like at this rotation.
func (o Orientation) Size(fbW, fbH int) (w, h int) {
	if o.Sideways() {
		return fbH, fbW
	}
	return fbW, fbH
}

// Panel maps a viewed pixel to the framebuffer pixel that carries it. x and y are in viewed
// coordinates, whose extent is Size(fbW, fbH).
func (o Orientation) Panel(x, y, fbW, fbH int) (px, py int) {
	switch o {
	case Rotate90:
		return y, fbH - 1 - x
	case Rotate180:
		return fbW - 1 - x, fbH - 1 - y
	case Rotate270:
		return fbW - 1 - y, x
	}
	return x, y
}

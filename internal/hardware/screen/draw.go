package screen

import "image"

// Bounds is the whole panel, in viewed coordinates.
func (p *Panel) Bounds() image.Rectangle { return image.Rect(0, 0, p.Width, p.Height) }

// Draw puts an image inside a box, as large as fits whole and centred, over bg.
//
// Nearest neighbour: the artwork is scaled once per draw and a filter would cost a multiply per
// pixel on four small cores for a difference nobody would see at this size.
func (p *Panel) Draw(img image.Image, box image.Rectangle, bg Color) {
	b := img.Bounds()
	if b.Empty() || box.Empty() {
		return
	}

	scale := min(float64(box.Dx())/float64(b.Dx()), float64(box.Dy())/float64(b.Dy()))
	w := int(float64(b.Dx()) * scale)
	h := int(float64(b.Dy()) * scale)
	if w == 0 || h == 0 {
		return
	}

	ox := box.Min.X + (box.Dx()-w)/2
	oy := box.Min.Y + (box.Dy()-h)/2

	for y := range h {
		sy := b.Min.Y + y*b.Dy()/h
		for x := range w {
			sx := b.Min.X + x*b.Dx()/w

			c := From(img.At(sx, sy))
			switch {
			case c.A == 0:
				continue
			case c.A < 0xFF:
				c = Over(c, bg)
			}
			p.Set(ox+x, oy+y, c)
		}
	}
}

// Blit puts an image on the panel at its own size, top-left at the given point.
func (p *Panel) Blit(img image.Image, at image.Point, bg Color) {
	b := img.Bounds()

	for y := range b.Dy() {
		for x := range b.Dx() {
			c := From(img.At(b.Min.X+x, b.Min.Y+y))
			switch {
			case c.A == 0:
				continue
			case c.A < 0xFF:
				c = Over(c, bg)
			}
			p.Set(at.X+x, at.Y+y, c)
		}
	}
}

// Over puts a partly transparent colour on a background. src is premultiplied, as image.RGBA stores
// it, so the source term is already scaled.
func Over(src, dst Color) Color {
	inv := 255 - int(src.A)

	return Color{
		R: byte(int(src.R) + int(dst.R)*inv/255),
		G: byte(int(src.G) + int(dst.G)*inv/255),
		B: byte(int(src.B) + int(dst.B)*inv/255),
		A: 0xFF,
	}
}

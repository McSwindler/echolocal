package screen

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

// Shot is what the panel is showing, in viewed orientation.
//
// It reads the page the controller is fetching rather than the one being drawn into, so a capture
// taken between a draw and a Flip shows what a person can see and not what is about to appear.
func (p *Panel) Shot() (*image.RGBA, error) {
	if p.mem == nil {
		return nil, fmt.Errorf("screen: the panel is closed")
	}

	page := p.page * p.pageBytes
	if page+p.pageBytes > len(p.mem) {
		return nil, fmt.Errorf("screen: page %d is outside the mapping", p.page)
	}

	img := image.NewRGBA(image.Rect(0, 0, p.Width, p.Height))
	for y := range p.Height {
		for x := range p.Width {
			px, py := p.rot.Panel(x, y, p.fbW, p.fbH)

			at := page + py*p.stride + px*4
			if at+4 > len(p.mem) {
				continue
			}

			// RGBA in memory, as Set writes it. Alpha is what the controller was given and says
			// nothing about what is visible, so a capture is opaque.
			out := img.PixOffset(x, y)
			img.Pix[out+0] = p.mem[at+0]
			img.Pix[out+1] = p.mem[at+1]
			img.Pix[out+2] = p.mem[at+2]
			img.Pix[out+3] = 0xFF
		}
	}
	return img, nil
}

// Save writes a capture to a PNG.
func (p *Panel) Save(path string) error {
	img, err := p.Shot()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("screen: encoding %s: %w", path, err)
	}
	return f.Close()
}

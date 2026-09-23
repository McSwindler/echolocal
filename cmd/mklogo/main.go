// Command mklogo scales the artwork down to what a panel can show.
//
// The originals are committed at whatever size they were drawn, and are far larger than any board's
// screen. What echod carries is the output of this, so a Show build embeds a couple of hundred
// kilobytes rather than a couple of megabytes.
//
//	mklogo -in assets/logo_dark.png -out internal/feature/splash/assets/logo_dark.png
package main

import (
	"fmt"
	"image"
	"image/png"
	_ "image/png"
	"math"
	"os"
)

func main() {
	var (
		in   = flagString("-in")
		out  = flagString("-out")
		w, h = 768, 512
	)
	if in == "" || out == "" {
		fmt.Fprintln(os.Stderr, "mklogo: -in and -out are required")
		os.Exit(1)
	}

	if err := run(in, out, w, h); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// flagString is a tiny reader for -name value, since this takes two of them and a flag set is more
// ceremony than that is worth.
func flagString(name string) string {
	for i, a := range os.Args {
		if a == name && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}

func run(in, out string, w, h int) error {
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("mklogo: decoding %s: %w", in, err)
	}

	dst := scale(src, w, h)

	o, err := os.Create(out)
	if err != nil {
		return err
	}
	defer o.Close()

	if err := png.Encode(o, dst); err != nil {
		return fmt.Errorf("mklogo: encoding %s: %w", out, err)
	}
	if err := o.Close(); err != nil {
		return err
	}

	fi, err := os.Stat(out)
	if err != nil {
		return err
	}
	fmt.Printf("%s -> %s  %dx%d, %d KB\n", in, out, dst.Bounds().Dx(), dst.Bounds().Dy(), fi.Size()/1024)
	return nil
}

// scale is a box filter: every output pixel is the average of the input pixels it covers. Right for
// making something smaller, which is the only direction this goes, and it needs nothing outside the
// standard library.
//
// Averaging happens in premultiplied alpha. Mixing straight colours would let the colour of a fully
// transparent pixel bleed into its neighbours, which is what puts a dark halo around artwork.
func scale(src image.Image, w, h int) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	sx := float64(b.Dx()) / float64(w)
	sy := float64(b.Dy()) / float64(h)

	for oy := range h {
		y0, y1 := int(float64(oy)*sy), int(float64(oy+1)*sy)
		if y1 <= y0 {
			y1 = y0 + 1
		}

		for ox := range w {
			x0, x1 := int(float64(ox)*sx), int(float64(ox+1)*sx)
			if x1 <= x0 {
				x1 = x0 + 1
			}

			var sr, sg, sb, sa, n float64
			for y := y0; y < y1 && y < b.Dy(); y++ {
				for x := x0; x < x1 && x < b.Dx(); x++ {
					// RGBA returns premultiplied, which is what should be averaged.
					r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
					sr += float64(r)
					sg += float64(g)
					sb += float64(bl)
					sa += float64(a)
					n++
				}
			}
			if n == 0 {
				continue
			}

			pa := sa / n
			at := dst.PixOffset(ox, oy)
			dst.Pix[at+3] = byte(math.Round(pa / 257))
			if pa == 0 {
				continue
			}

			// image.RGBA is premultiplied too, so the averages go in as they are.
			dst.Pix[at+0] = byte(math.Round(sr / n / 257))
			dst.Pix[at+1] = byte(math.Round(sg / n / 257))
			dst.Pix[at+2] = byte(math.Round(sb / n / 257))
		}
	}
	return dst
}

//go:build board_checkers || board_cronos || board_rook

package control

import (
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/hardware/gpu"
	"github.com/ygelfand/echolocal/internal/lib/analysis"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/visual"
)

const (
	DefaultThumbs = "/data/local/tmp/echolocal-thumbs"
	thumbFrames   = 50
	thumbStep     = 40 * time.Millisecond
	thumbWidth    = 384
)

func thumbsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "thumbs [dir]",
		Short: "Render the visual picker's thumbnails on this panel",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := DefaultThumbs
			if len(args) > 0 {
				dir = args[0]
			}
			n, err := thumbs(dir)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %d thumbnails to %s\n", n, dir)
			return nil
		},
	}
}

func thumbs(dir string) (int, error) {
	w, h := display.Get().Size()
	if w == 0 {
		return 0, errors.New("there is no panel to size the thumbnails by")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}

	n := 0
	for _, k := range visual.Built() {
		img, err := renderThumb(k, w, h)
		if err != nil {
			return n, fmt.Errorf("%s: %w", k, err)
		}
		f, err := os.Create(filepath.Join(dir, string(k)+".jpg"))
		if err != nil {
			return n, err
		}
		err = jpeg.Encode(f, shrink(img, thumbWidth, thumbWidth*h/w), &jpeg.Options{Quality: 82})
		f.Close()
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func renderThumb(k visual.Kind, w, h int) (*image.RGBA, error) {
	l, err := gpu.OpenOffscreen(w, h)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	v := visual.New(k)
	x := visual.Input{Mic: loud(), Speaker: loud(), Dt: thumbStep, Label: config.DefaultLabel}
	for i := range thumbFrames {
		x.Now = time.Duration(i+1) * thumbStep
		x.Replying = i >= thumbFrames/2
		if err := v.Shade(l, i == 0, ui.Rect{W: w, H: h}, x); err != nil {
			return nil, err
		}
		time.Sleep(thumbStep)
	}
	return l.Read()
}

func loud() analysis.Analysis {
	var a analysis.Analysis
	a.Level, a.Peak, a.Onsets = 0.4, 0.7, 3
	for b := range a.Bands {
		a.Bands[b] = float32(0.35 + 0.3*math.Sin(float64(b)*0.55) + 0.25*math.Cos(float64(b)*0.21))
	}
	for i := range a.Wave {
		p := 2 * math.Pi * float64(i) / float64(len(a.Wave))
		a.Wave[i] = float32(0.55*math.Sin(3*p) + 0.3*math.Sin(7*p+1) + 0.1*math.Sin(13*p))
	}
	return a
}

func shrink(src *image.RGBA, tw, th int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewRGBA(image.Rect(0, 0, tw, th))
	for y := range th {
		y0, y1 := y*h/th, (y+1)*h/th
		for x := range tw {
			x0, x1 := x*w/tw, (x+1)*w/tw
			var r, g, b, n int
			for sy := y0; sy < y1; sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := x0; sx < x1; sx++ {
					r, g, b, n = r+int(row[sx*4]), g+int(row[sx*4+1]), b+int(row[sx*4+2]), n+1
				}
			}
			o := out.PixOffset(x, y)
			out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = uint8(r/n), uint8(g/n), uint8(b/n), 255
		}
	}
	return out
}

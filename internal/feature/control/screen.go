//go:build board_checkers || board_cronos || board_rook

package control

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/hardware/touch"
)

// showing is what the panel can be asked.
func showing() []*cobra.Command {
	panel := &cobra.Command{
		Use:   "screen",
		Short: "The panel",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}

	shot := &cobra.Command{
		Use:   "shot <path>",
		Short: "Write what the panel is showing to a PNG",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := save(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), args[0])
			return nil
		},
	}

	tap := &cobra.Command{
		Use:   "tap <x> <y>",
		Short: "Touch the panel where a finger would",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			at, err := places(args)
			if err != nil {
				return err
			}

			touch.Get().Feed(touch.Contact{X: at[0], Y: at[1], Phase: touch.Down, At: time.Now()})
			time.Sleep(tapHeld)
			touch.Get().Feed(touch.Contact{X: at[0], Y: at[1], Phase: touch.Up, At: time.Now()})
			return nil
		},
	}

	swipe := &cobra.Command{
		Use:   "swipe <x> <y> <x> <y>",
		Short: "Drag a finger across the panel",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			at, err := places(args)
			if err != nil {
				return err
			}

			touch.Get().Feed(touch.Contact{X: at[0], Y: at[1], Phase: touch.Down, At: time.Now()})
			for i := 1; i <= swipeSteps; i++ {
				time.Sleep(swipeStep)
				touch.Get().Feed(touch.Contact{
					X:     at[0] + (at[2]-at[0])*i/swipeSteps,
					Y:     at[1] + (at[3]-at[1])*i/swipeSteps,
					Phase: touch.Move, At: time.Now(),
				})
			}
			touch.Get().Feed(touch.Contact{X: at[2], Y: at[3], Phase: touch.Up, At: time.Now()})
			return nil
		},
	}

	panel.AddCommand(shot, tap, swipe, thumbsCommand())
	return []*cobra.Command{panel, setCommand()}
}

const (
	// tapHeld is how long an injected touch stays down.
	tapHeld = 60 * time.Millisecond

	swipeSteps = 8
	swipeStep  = 16 * time.Millisecond
)

func save(path string) error {
	img, err := composed()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// composed is the panel as SurfaceFlinger composed it, turned the way it is viewed, or echod's own
// layer where there is no helper.
func composed() (image.Image, error) {
	c := display.Get().Helper()
	if c == nil {
		return drawn()
	}
	pix, nw, nh, err := c.ScreenRead()
	if err != nil {
		return nil, err
	}
	rot := display.Get().Orientation()
	w, h := rot.Size(nw, nh)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			px, py := rot.Project(nw, nh, x, y)
			copy(img.Pix[img.PixOffset(x, y):][:4], pix[(py*nw+px)*4:][:4])
		}
	}
	return img, nil
}

func drawn() (image.Image, error) {
	pixels, w, h, err := display.Get().Shot()
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range w * h {
		img.Set(i%w, i/w, color.RGBA{R: pixels[i*3], G: pixels[i*3+1], B: pixels[i*3+2], A: 0xff})
	}
	return img, nil
}

func places(args []string) ([]int, error) {
	out := make([]int, 0, len(args))
	for _, a := range args {
		v, err := strconv.Atoi(a)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

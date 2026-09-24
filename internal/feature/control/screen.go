package control

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/hardware/screen"
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
			p := screen.Get().Panel()
			if p == nil {
				return fmt.Errorf("there is no panel on this device")
			}
			if err := p.Save(args[0]); err != nil {
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

	panel.AddCommand(shot, tap, swipe)
	return []*cobra.Command{panel}
}

const (
	// tapHeld is how long an injected touch stays down.
	tapHeld = 60 * time.Millisecond

	swipeSteps = 8
	swipeStep  = 16 * time.Millisecond
)

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

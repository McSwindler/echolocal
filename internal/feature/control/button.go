package control

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/hardware/buttons"
)

const between = 120 * time.Millisecond

var pressable = map[string]buttons.Name{
	"up":     buttons.VolumeUp,
	"down":   buttons.VolumeDown,
	"mute":   buttons.Mute,
	"action": buttons.Action,
}

func pressing() *cobra.Command {
	return &cobra.Command{
		Use:   "button up|down|mute|action [times]",
		Short: "Press a button",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			name, ok := pressable[args[0]]
			if !ok {
				return fmt.Errorf("no button %q, want one of %s", args[0], strings.Join(pressableNames(), ", "))
			}
			times := 1
			if len(args) > 1 {
				n, err := strconv.Atoi(args[1])
				if err != nil || n < 1 {
					return fmt.Errorf("times must be a positive number")
				}
				times = n
			}
			for i := range times {
				if i > 0 {
					time.Sleep(between)
				}
				buttons.Get().Events.Emit(buttons.Event{Name: name, Kind: buttons.Tap})
			}
			return nil
		},
	}
}

func pressableNames() []string {
	return []string{"up", "down", "mute", "action"}
}

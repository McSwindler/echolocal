//go:build board_checkers || board_cronos || board_rook

package control

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	pages "github.com/ygelfand/echolocal/internal/feature/settings"
	screens "github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/hardware/touch"
)

const (
	readyWait  = 90 * time.Second
	readyEvery = 200 * time.Millisecond
)

func panelCommands() []*cobra.Command {
	open := &cobra.Command{
		Use:   "open <page>",
		Short: "Put a settings page up: " + strings.Join(pages.Pages(), ", "),
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if !pages.OpenPage(args[0]) {
				return fmt.Errorf("no page %q, want one of %s", args[0], strings.Join(pages.Pages(), ", "))
			}
			return nil
		},
	}

	stack := &cobra.Command{
		Use:   "stack",
		Short: "The screens up, top first, and which are held",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			views := screens.Get().Stack()
			if len(views) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "empty")
				return nil
			}
			for i := len(views) - 1; i >= 0; i-- {
				v := views[i]
				title := ""
				if p, ok := v.View.(*screens.Page); ok {
					title = fmt.Sprintf(" %q", p.Title)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%d %T%s held=%v covers=%v\n", i, v.View, title, v.Held, v.View.Covers())
			}
			return nil
		},
	}

	ready := &cobra.Command{
		Use:   "ready [milliseconds]",
		Short: "Wait until the boot screen lets go of the panel, and say what has it",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wait := readyWait
			if len(args) > 0 {
				ms, err := strconv.Atoi(args[0])
				if err != nil || ms < 0 {
					return fmt.Errorf("milliseconds must be a number")
				}
				wait = time.Duration(ms) * time.Millisecond
			}
			for deadline := time.Now().Add(wait); ; time.Sleep(readyEvery) {
				if at, showing := display.Get().Showing(); showing && at < display.PriorityBoot {
					fmt.Fprintln(cmd.OutOrStdout(), shown(at))
					return nil
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("still starting after %s", wait)
				}
			}
		},
	}

	hold := &cobra.Command{
		Use:   "hold <x> <y> <milliseconds>",
		Short: "Touch the panel and keep touching",
		Args:  cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := places(args)
			if err != nil {
				return err
			}
			if at[2] < 0 {
				return fmt.Errorf("milliseconds must not be negative")
			}
			touch.Get().Feed(touch.Contact{X: at[0], Y: at[1], Phase: touch.Down, At: time.Now()})
			time.Sleep(time.Duration(at[2]) * time.Millisecond)
			touch.Get().Feed(touch.Contact{X: at[0], Y: at[1], Phase: touch.Up, At: time.Now()})
			return nil
		},
	}

	return []*cobra.Command{open, stack, ready, hold}
}

func shown(at display.Priority) string {
	switch at {
	case display.PriorityDashboard:
		return "dashboard"
	case display.PrioritySetup:
		return "setup"
	case display.PriorityUI:
		return "screen"
	case display.PriorityNotice:
		return "notice"
	case display.PriorityAlert:
		return "alert"
	}
	return "boot"
}

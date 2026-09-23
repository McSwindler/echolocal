package control

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/hardware/metrics"
	"github.com/ygelfand/echolocal/internal/layout"
)

// What the harness can do, in one place.
//
// A command carries its own description, so there is one of them and it cannot go stale against a
// help string written out beside it.
//
// Built per invocation rather than once: commands hold parsed state, and these arrive down a socket
// from whoever is poking at the device, not from a process that starts and exits.
func (c *Control) tree() *cobra.Command {
	root := &cobra.Command{
		Use:   "ctl",
		Short: "Drive the running device",
		Long:  "Everything here acts on the device as it is, now.",

		SilenceUsage:  true,
		SilenceErrors: true,

		// A bare invocation is not an error, and neither is an empty line on the socket.
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}

	// Nothing down this socket has a shell to complete for.
	root.CompletionOptions.DisableDefaultCmd = true

	root.AddCommand(telling()...)
	root.AddCommand(showing()...)
	return root
}

// telling is what the device will say about itself.
func telling() []*cobra.Command {
	version := &cobra.Command{
		Use:   "version",
		Short: "What this build is",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			b := component.Board()
			fmt.Fprintf(cmd.OutOrStdout(), "version %s\ncommit  %s\nbuilt   %s\nboard   %s\n",
				layout.Version, layout.GitCommit, layout.BuildDate, b.Codename)
			return nil
		},
	}

	state := &cobra.Command{
		Use:   "state",
		Short: "What the device is doing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d := config.Get().Device
			up := time.Duration(metrics.Uptime() * float64(time.Second)).Round(time.Second)

			fmt.Fprintf(cmd.OutOrStdout(), "name    %s\naddr    %s\nmodel   %s\nuptime  %s\n",
				d.Name, d.Addr, d.Board.Model, up)
			return nil
		},
	}

	return []*cobra.Command{version, state}
}

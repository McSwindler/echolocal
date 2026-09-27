package echoctl

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/board"
)

func newBoardCmd() *cobra.Command {
	var (
		serial string
		sh     bool
	)

	c := &cobra.Command{
		Use:   "board",
		Short: "Say what board a device is, and what echod is installed as there",
		Long: "The Makefile's device targets read this rather than carrying their own table, so\n" +
			"there is one answer to which service to stop and which board to build for.\n\n" +
			"  echoctl board\n" +
			"  eval \"$(echoctl board --sh)\"",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := connect(cmd.Context(), cmd.OutOrStdout(), serial)
			if err != nil {
				return err
			}

			device, err := d.Getprop("ro.product.device")
			if err != nil {
				return err
			}

			// Strictly, unlike installer.BoardOf: what reads this goes on to remount /system and
			// write to it, so an unrecognised device has to stop rather than be assumed a biscuit.
			b, known := board.For(device)
			if !known {
				return fmt.Errorf("this device calls itself %q, which is not a board this build knows", device)
			}

			out := cmd.OutOrStdout()
			if sh {
				fmt.Fprintf(out, "codename=%s\ndevice=%s\nservice=%s\nbinary=%s\nlabel=%s\n",
					b.Codename, device, b.ServiceName, b.Service, b.StockLabel)
				return nil
			}

			fmt.Fprintf(out, "  %s %s\n", styleKey.Render("board"), b.Codename)
			fmt.Fprintf(out, "  %s %s\n", styleKey.Render("device"), device)
			fmt.Fprintf(out, "  %s %s\n", styleKey.Render("service"), b.ServiceName)
			fmt.Fprintf(out, "  %s %s\n", styleKey.Render("binary"), b.Service)
			fmt.Fprintf(out, "  %s %s\n", styleKey.Render("label"), b.StockLabel)
			return nil
		},
	}

	c.Flags().StringVar(&serial, "serial", "", "device to act on")
	c.Flags().BoolVar(&sh, "sh", false, "print shell assignments instead")
	return c
}

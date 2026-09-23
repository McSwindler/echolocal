package control

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/hardware/screen"
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

	panel.AddCommand(shot)
	return []*cobra.Command{panel}
}

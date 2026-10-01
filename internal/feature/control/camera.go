//go:build board_checkers || board_cronos

package control

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/livecam"
	"github.com/ygelfand/echolocal/internal/feature/vision"
)

const stillPath = "/data/local/tmp/still.jpg"

func init() {
	boardCommands = append(boardCommands, cameraCommand)
	boardSettings = append(boardSettings, cameraSettings)
}

func cameraCommand() *cobra.Command {
	camera := &cobra.Command{
		Use:   "camera",
		Short: "The camera",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	camera.AddCommand(&cobra.Command{
		Use:   "still [path]",
		Short: "Take the picture Home Assistant would get, to " + stillPath + " or path",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			at := stillPath
			if len(args) > 0 && args[0] != "" {
				at = args[0]
			}
			began := time.Now()
			pic, err := vision.Get().Still()
			if err != nil {
				return err
			}
			took := time.Since(began)
			if err := os.WriteFile(at, pic, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d bytes in %s to %s\n", len(pic), took.Round(time.Millisecond), at)
			return nil
		},
	})
	return camera
}

func cameraSettings() []setting {
	rows := livecam.Table().Rows()
	out := make([]setting, 0, len(rows))
	for _, s := range rows {
		out = append(out, setting{
			name: "camera." + s.Name,
			says: func(c config.Config) string {
				k, _ := livecam.Table().Configured(livecam.DefaultKnobs(), c.Camera.Settings)
				return s.Read(&k)
			},
			use: func(v string) error { return livecam.Set(s.Name, v) },
		})
	}
	return out
}

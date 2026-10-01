//go:build !(board_checkers || board_cronos || board_rook)

package control

import "github.com/spf13/cobra"

func showing() []*cobra.Command { return nil }

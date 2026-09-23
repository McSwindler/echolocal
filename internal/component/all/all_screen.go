//go:build board_checkers || board_cronos

package all

// The screen, its palette and the artwork it shows. Most of a megabyte of that is the artwork,
// which has no business in a build for a board with no panel.
import (
	_ "github.com/ygelfand/echolocal/internal/feature/splash"
	_ "github.com/ygelfand/echolocal/internal/feature/theme"
	_ "github.com/ygelfand/echolocal/internal/hardware/screen"
	_ "github.com/ygelfand/echolocal/internal/hardware/touch"
)

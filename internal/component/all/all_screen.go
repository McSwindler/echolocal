//go:build board_checkers || board_cronos || board_rook

package all

// The screen, its palette and the artwork it shows. Most of a megabyte of that is the artwork,
// which has no business in a build for a board with no panel.
import (
	_ "github.com/ygelfand/echolocal/internal/feature/assistant"
	_ "github.com/ygelfand/echolocal/internal/feature/brightness"
	_ "github.com/ygelfand/echolocal/internal/feature/dashboard"
	_ "github.com/ygelfand/echolocal/internal/feature/drawer"
	_ "github.com/ygelfand/echolocal/internal/feature/gui"
	_ "github.com/ygelfand/echolocal/internal/feature/idle"
	_ "github.com/ygelfand/echolocal/internal/feature/message"
	_ "github.com/ygelfand/echolocal/internal/feature/poster"
	_ "github.com/ygelfand/echolocal/internal/feature/privacy"
	_ "github.com/ygelfand/echolocal/internal/feature/screen"
	_ "github.com/ygelfand/echolocal/internal/feature/settings"
	_ "github.com/ygelfand/echolocal/internal/feature/shell"
	_ "github.com/ygelfand/echolocal/internal/feature/splash"
	_ "github.com/ygelfand/echolocal/internal/feature/visuals"
	_ "github.com/ygelfand/echolocal/internal/feature/volume"
	_ "github.com/ygelfand/echolocal/internal/feature/web"
	_ "github.com/ygelfand/echolocal/internal/hardware/display"
	_ "github.com/ygelfand/echolocal/internal/hardware/touch"
)

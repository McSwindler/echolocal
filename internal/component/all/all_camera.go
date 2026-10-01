//go:build board_checkers || board_cronos

package all

import (
	_ "github.com/ygelfand/echolocal/internal/feature/livecam"
	_ "github.com/ygelfand/echolocal/internal/feature/rtspd"
	_ "github.com/ygelfand/echolocal/internal/feature/vision"
)

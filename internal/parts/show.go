//go:build board_checkers || board_cronos

package parts

import (
	_ "embed"

	"github.com/ygelfand/echolocal/internal/layout"
)

//go:embed payload/echolocal-surface
var surface []byte

func init() {
	register(Part{Name: "SurfaceFlinger helper", Path: layout.Surface, Data: surface, Mode: 0o755, Service: layout.SurfaceService})
	register(InitRC)
}

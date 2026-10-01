//go:build board_checkers || board_cronos

package parts

import (
	_ "embed"

	"github.com/ygelfand/echolocal/internal/layout"
)

//go:embed payload/echolocal-surface
var surface []byte

//go:embed payload/libecholocal-camshim.so
var camshim []byte

//go:embed payload/echolocal-camera
var camera []byte

func init() {
	register(Part{Name: "SurfaceFlinger helper", Path: layout.Surface, Data: surface, Mode: 0o755, Service: layout.SurfaceService})
	register(Part{Name: "camera service preload", Path: layout.CamShim, Data: camshim, Mode: 0o644, Service: layout.CameraServerService})
	register(Part{Name: "camera helper", Path: layout.Camera, Data: camera, Mode: 0o755, Service: layout.CameraService})
	register(InitRC)
}

package assets

import (
	"embed"
	"fmt"
)

// What an install writes to the device, other than echod itself. These are committed rather than
// staged, so every build carries them: a couple of kilobytes together.
//
// The boot image lives in internal/host/bootimg, which publishes it as a release asset and fetches
// it at install time against a compiled-in hash. Nine megabytes per board is more than echoctl
// should carry.

//go:embed patch
var patches embed.FS

// Patch is the file a board's sysimg.Layout names, by its path under patch/.
func Patch(name string) ([]byte, error) {
	b, err := patches.ReadFile("patch/" + name)
	if err != nil {
		return nil, fmt.Errorf("assets: no patch %q", name)
	}
	return b, nil
}

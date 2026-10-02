package all

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

var screenOnly = []string{
	"github.com/ygelfand/echolocal/internal/lib/surface",
	"github.com/ygelfand/echolocal/internal/hardware/display",
	"github.com/ygelfand/echolocal/internal/hardware/gpu",
	"github.com/ygelfand/echolocal/internal/feature/visuals",
	"github.com/ygelfand/echolocal/internal/feature/idle",
	"github.com/ygelfand/echolocal/internal/ui/visual",
	"github.com/ygelfand/echolocal/internal/hardware/touch",
	"github.com/ygelfand/echolocal/internal/ui/reveal",
	"github.com/ygelfand/echolocal/internal/feature/brightness",
	"github.com/ygelfand/echolocal/internal/hardware/alsps",
	"github.com/ygelfand/echolocal/internal/feature/gui",
	"github.com/ygelfand/echolocal/internal/feature/screen",
	"github.com/ygelfand/echolocal/internal/feature/dashboard",
	"github.com/ygelfand/echolocal/internal/feature/web",
	"github.com/ygelfand/echolocal/internal/feature/poster",
	"github.com/ygelfand/echolocal/internal/feature/message",
	"github.com/ygelfand/echolocal/internal/feature/assistant",
	"github.com/ygelfand/echolocal/internal/feature/videoplayer",
	"github.com/ygelfand/echolocal/pkg/gogui",
	"github.com/go-gui-org/go-gui/gui",
}

var castOnly = []string{
	"github.com/ygelfand/echolocal/internal/feature/chromecast",
	"github.com/ygelfand/echolocal/internal/hardware/video",
	"github.com/ygelfand/echolocal/internal/lib/webm",
	"github.com/ygelfand/echolocal/internal/lib/hls",
	"github.com/ygelfand/echolocal/internal/lib/cenc",
	"github.com/ygelfand/echolocal/internal/lib/cast",
}

var cameraOnly = []string{
	"github.com/ygelfand/echolocal/internal/feature/livecam",
	"github.com/ygelfand/echolocal/internal/feature/rtspd",
	"github.com/ygelfand/echolocal/internal/feature/vision",
	"github.com/ygelfand/echolocal/internal/hardware/mtkcamera",
	"github.com/ygelfand/echolocal/internal/lib/rtsp",
	"github.com/ygelfand/echolocal/internal/lib/onvif",
}

func goList(t *testing.T, tags, pkg string, flags ...string) []string {
	t.Helper()
	gobin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go toolchain to list the build with")
	}
	args := append([]string{"list", "-e"}, flags...)
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	cmd := exec.Command(gobin, append(args, pkg)...)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=arm", "CGO_ENABLED=0")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %s: %v", tags, err)
	}
	return strings.Fields(string(out))
}

func echodDeps(t *testing.T, tags string) []string {
	return goList(t, tags, "github.com/ygelfand/echolocal/cmd/echod", "-deps")
}

func TestOnlyHelperBoardsEmbedTheHelper(t *testing.T) {
	for tags, want := range map[string]bool{
		"":               false,
		"board_rook":     false,
		"board_checkers": true,
		"board_cronos":   true,
	} {
		files := goList(t, tags, "github.com/ygelfand/echolocal/internal/parts", "-f", `{{join .EmbedPatterns " "}}`)
		for _, part := range []string{"payload/echolocal-surface", "payload/echolocal-camera", "payload/libecholocal-camshim.so"} {
			if got := slices.Contains(files, part); got != want {
				t.Errorf("%q embeds %s %t, want %t (%v)", tags, part, got, want, files)
			}
		}
	}
}

func TestTheSharedBuildLinksNoScreen(t *testing.T) {
	deps := echodDeps(t, "")
	for _, pkg := range slices.Concat(screenOnly, castOnly, cameraOnly) {
		if slices.Contains(deps, pkg) {
			t.Errorf("the shared echod links %s", pkg)
		}
	}
}

func TestOnlyTheShowsCast(t *testing.T) {
	for tag, want := range map[string]bool{
		"board_checkers": true,
		"board_cronos":   true,
		"board_rook":     false,
	} {
		deps := echodDeps(t, tag)
		for _, pkg := range slices.Concat(castOnly, cameraOnly) {
			if got := slices.Contains(deps, pkg); got != want {
				t.Errorf("%s links %s %t, want %t", tag, pkg, got, want)
			}
		}
	}
}

func TestScreenBoardsLinkTheScreen(t *testing.T) {
	for _, tag := range []string{"board_checkers", "board_cronos", "board_rook"} {
		deps := echodDeps(t, tag)
		for _, pkg := range screenOnly {
			if !slices.Contains(deps, pkg) {
				t.Errorf("%s does not link %s", tag, pkg)
			}
		}
	}
}

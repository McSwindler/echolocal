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
}

var castOnly = []string{
	"github.com/ygelfand/echolocal/internal/feature/chromecast",
	"github.com/ygelfand/echolocal/internal/hardware/video",
	"github.com/ygelfand/echolocal/internal/feature/videoplayer",
	"github.com/ygelfand/echolocal/internal/lib/webm",
	"github.com/ygelfand/echolocal/internal/lib/hls",
	"github.com/ygelfand/echolocal/internal/lib/cenc",
	"github.com/ygelfand/echolocal/internal/lib/cast",
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
		if got := slices.Contains(files, "payload/echolocal-surface"); got != want {
			t.Errorf("%q embeds the helper %t, want %t (%v)", tags, got, want, files)
		}
	}
}

func TestTheSharedBuildLinksNoScreen(t *testing.T) {
	deps := echodDeps(t, "")
	for _, pkg := range append(screenOnly, castOnly...) {
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
		for _, pkg := range castOnly {
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

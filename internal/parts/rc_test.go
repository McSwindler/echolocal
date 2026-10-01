package parts

import (
	"slices"
	"strings"
	"testing"

	"github.com/ygelfand/echolocal/internal/layout"
)

func rcLines() [][]string {
	var lines [][]string
	for line := range strings.SplitSeq(string(InitRC.Data), "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			lines = append(lines, f)
		}
	}
	return lines
}

// Android 7's init rejects a service name longer than 16 characters.
func TestServiceNamesFitAndroid7(t *testing.T) {
	var names []string
	for _, f := range rcLines() {
		if f[0] == "service" && len(f) > 1 {
			names = append(names, f[1])
			if len(f[1]) > 16 {
				t.Errorf("service %q is %d characters, init takes 16", f[1], len(f[1]))
			}
		}
	}
	for _, want := range []string{layout.Service, layout.SurfaceService, layout.CameraService} {
		if !slices.Contains(names, want) {
			t.Errorf("no service %q in %v", want, names)
		}
	}
}

// Android 7's init reads /system/etc/init during mount_all, after these triggers have run.
func TestNoTriggerFiresBeforeSystemIsRead(t *testing.T) {
	for _, f := range rcLines() {
		if f[0] == "on" && len(f) > 1 && slices.Contains([]string{"early-init", "init", "late-init", "early-fs", "fs"}, f[1]) {
			t.Errorf("on %s never runs from /system/etc/init", f[1])
		}
	}
}

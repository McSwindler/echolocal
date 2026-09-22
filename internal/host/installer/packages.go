package installer

import (
	"fmt"
	"strings"
)

// hidePackages hides the Amazon packages the board names and stops the ones running.
//
// A hide persists, so this is install-time work rather than something echod redoes. `pm list
// packages` omits what is already hidden, which is what makes a second run find nothing to do.
func hidePackages(r *run) (string, bool, error) {
	want := r.cfg.Profile.Hide
	if len(want) == 0 {
		return "no packages to hide on this board", true, nil
	}

	out, err := r.d.Shell("pm list packages")
	if err != nil {
		return "", false, err
	}
	visible := make(map[string]bool)
	for _, line := range strings.Fields(out) {
		visible[strings.TrimPrefix(line, "package:")] = true
	}

	var todo []string
	for _, p := range want {
		if visible[p.Name] {
			todo = append(todo, p.Name)
		}
	}
	if len(todo) == 0 {
		return fmt.Sprintf("all %d already hidden", len(want)), true, nil
	}

	// Both in one invocation each. A shell per package is two adb round trips apiece, which on a list
	// this long is most of a minute of doing nothing.
	var hide, stop strings.Builder
	for _, name := range todo {
		fmt.Fprintf(&hide, "pm hide %s;", name)
		fmt.Fprintf(&stop, "am force-stop %s;", name)
	}

	// pm reports the state it ended in rather than failing, so the answer is the thing to read.
	said, err := r.d.Shell(hide.String())
	if err != nil {
		return "", false, err
	}
	for _, name := range todo {
		if !strings.Contains(said, name+" new hidden state: true") {
			return "", false, fmt.Errorf("hiding %s: %s", name, strings.TrimSpace(said))
		}
	}

	// Hiding blocks the next launch and leaves a running process alone, and a live audio client keeps
	// mediaserver holding the PCM devices.
	if _, err := r.d.Shell(stop.String()); err != nil {
		return "", false, err
	}

	// A package marked persistent is restarted by system_server whatever the force-stop does, so a
	// boot is what settles this.
	r.reboot = true
	return fmt.Sprintf("%d of %d hidden", len(todo), len(want)), false, nil
}

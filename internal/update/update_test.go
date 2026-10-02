package update

import (
	"os"
	"path/filepath"
	"testing"
)

// somewhere writable points the paths at a temp directory, since the real ones are under a read-only
// /system that only exists on the device.
func somewhere(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	t.Cleanup(func() {
		prev, old, mount, writable, reboot = layoutPrev, layoutOld, layoutMount, remount, layoutReboot
		binary, aside, systemOld, systemPrev = layoutBinary, layoutAside, layoutSystemOld, layoutSystemPrev
	})
	data, system := filepath.Join(dir, "data"), filepath.Join(dir, "system")
	for _, d := range []string{data, system} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	prev, old, mount = filepath.Join(data, "echod.prev"), filepath.Join(data, "echod.old"), system
	binary, aside = filepath.Join(system, "echod"), filepath.Join(system, "echod.aside")
	systemOld, systemPrev = filepath.Join(system, "echod.old"), filepath.Join(system, "echod.prev")

	// The directory is already writable, so the remount is the one thing that cannot be exercised here.
	writable = func(bool) error { return nil }

	// Android is not here, so nothing asks init for a reboot unless a test means to.
	reboot = func() {}
	return data
}

var (
	layoutPrev   = prev
	layoutOld    = old
	layoutMount  = mount
	layoutReboot = reboot

	layoutBinary     = binary
	layoutAside      = aside
	layoutSystemOld  = systemOld
	layoutSystemPrev = systemPrev
)

func TestASystemPrevIsATrialToo(t *testing.T) {
	somewhere(t)

	if err := os.WriteFile(systemPrev, []byte("older"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !OnTrial() {
		t.Error("not on trial with a previous binary on /system")
	}
}

func TestCommitMovesASystemPrevToData(t *testing.T) {
	somewhere(t)

	for path, body := range map[string]string{systemPrev: "older", systemOld: "oldest", aside: "stale"} {
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	Commit()

	if OnTrial() {
		t.Error("still on trial after committing")
	}
	if got, err := os.ReadFile(old); err != nil || string(got) != "older" {
		t.Errorf("old holds %q, %v; want the previous binary", got, err)
	}
	for _, p := range []string{systemPrev, systemOld, aside} {
		if exists(p) {
			t.Errorf("%s is still on /system", filepath.Base(p))
		}
	}
}

func TestSwapKeepsThePreviousOnData(t *testing.T) {
	dir := somewhere(t)

	staged := filepath.Join(dir, "incoming")
	if err := os.WriteFile(binary, []byte("running"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staged, []byte("newer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := swap(staged, "1.2.3"); err != nil {
		t.Fatal(err)
	}

	if got, _ := os.ReadFile(binary); string(got) != "newer" {
		t.Errorf("binary holds %q, want the new one", got)
	}
	if got, _ := os.ReadFile(prev); string(got) != "running" {
		t.Errorf("prev holds %q, want the one it replaced", got)
	}
	entries, err := os.ReadDir(filepath.Dir(binary))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("/system holds %d files, want only the binary", len(entries))
	}
}

func TestOnTrialIsThePresenceOfPrev(t *testing.T) {
	somewhere(t)

	if OnTrial() {
		t.Error("on trial with no previous binary")
	}
	if err := os.WriteFile(prev, []byte("older"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !OnTrial() {
		t.Error("not on trial with a previous binary sitting there")
	}
}

// Committing files the previous binary one generation back, which is what stops a later boot from
// treating it as a trial that never finished. Remounting fails off the device, so what is asserted here
// is the rename, on a directory that is already writable.
func TestCommitFilesThePreviousBinaryAway(t *testing.T) {
	somewhere(t)

	if err := os.WriteFile(prev, []byte("older"), 0o755); err != nil {
		t.Fatal(err)
	}
	Commit()

	if OnTrial() {
		t.Error("still on trial after committing")
	}
	if _, err := os.Stat(old); err != nil {
		t.Errorf("the previous binary was not kept: %v", err)
	}
}

// Nothing to keep is the ordinary case — most starts follow no update at all — and it must not touch
// the filesystem or report anything.
func TestCommitWithNothingOnTrial(t *testing.T) {
	dir := somewhere(t)

	Commit()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("committing nothing left %d files behind", len(entries))
	}
}

func TestStartWithNoUpdateSaysSo(t *testing.T) {
	somewhere(t)

	onTrial, rebooting := Start()
	if onTrial || rebooting {
		t.Errorf("no previous binary reported trial=%t rebooting=%t", onTrial, rebooting)
	}
}

// Two requests are one restart: the channel is buffered at one and a full one is dropped, so a caller
// that asks twice cannot leave a second request behind to be acted on later.
func TestRestartCoalesces(t *testing.T) {
	Restart("first")
	Restart("second")

	if got := <-Wanted(); got != "first" {
		t.Errorf("wanted %q, want first", got)
	}
	select {
	case got := <-Wanted():
		t.Errorf("a second request was queued: %q", got)
	default:
	}
}

// Reboot is the ask, not the doing: what happens after the property is set is init's, so the test
// proves the ask gets made once.
func TestRebootAsksInit(t *testing.T) {
	somewhere(t)

	asked := make(chan string, 1)
	reboot = func() { asked <- "asked" }

	Reboot("testing")

	if got := <-asked; got != "asked" {
		t.Errorf("reboot sent %q, want the ask to init", got)
	}
}

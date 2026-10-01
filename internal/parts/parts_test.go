package parts

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type fake struct {
	mounts    []bool
	restarted []string
}

func stub(t *testing.T) *fake {
	t.Helper()
	f := &fake{}
	w, r, l := writable, restart, label
	writable = func(rw bool) error { f.mounts = append(f.mounts, rw); return nil }
	restart = func(s string) error { f.restarted = append(f.restarted, s); return nil }
	label = func(string) error { return nil }
	t.Cleanup(func() { writable, restart, label = w, r, l })
	return f
}

func TestEnsureReplacesWhatDiffersAndRestartsItsService(t *testing.T) {
	f := stub(t)
	dir := t.TempDir()
	helper := filepath.Join(dir, "helper")
	rc := filepath.Join(dir, "echolocal.rc")
	same := filepath.Join(dir, "same")

	if err := os.WriteFile(helper, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(same, []byte("kept"), 0o755); err != nil {
		t.Fatal(err)
	}

	pending := ensure([]Part{
		{Name: "helper", Path: helper, Data: []byte("new"), Mode: 0o755, Service: "helper"},
		{Name: "service", Path: rc, Data: []byte("rc"), Mode: 0o644, Boot: true},
		{Name: "same", Path: same, Data: []byte("kept"), Mode: 0o755, Service: "same"},
	})

	if !pending {
		t.Error("a changed boot part did not ask for a reboot")
	}
	for path, want := range map[string]string{helper: "new", rc: "rc", same: "kept"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("%s holds %q (%v), want %q", filepath.Base(path), got, err, want)
		}
	}
	if fi, err := os.Stat(helper); err != nil || fi.Mode().Perm() != 0o755 {
		t.Errorf("helper mode %v (%v), want 0755", fi.Mode().Perm(), err)
	}
	if !slices.Equal(f.restarted, []string{"helper"}) {
		t.Errorf("restarted %v, want only helper", f.restarted)
	}
	if !slices.Equal(f.mounts, []bool{true, false}) {
		t.Errorf("remounts %v, want writable then read-only", f.mounts)
	}
	if _, err := os.Stat(helper + ".new"); !os.IsNotExist(err) {
		t.Errorf("left %s.new behind", filepath.Base(helper))
	}
}

func TestEnsureLeavesSystemAloneWhenNothingDiffers(t *testing.T) {
	f := stub(t)
	path := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(path, []byte("same"), 0o755); err != nil {
		t.Fatal(err)
	}

	if ensure([]Part{{Name: "service", Path: path, Data: []byte("same"), Mode: 0o644, Boot: true}}) {
		t.Error("an unchanged boot part asked for a reboot")
	}
	if len(f.mounts) != 0 || len(f.restarted) != 0 {
		t.Errorf("remounted %v and restarted %v with nothing to change", f.mounts, f.restarted)
	}
}

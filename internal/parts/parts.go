// Package parts is what echod needs on /system besides itself. Each one is carried inside echod and
// put in place when it starts, so an update or a rollback brings its own.
package parts

import (
	"bytes"
	_ "embed"
	"log/slog"
	"os"

	"github.com/ygelfand/echolocal/internal/android/prop"
	"github.com/ygelfand/echolocal/internal/layout"
	"github.com/ygelfand/echolocal/internal/update"
)

//go:embed echolocal.rc
var initRC []byte

// Part is one file echod keeps current, and the init service that runs it. Boot is a part init only
// reads at boot.
type Part struct {
	Name    string
	Path    string
	Data    []byte
	Mode    os.FileMode
	Service string
	Boot    bool
}

// InitRC is echod's own service definition, for a board that has one rather than taking over Amazon's.
var InitRC = Part{Name: "service", Path: layout.InitRC, Data: initRC, Mode: 0o644, Boot: true}

var registered []Part

func register(p Part) { registered = append(registered, p) }

// All is what this build carries.
func All() []Part { return registered }

var (
	writable = update.Writable
	restart  = prop.Restart
	label    = setLabel
)

// Ensure rewrites whichever parts differ from the ones this build carries, restarts their services,
// and reports whether a part init only reads at boot changed. Nothing is written when nothing differs.
func Ensure() (rebootPending bool) { return ensure(registered) }

func ensure(ps []Part) (rebootPending bool) {
	var stale []Part
	for _, p := range ps {
		if have, err := os.ReadFile(p.Path); err == nil && bytes.Equal(have, p.Data) {
			continue
		}
		stale = append(stale, p)
	}
	if len(stale) == 0 {
		return false
	}

	if err := writable(true); err != nil {
		slog.Error("remounting to update parts failed", "err", err)
		return false
	}
	defer func() {
		if err := writable(false); err != nil {
			slog.Error("remounting read-only failed", "err", err)
		}
	}()

	for _, p := range stale {
		if err := replace(p); err != nil {
			slog.Error("updating a part failed", "part", p.Name, "path", p.Path, "err", err)
			continue
		}
		slog.Info("part updated", "part", p.Name, "path", p.Path)

		if p.Service != "" {
			if err := restart(p.Service); err != nil {
				slog.Error("restarting a part failed", "part", p.Name, "service", p.Service, "err", err)
			}
		}
		if p.Boot {
			slog.Warn("init reads this at the next boot", "part", p.Name)
			rebootPending = true
		}
	}
	return rebootPending
}

func replace(p Part) error {
	tmp := p.Path + ".new"
	if err := os.WriteFile(tmp, p.Data, p.Mode); err != nil {
		return err
	}
	if err := os.Chmod(tmp, p.Mode); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := label(tmp); err != nil {
		slog.Warn("labelling a part failed", "path", tmp, "err", err)
	}
	if err := os.Rename(tmp, p.Path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

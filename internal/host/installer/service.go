package installer

import (
	"bytes"

	"github.com/ygelfand/echolocal/internal/layout"
	"github.com/ygelfand/echolocal/internal/parts"
)

// ownService reports whether the board runs echod as a service of its own.
func (r *run) ownService() bool { return r.board().ServiceName == layout.Service }

// installService writes echod's service definition. echod puts its other parts in place itself.
func installService(r *run) (string, bool, error) {
	if !r.ownService() {
		return "the board takes over one of Amazon's", true, nil
	}

	rc := parts.InitRC
	existing, _ := r.d.ReadFile(rc.Path)
	if bytes.Equal(bytes.TrimSpace(existing), bytes.TrimSpace(rc.Data)) {
		return rc.Name + " already installed", true, nil
	}
	if err := r.d.WriteFile(rc.Path, rc.Data, rc.Mode); err != nil {
		return "", false, err
	}
	if _, err := r.d.Shell("restorecon " + layout.Binary + " " + rc.Path); err != nil {
		return "", false, err
	}
	r.reboot = true
	return rc.Path, false, nil
}

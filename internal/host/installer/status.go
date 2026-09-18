package installer

import (
	"strconv"
	"strings"
	"time"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/device"
	"github.com/ygelfand/echolocal/internal/layout"
)

// State is what echod's installation looks like on a device.
type State struct {
	Serial  string
	Model   string
	Product string
	SDK     string
	Uptime  float64

	// Name is what Home Assistant calls the device, and Provisioned whether it has a key.
	Name        string
	Provisioned bool

	// Board is what the device says it is, and Service the init service echod is installed as there.
	// Reported so that whatever displays this does not have to work out the board a second time.
	Board   board.Board
	Service string
	Backup  string

	Installed  bool
	LinkTarget string
	HaveBackup bool
	Version    string

	ServiceState string
	AgentState   string
	StartedAt    string
}

// RunningFor is how long the current echod process has been up, from the uptime it recorded at
// start against the device's uptime now.
func (s State) RunningFor() (time.Duration, bool) {
	started, err := strconv.ParseFloat(s.StartedAt, 64)
	if err != nil || started <= 0 || s.Uptime < started {
		return 0, false
	}
	return time.Duration((s.Uptime - started) * float64(time.Second)), true
}

// ReadState reads installation and runtime state without changing anything.
func ReadState(d *device.Device) (State, error) {
	s := State{Serial: d.Serial()}

	s.Model, _ = d.Getprop("ro.product.model")
	s.Product, _ = d.Getprop("ro.product.device")
	s.SDK, _ = d.Getprop("ro.build.version.sdk")
	s.Uptime, _ = d.Uptime()

	// The device already said what it is, so the board comes from that rather than another getprop.
	s.Board = board.Biscuit
	if b, ok := board.For(s.Product); ok {
		s.Board = b
	}
	s.Service, s.Backup = s.Board.Service, s.Board.Backup()

	link, err := d.IsSymlink(s.Service)
	if err != nil {
		return s, err
	}
	if link {
		target, err := d.Shell("readlink " + s.Service)
		if err != nil {
			return s, err
		}
		s.LinkTarget = strings.TrimSpace(target)
		s.Installed = s.LinkTarget == layout.Binary
	}

	if s.HaveBackup, err = d.Exists(s.Backup); err != nil {
		return s, err
	}

	if s.Installed {
		// A binary that will not execute is what a file listing hides.
		if out, err := d.Shell(layout.Binary + " --version"); err == nil {
			s.Version = strings.TrimSpace(out)
		} else {
			s.Version = "will not run: " + err.Error()
		}
	}

	if s.Name, err = ReadName(d); err != nil {
		return s, err
	}
	key, err := ReadKey(d)
	if err != nil {
		return s, err
	}
	s.Provisioned = key != ""

	s.ServiceState, _ = d.Getprop("init.svc." + s.Board.ServiceName)
	s.AgentState, _ = d.Getprop(layout.StateProp)
	s.StartedAt, _ = d.Getprop(layout.StartedProp)
	return s, nil
}

// Package control is a way to drive the device without touching it.
//
// A line-oriented socket that acts on the device as it is running: a wake word fired, the ring
// painted, the screen asked what it is showing. All of it from a terminal on the device, against
// the live process rather than a second one that would have to take the hardware first.
//
// Local only: an abstract unix socket, which has no filesystem presence and cannot be reached off
// the device.
package control

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/service"
)

func init() {
	component.Register(component.Device, Get, component.Order(90),
		component.Supervise(service.Restart(time.Second, time.Minute)))
}

// Socket is the abstract address. The leading NUL is what makes it abstract.
const Socket = "@echod"

type Control struct {
	// gate guards the listener, which Run and the component's own shutdown both reach.
	gate     sync.Mutex
	listener net.Listener

	// addr is where it listens. Socket everywhere real; a test points it at a path of its own,
	// because the name is fixed and two tests binding it at once is a race with the device.
	addr string
}

var (
	once   sync.Once
	shared *Control
)

func Get() *Control { once.Do(func() { shared = build() }); return shared }

func build() *Control { return &Control{addr: Socket} }

func (c *Control) Name() string { return "control" }

// Run holds the socket for the life of the process.
func (c *Control) Run(ctx context.Context) error {
	if err := c.up(); err != nil {
		return err
	}

	<-ctx.Done()
	c.down()
	return nil
}

// up opens the socket, if it is not open already.
func (c *Control) up() error {
	c.gate.Lock()
	defer c.gate.Unlock()

	if c.listener != nil {
		return nil
	}

	l, err := net.Listen("unix", c.addr)
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	c.listener = l

	// Loudly, because it is a way in. Anything on the device that can open an abstract socket can
	// drive it, with no pairing and no record of who did.
	slog.Warn("the control socket is open and unauthenticated", "address", c.addr)

	go c.accept(l)
	return nil
}

// down closes it, so nothing is left holding the address.
func (c *Control) down() {
	c.gate.Lock()
	defer c.gate.Unlock()

	if c.listener == nil {
		return
	}
	c.listener.Close()
	c.listener = nil

	slog.Info("the control socket is closed", "address", c.addr)
}

// accept takes callers until the listener is closed, which is the only way it ends.
func (c *Control) accept(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go c.serve(conn)
	}
}

func (c *Control) serve(conn net.Conn) {
	defer conn.Close()

	in := bufio.NewScanner(conn)
	for in.Scan() {
		out, err := c.run(strings.Fields(in.Text()))

		// Before the error, not instead of it. A command that got several steps in before failing
		// has already said what worked, and that is the part worth reading.
		if out != "" {
			fmt.Fprintln(conn, strings.TrimRight(out, "\n"))
		}
		if err != nil {
			fmt.Fprintf(conn, "error: %v\n", err)
			continue
		}
		fmt.Fprintln(conn, "ok")
	}
}

// run does one command, through the tree in tree.go.
//
// Output is collected rather than printed, because the caller is a socket and not a terminal: what
// the command wrote and what it failed with go back over the same connection in the right order.
func (c *Control) run(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}

	var out strings.Builder

	root := c.tree()
	root.SetArgs(args)
	root.SetOut(&out)
	root.SetErr(&out)

	err := root.Execute()
	return out.String(), err
}

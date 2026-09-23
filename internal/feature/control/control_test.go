package control

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sock is a path short enough to bind. sockaddr_un holds 104 bytes on darwin, and t.TempDir builds
// its path out of the test's name, which is enough to overrun it.
func sock(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "c")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "s")
}

// dial brings a socket up on a path of its own and connects to it. The real address is a fixed
// name, and two tests binding it at once is a race with the device.
func dial(t *testing.T) net.Conn {
	t.Helper()

	c := &Control{addr: sock(t)}
	if err := c.up(); err != nil {
		t.Fatalf("up: %v", err)
	}
	t.Cleanup(c.down)

	conn, err := net.DialTimeout("unix", c.addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	return conn
}

// say sends one command and reads back to the line that ends it.
func say(t *testing.T, conn net.Conn, in *bufio.Scanner, line string) (string, bool) {
	t.Helper()

	if _, err := fmt.Fprintln(conn, line); err != nil {
		t.Fatalf("send %q: %v", line, err)
	}

	var out strings.Builder
	for in.Scan() {
		got := in.Text()
		if got == "ok" {
			return out.String(), true
		}
		if strings.HasPrefix(got, "error: ") {
			return got, false
		}
		out.WriteString(got)
		out.WriteByte('\n')
	}
	t.Fatalf("the connection closed before %q was answered", line)
	return "", false
}

func TestACommandIsAnsweredAndTerminated(t *testing.T) {
	conn := dial(t)
	in := bufio.NewScanner(conn)

	out, ok := say(t, conn, in, "version")
	if !ok {
		t.Fatalf("version failed: %s", out)
	}
	if !strings.Contains(out, "board") {
		t.Errorf("version said %q, want it to name the board", out)
	}
}

// Every reply ends in exactly one of ok or error:, or a caller reading a sequence hangs waiting for
// a command that already finished.
func TestAnUnknownCommandIsAnError(t *testing.T) {
	conn := dial(t)
	in := bufio.NewScanner(conn)

	out, ok := say(t, conn, in, "nonesuch")
	if ok {
		t.Fatalf("an unknown command reported success: %s", out)
	}
	if !strings.HasPrefix(out, "error: ") {
		t.Errorf("said %q, want an error: line", out)
	}
}

// A blank line is what a sequence file full of comments sends, and it must not be an error or the
// rest of the sequence stops.
func TestABlankLineIsAnswered(t *testing.T) {
	conn := dial(t)
	in := bufio.NewScanner(conn)

	if _, ok := say(t, conn, in, ""); !ok {
		t.Error("a blank line was reported as a failure")
	}
}

// One connection takes commands until the caller goes away, which is what lets a sequence run in
// order without reconnecting between each.
func TestOneConnectionTakesSeveralCommands(t *testing.T) {
	conn := dial(t)
	in := bufio.NewScanner(conn)

	for _, cmd := range []string{"version", "state", "version"} {
		if out, ok := say(t, conn, in, cmd); !ok {
			t.Fatalf("%s failed: %s", cmd, out)
		}
	}
}

// down has to release the address, or nothing can open it again without a restart.
func TestDownReleasesTheAddress(t *testing.T) {
	c := &Control{addr: sock(t)}
	if err := c.up(); err != nil {
		t.Fatalf("up: %v", err)
	}
	c.down()

	if err := c.up(); err != nil {
		t.Fatalf("reopening after down: %v", err)
	}
	c.down()
}

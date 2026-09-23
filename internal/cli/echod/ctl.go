package echod

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/feature/control"
)

// sequence is the commands to send: the arguments as one command, several separated by semicolons,
// or a command to a line from stdin when the only argument is -.
func sequence(stdin io.Reader, args []string) ([]string, error) {
	if len(args) == 1 && args[0] == "-" {
		var out []string
		in := bufio.NewScanner(stdin)
		for in.Scan() {
			// Blank lines and comments, so a sequence in a file can be read a month later.
			line := strings.TrimSpace(in.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			out = append(out, line)
		}
		return out, in.Err()
	}

	var out []string
	for _, part := range strings.Split(strings.Join(args, " "), ";") {
		if line := strings.Join(strings.Fields(part), " "); line != "" {
			out = append(out, line)
		}
	}
	return out, nil
}

// answer reads one command's output, up to the ok that ends it.
func answer(in *bufio.Scanner) error {
	for in.Scan() {
		line := in.Text()
		if line == "ok" {
			return nil
		}
		fmt.Println(line)

		if after, found := strings.CutPrefix(line, "error: "); found {
			return fmt.Errorf("%s", after)
		}
	}
	if err := in.Err(); err != nil {
		return err
	}
	return fmt.Errorf("the daemon closed the connection")
}

func newCtlCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ctl [command...]",
		Short: "Drive a running echod",
		Long: "Sends commands to the daemon's control socket and prints what comes back.\n" +
			"It acts on the process that holds the hardware, so nothing has to be stopped\n" +
			"first the way the offline tools need.\n\n" +
			"Several commands separated by ; run in order on one connection, and stop at\n" +
			"the first one that fails. A single - reads them from stdin instead, a command\n" +
			"to a line, which is how a longer sequence is kept in a file.\n\n" +
			"  echod ctl version\n" +
			"  echod ctl 'state ; version'\n" +
			"  echod ctl - < sequence.txt",
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, err := net.DialTimeout("unix", control.Socket, 2*time.Second)
			if err != nil {
				return fmt.Errorf(
					"no control socket — it is off unless asked for, see the Control socket switch: %w",
					err)
			}
			defer conn.Close()

			commands, err := sequence(cmd.InOrStdin(), args)
			if err != nil {
				return err
			}
			if len(commands) == 0 {
				return fmt.Errorf("nothing to run")
			}

			// A command may run for as long as it was asked to, so the answers take as long as the
			// commands do.
			if err := conn.SetReadDeadline(time.Now().Add(time.Hour)); err != nil {
				return err
			}

			// One at a time, waiting for each to finish. Sending them all at once would leave the
			// rest of a sequence running after one of them had already failed, which is the
			// opposite of what somebody scripting a check wants.
			in := bufio.NewScanner(conn)
			for _, line := range commands {
				if _, err := fmt.Fprintln(conn, line); err != nil {
					return err
				}
				if err := answer(in); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

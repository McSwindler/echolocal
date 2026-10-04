package audioproxy

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"log/slog"
)

const fifoPath = "/tmp/echod-mic"

type Proxy struct {
	f          *os.File
	cmd        *exec.Cmd
}

// Start creates the FIFO pipe from audioproxyd
func Start() (*Proxy, error) {
	if err := ensureFIFO(fifoPath); err != nil {
		return nil, err
	}

	slog.Info("Starting FIFO pipe from audioproxyd for BMicsOut")

	cmd := exec.Command(
		"/usr/local/bin/shmbuf_tool",
		"-m", "2",
		"-s", "1",
		"-S", "BMicsOut.shm",
		"-o", fifoPath,
	)

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start shmbuf_tool: %w", err)
	}

	// Blocking open. This waits until shmbuf_tool opens the FIFO for writing.
	f, err := os.Open(fifoPath)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("open microphone FIFO: %w", err)
	}

	return &Proxy{
		f:   f,
		cmd: cmd,
	}, nil
}

func (p *Proxy) Stop() {
	if p.f != nil {
		p.f.Close()
	}

	slog.Info("Killing FIFO pipe from audioproxyd")
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
		_ = p.cmd.Process.Kill()
	}
	_ = p.cmd.Wait()
}

func ensureFIFO(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	err := syscall.Mkfifo(path, 0644)

	if err != nil {
		return fmt.Errorf("mkfifo: %w", err)
	}
	return nil
}

func (p *Proxy) Read(b []byte) (int, error) {
	return p.f.Read(b)
}
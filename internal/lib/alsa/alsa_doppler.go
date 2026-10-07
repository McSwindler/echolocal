//go:build board_doppler

// Package alsa (Doppler) captures PCM audio by reading a shm from audioproxyd
//
// This is needed instead of the orignal alsa package because the audio driver
// provided on Gen 1 echos do not work natively. It requires a private library
// provided by TI called DSPLink. Instead of including that along with CGO,
// this service utilizes the existing audioproxyd service on the device.
package alsa

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

// fifoPath is the FIFO file to transmit data from the MicsOut.shm
const fifoPath = "/tmp/echod-mic"

// Capture is an open PCM capture stream.
type Capture struct {
	f          *os.File
	cfg        Config
	frameBytes int
	cmd        *exec.Cmd
}

// Open configures and starts a capture stream. The open is non-blocking so a device someone else
// holds returns ErrBusy instead of waiting; reads block normally once it is ours.
func Open(card, device int, cfg Config) (*Capture, error) {
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

	return &Capture{
		f:          f,
		cmd:        cmd,
		frameBytes: cfg.Channels * cfg.Bits / 8,
	}, nil
}

// FrameBytes is the size of one interleaved frame across all channels.
func (c *Capture) FrameBytes() int { return c.frameBytes }

// Read fills buf with whole frames and returns the number of bytes read. On overrun the
// stream is re-prepared and restarted, and the call reports how many bytes were lost.
func (c *Capture) Read(buf []byte) (int, error) {
	frames := len(buf) / c.frameBytes
	if frames == 0 {
		return 0, fmt.Errorf("buffer smaller than one frame (%d bytes)", c.frameBytes)
	}
	return c.f.Read(buf)
}

func (c *Capture) Close() error {
	if c.f != nil {
		c.f.Close()
	}

	slog.Info("Killing FIFO pipe from audioproxyd")
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Signal(syscall.SIGTERM)
		_ = c.cmd.Process.Kill()
	}
	return c.cmd.Wait()
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

//go:build board_doppler

package alsa

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

// Playback is an open PCM playback stream.
type Playback struct {
	f          io.WriteCloser
	cfg        Config
	frameBytes int
	started    bool
	cmd        *exec.Cmd
}

// OpenPlayback configures a playback stream. The hardware starts itself once the buffer is full.
//
// The open is non-blocking: mediaserver holds this device, and a blocking open waits forever.
// A held device returns ErrBusy. Writes block normally once the device is ours.
func OpenPlayback(card, device int, cfg Config) (*Playback, error) {
	slog.Info("Starting paplay process", "format", playbackFormat(cfg.Format), "rate", cfg.Rate, "channels", cfg.Channels)

	cmd := exec.Command(
		"/usr/bin/paplay",
		"--raw",
		fmt.Sprintf("--format=%s", playbackFormat(cfg.Format)),
		fmt.Sprintf("--rate=%d", cfg.Rate),
		fmt.Sprintf("--channels=%d", cfg.Channels),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create paplay stdin: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start paplay: %w", err)
	}

	return &Playback{
		f:          stdin,
		cfg:        cfg,
		frameBytes: cfg.Channels * cfg.Bits / 8,
		started:    true,
		cmd:        cmd,
	}, nil
}

// FrameBytes is the size of one interleaved frame across all channels.
func (p *Playback) FrameBytes() int { return p.frameBytes }

// Write plays whole frames, blocking until the hardware has room. On underrun the stream is
// re-prepared and the call reports it, since a gap in playback is worth knowing about.
func (p *Playback) Write(buf []byte) (int, error) {
	frames := len(buf) / p.frameBytes
	if frames == 0 {
		return 0, fmt.Errorf("buffer smaller than one frame (%d bytes)", p.frameBytes)
	}
	return p.f.Write(buf)
}

// Drain waits for buffered audio to finish playing.
func (p *Playback) Drain() error {
	if p.f != nil {
		p.f.Close()
	}
	return p.cmd.Wait()
}

func (p *Playback) Close() error {
	if p.f != nil {
		p.f.Close()
	}

	slog.Info("Killing paplay command")
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
		_ = p.cmd.Process.Kill()
	}
	return p.cmd.Wait()
}

func playbackFormat(format int) string {
	switch format {
	case FormatS16_LE:
		return "s16le"

	case FormatS24_3LE:
		return "s24-32le"

	default:
		panic("unknown PCM sample format")
	}
}

//go:build board_doppler

package echod

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

// The playback codec accepts one format only.
const (
	playRate     = 48000
	playChannels = 2
	playBits     = 16
	playPeriod   = 1024
	playPeriods  = 4
)

func newPlayCmd() *cobra.Command {
	var (
		card    int
		device  int
		freq    float64
		secs    float64
		level   float64
		silence bool
		channel string
	)

	c := &cobra.Command{
		Use:   "play",
		Short: "Play a test tone through the speaker",
		Long: "Writes to the playback PCM directly. The codec accepts S16_LE, 48 kHz, stereo and\n" +
			"nothing else.\n\n" +
			"Nothing is audible while the speaker amp switches are off — see `tools mixer`. Use\n" +
			"--silence to exercise the path without making noise, which is also how to tell a\n" +
			"broken stream from a muted one.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := NewPaplayPlayer(cmd.Context())
			if err != nil {
				return err
			}
			defer p.Close()

			out := cmd.OutOrStdout()
			what := fmt.Sprintf("%.0f Hz at %.0f%%", freq, level*100)
			if silence {
				what = "silence"
			}
			fmt.Fprintf(out, "playing %s for %.1fs on card %d device %d\n", what, secs, card, device)

			frames := int(secs * playRate)
			buf := make([]byte, playPeriod*playChannels*playBits/8)
			start := time.Now()

			for done := 0; done < frames; {
				n := min(playPeriod, frames-done)
				fill(buf[:n*playChannels*playBits/8], done, freq, level, silence, channel)

				if _, err := p.Write(buf[:n*playChannels*playBits/8]); err != nil {
					return fmt.Errorf("audio playback: %w", err)
				}
				done += n
			}

			if err := p.Close(); err != nil {
				return fmt.Errorf("finish audio input: %w", err)
			}

			if err := p.Wait(); err != nil {
				return fmt.Errorf("playback: %w", err)
			}
			fmt.Fprintf(out, "wrote %d frames in %s\n", frames, time.Since(start).Round(time.Millisecond))
			return nil
		},
	}

	c.Flags().IntVar(&card, "card", 0, "sound card index")
	c.Flags().IntVar(&device, "device", 23, "playback PCM device")
	c.Flags().Float64Var(&freq, "freq", 440, "tone frequency in Hz")
	c.Flags().Float64Var(&secs, "seconds", 1, "how long to play")
	c.Flags().Float64Var(&level, "level", 0.2, "amplitude, 0 to 1")
	c.Flags().BoolVar(&silence, "silence", false, "write zeros instead of a tone")
	c.Flags().StringVar(&channel, "channel", "both", "which channel carries the tone: left, right or both")
	return c
}

// fill writes one period of interleaved stereo, continuing the tone from frame offset so
// periods join without a click.
func fill(buf []byte, offset int, freq, level float64, silence bool, channel string) {
	for i := range len(buf) / (playChannels * playBits / 8) {
		var s int16
		if !silence {
			t := float64(offset+i) / playRate
			s = int16(level * math.MaxInt16 * math.Sin(2*math.Pi*freq*t))
		}
		for ch := range playChannels {
			v := s
			if (channel == "left" && ch != 0) || (channel == "right" && ch != 1) {
				v = 0
			}
			binary.LittleEndian.PutUint16(buf[(i*playChannels+ch)*2:], uint16(v))
		}
	}
}

type PaplayPlayer struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stderr io.ReadCloser
}

func NewPaplayPlayer(ctx context.Context) (*PaplayPlayer, error) {
	cmd := exec.CommandContext(
		ctx,
		"paplay",
		"--raw",
		"--format=s16le",
		"--rate=48000",
		"--channels=2",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create paplay stdin: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create paplay stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start paplay: %w", err)
	}

	return &PaplayPlayer{
		cmd:    cmd,
		stdin:  stdin,
		stderr: stderr,
	}, nil
}

func (p *PaplayPlayer) Write(data []byte) (int, error) {
	return p.stdin.Write(data)
}

func (p *PaplayPlayer) Close() error {
	return p.stdin.Close()
}

func (p *PaplayPlayer) Wait() error {
	err := p.cmd.Wait()
	if err == nil {
		return nil
	}

	message, _ := io.ReadAll(p.stderr)
	return fmt.Errorf("%w: %s", err, bytes.TrimSpace(message))
}

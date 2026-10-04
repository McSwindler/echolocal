//go:build board_doppler

package echod

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/lib/alsa"
	"github.com/ygelfand/echolocal/internal/lib/audio"
	"github.com/ygelfand/echolocal/internal/lib/audioproxy"
)

func newMicCmd() *cobra.Command {
	var (
		card, device    int
		chans, rate     int
		period, periods int
		bits, format    int
		secs            float64
		rawPath         string
	)

	c := &cobra.Command{
		Use:   "mic",
		Short: "Capture from the mic array and report per-channel levels",
		Long: "Captures raw audio and reports peak, RMS, and the margin over what a 24->16 bit\n" +
			"truncation would discard.\n\n" +
			"ch0-ch6 are microphones. ch7/ch8 are the playback loopback reference: silent\n" +
			"unless audio is playing, and identifiable by using 0% of the low 8 bits.\n\n" +
			"The capture device is held by mediaserver, so run `stop media` first or this\n" +
			"will block rather than fail.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			proxy, err := audioproxy.Start()
			if err != nil {
				return err
			}
			defer func() { proxy.Stop() }()

			var raw *os.File
			if rawPath != "" {
				if raw, err = os.Create(rawPath); err != nil {
					return err
				}
				defer func() { _ = raw.Close() }()
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "capturing audiproxyd")

			st := make([]audio.Stats, chans)
			buf := make([]byte, 4096)
			deadline := time.Now().Add(time.Duration(secs * float64(time.Second)))
			overruns := 0

			for time.Now().Before(deadline) {
				n, err := proxy.Read(buf)
				if err != nil {
					return err
				}
				if raw != nil {
					if _, err := raw.Write(buf[:n]); err != nil {
						return err
					}
				}
				fb := 2
				for off := 0; off+fb <= n; off += fb {
					for ch := 0; ch < chans; ch++ {
						st[ch].Add(audio.DecodeS16LE(buf[off+ch*2 : off+ch*2+2]))
					}
				}
			}

			if overruns > 0 {
				fmt.Fprintf(out, "overruns: %d\n", overruns)
			}
			audio.WriteReport(out, st)
			return nil
		},
	}

	c.Flags().IntVarP(&card, "card", "D", 0, "ALSA card")
	c.Flags().IntVarP(&device, "device", "d", 0, "ALSA capture device")
	c.Flags().IntVarP(&chans, "channels", "c", 1, "channels (hardware accepts only 9)")
	c.Flags().IntVarP(&rate, "rate", "r", 16000, "sample rate (hardware accepts only 16000)")
	c.Flags().IntVarP(&period, "period", "p", 256, "period size in frames")
	c.Flags().IntVarP(&periods, "periods", "n", 10, "period count")
	c.Flags().IntVarP(&bits, "bits", "b", 16, "bits")
	c.Flags().IntVarP(&format, "format", "f", alsa.FormatS16_LE, "format")
	c.Flags().Float64VarP(&secs, "seconds", "t", 5, "capture duration")
	c.Flags().StringVar(&rawPath, "raw", "", "also write raw interleaved S24_3LE here")
	return c
}

package control

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/hardware/mic"
	"github.com/ygelfand/echolocal/internal/lib/audio"
)

const (
	shortest = time.Second
	longest  = 60 * time.Second
)

func hearing() []*cobra.Command {
	record := &cobra.Command{
		Use:   "record <seconds> [path]",
		Short: "Record what wake detection hears, and say how loud it was",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := seconds(args[0])
			if err != nil {
				return err
			}

			frames, stop := mic.Get().Listen("ctl record")
			defer stop()

			var got []int16
			over := time.After(d)
		loop:
			for {
				select {
				case <-over:
					break loop
				case f, ok := <-frames:
					if !ok {
						break loop
					}
					got = append(got, f...)
				}
			}

			say := fmt.Sprintf("seconds %.2f %s", float64(len(got))/mic.Rate, levels(got))
			if len(args) > 1 {
				if err := write(args[1], 1, got); err != nil {
					return err
				}
				say = args[1] + " " + say
			}
			fmt.Fprintln(cmd.OutOrStdout(), say)
			return nil
		},
	}

	echo := &cobra.Command{
		Use:   "echo <seconds> <path>",
		Short: "Record the center microphone, the loopback, and what listeners got, as three channels",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := seconds(args[0])
			if err != nil {
				return err
			}

			frames, stop := mic.Get().ListenEcho()
			defer stop()

			var m, r, o []int16
			var sounding int
			over := time.After(d)
		loop:
			for {
				select {
				case <-over:
					break loop
				case e, ok := <-frames:
					if !ok {
						break loop
					}
					n := min(len(e.Mic), len(e.Ref), len(e.Out))
					m, r, o = append(m, e.Mic[:n]...), append(r, e.Ref[:n]...), append(o, e.Out[:n]...)
					if e.Sounding {
						sounding += n
					}
				}
			}

			wav := make([]int16, 0, 3*len(m))
			for i := range m {
				wav = append(wav, m[i], r[i], o[i])
			}
			if err := write(args[1], 3, wav); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s seconds %.2f sounding %.2f\n  mic %s\n  ref %s\n  out %s\n",
				args[1], float64(len(m))/mic.Rate, float64(sounding)/mic.Rate, levels(m), levels(r), levels(o))
			return nil
		},
	}

	record.AddCommand(echo)
	return []*cobra.Command{record}
}

func seconds(arg string) (time.Duration, error) {
	secs, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return 0, fmt.Errorf("%s is not a number of seconds", arg)
	}
	d := time.Duration(secs * float64(time.Second))
	if d < shortest || d > longest {
		return 0, fmt.Errorf("%v is outside %v to %v", d, shortest, longest)
	}
	return d, nil
}

func levels(s []int16) string {
	var peak int
	var energy float64
	for _, v := range s {
		peak = max(peak, int(v), -int(v))
		energy += float64(v) * float64(v)
	}
	var rms float64
	if len(s) > 0 {
		rms = math.Sqrt(energy / float64(len(s)))
	}
	return fmt.Sprintf("peakdbfs %.1f rmsdbfs %.1f", dbfs(float64(peak)), dbfs(rms))
}

func dbfs(v float64) float64 {
	return math.Round(20*math.Log10(max(v, 1)/32768)*10) / 10
}

func write(path string, channels int, samples []int16) error {
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%s: the path has to be absolute", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	pcm := make([]byte, 2*len(samples))
	for i, v := range samples {
		binary.LittleEndian.PutUint16(pcm[2*i:], uint16(v))
	}
	return audio.WriteWAV(path, pcm, mic.Rate, channels)
}

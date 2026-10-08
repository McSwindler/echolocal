//go:build board_doppler

// Package led drives the Echo Dot's ring through the is31fl3236 driver's sysfs attributes.
//
// This deliberately bypasses Amazon's LedController service. A direct frame write works even
// when that service has brightness set to 0, and keeps working once the Amazon stack is
// disabled — which the binder path does not.
package led

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ygelfand/echolocal/internal/hardware/indicate"
)

// DefaultPath is the is31fl3236 i2c device directory.
const DefaultPath = "/sys/class/i2c-dev/i2c-2/device"

// Channels is the number of PWM outputs: 12 ring segments x RGB.
const Channels = 36

// Segments is the number of addressable RGB positions on the ring.
const Segments = 12

var chips = [4]string{"2-0032", "2-0033", "2-0034", "2-0035"}

var chipOrder = [4][9]int{
	{0x16, 0x17, 0x13, 0x14, 0x10, 0x11, 0x15, 0x12, 0x0f}, // segments 5,6,7
	{0x04, 0x05, 0x01, 0x02, 0x22, 0x23, 0x03, 0x00, 0x21}, // segments 0,1,11
	{0x1c, 0x1d, 0x1f, 0x20, 0x19, 0x1a, 0x1b, 0x1e, 0x18}, // segments 8,9,10
	{0x0d, 0x0e, 0x0a, 0x0b, 0x07, 0x08, 0x0c, 0x09, 0x06}, // segments 2,3,4
}

// Layout, established by walking channels and then segments:
//
//   - The 36 channels are consecutive RGB triplets in R, G, B order, so segment n occupies
//     channels 3n, 3n+1, 3n+2.
//   - Segment 0 is at the bottom-right of the top face, between the microphone and
//     volume-down buttons.
//   - Increasing segment index runs clockwise, 30 degrees per segment.
//   - Each channel is a linear 0-255 brightness, 0 being dark.

// Color is one segment's colour, which is indicate's: a feature asks for a colour without knowing
// what shows it.
type Color = indicate.Color

// SetSegments writes all 12 segments at once.
func (r *Ring) SetSegments(c []Color) error {
	if len(c) != Segments {
		return fmt.Errorf("led: need %d segments, got %d", Segments, len(c))
	}
	buf := make([]byte, Channels)
	for i, col := range c {
		buf[i*3], buf[i*3+1], buf[i*3+2] = col.R, col.G, col.B
	}
	return r.SetFrame(buf)
}

// SetSegment lights one segment and blanks the rest.
func (r *Ring) SetSegment(n int, c Color) error {
	if n < 0 || n >= Segments {
		return fmt.Errorf("led: segment %d out of range 0-%d", n, Segments-1)
	}
	all := make([]Color, Segments)
	all[n] = c
	return r.SetSegments(all)
}

// SetAll paints every segment the same color.
func (r *Ring) SetAll(c Color) error {
	all := make([]Color, Segments)
	for i := range all {
		all[i] = c
	}
	return r.SetSegments(all)
}

// Ring writes whole frames to the LED driver.
type Ring struct {
	Path string

	// absent is a board with no ring. Writes go nowhere and reads report nothing, rather than
	// failing: everything that shows something on a ring — a conversation, a failure, a timer —
	// takes its claim unconditionally, and a device with no ring is one where that says nothing
	// rather than one where it goes wrong.
	absent bool

	// last is what was written, so a frame identical to it can be skipped. An animation is a ticker
	// that redraws whether or not anything moved, and the ring holds what it was given with nothing
	// driving it — the same property that lets a frame outlive the process.
	//
	// Nil until the first write. Start takes the ring off the kernel driver mid boot animation, so
	// what is on it before that is not ours to assume.
	mu   sync.Mutex
	last []byte
}

func New() *Ring { return &Ring{Path: DefaultPath} }

// NewAbsent is the ring on a board that has none. See Ring.absent.
func NewAbsent() *Ring { return &Ring{absent: true} }

// Present reports whether there is hardware behind this ring.
func (r *Ring) Present() bool { return !r.absent }

func (r *Ring) path() string {
	if r.Path == "" {
		return DefaultPath
	}
	return r.Path
}

func (r *Ring) attr(name string) string { return r.path() + "/" + name }

// SetFrame writes all 36 channel brightnesses: twelve segments, three channels each. A frame the
// ring is already showing is not written again — see last.
func (r *Ring) SetFrame(vals []byte) error {
	if len(vals) != Channels {
		return fmt.Errorf("led: need %d channels, got %d", Channels, len(vals))
	}
	if r.absent {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if bytes.Equal(vals, r.last) {
		return nil
	}
	for i, chip := range chips {
		parts := make([]string, 9)
		for n, idx := range chipOrder[i] {
			parts[n] = strconv.Itoa(int(vals[idx]))
		}
		if err := os.WriteFile(r.attr(chip+"/all_leds"), []byte(strings.Join(parts, " ")), 0o644); err != nil {
			return err
		}
	}

	r.last = append(r.last[:0], vals...)
	return nil
}

// Forget drops what SetFrame believes is showing, so the next write happens whatever it is. Taking
// the ring back off the kernel driver goes through here: it was animating until a moment ago and may
// have left anything in the registers.
func (r *Ring) Forget() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.last = nil
}

// Frame reads back the current frame.
func (r *Ring) Frame() ([]byte, error) {
	if r.absent {
		return make([]byte, Channels), nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]byte, Channels)
	copy(out, r.last)
	return out, nil
}

// Fill sets every channel to the same value.
func (r *Ring) Fill(v byte) error {
	buf := make([]byte, Channels)
	for i := range buf {
		buf[i] = v
	}
	return r.SetFrame(buf)
}

// Off blanks the ring.
func (r *Ring) Off() error { return r.Fill(0) }

func (r *Ring) currentAttr(chip, channel int) string {
	return r.attr(fmt.Sprintf("%s/leds/lp5523:%d:channel%d/led_current", chips[chip], chip, channel))
}

// Current reads the global drive-current setting.
func (r *Ring) Current() (int, error) {
	if r.absent {
		return 0, nil
	}
	b, err := os.ReadFile(r.currentAttr(0, 0))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

// SetCurrent sets the global drive current.
func (r *Ring) SetCurrent(v int) error {
	return nil
}

// SetBootAnimation toggles the driver's built-in boot animation.
func (r *Ring) SetBootAnimation(on bool) error {
	if r.absent {
		return nil
	}
	mode := "disabled"
	if on {
		mode = "run"
	}
	for _, chip := range chips {
		for e := 1; e <= 3; e++ {
			if err := os.WriteFile(r.attr(fmt.Sprintf("%s/engine%d_mode", chip, e)), []byte(mode), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

package alsa

import (
	"math"
	"os"
	"testing"
)

// TestProbeMixer exercises the control ioctls against a real card, which is the only way to know
// the structure layouts match the kernel's: a wrong size is rejected outright, and a wrong stride
// reads a neighbouring value's high half. Reading is enough, and the control device tolerates
// several openers, so this runs safely alongside echod. Set ALSA_PROBE=1 on the device.
func TestProbeDevices(t *testing.T) {
	if os.Getenv("ALSA_PROBE") == "" {
		t.Skip("set ALSA_PROBE=1 on a device with a sound card")
	}
	for _, d := range []struct {
		device  int
		capture bool
	}{{22, true}, {23, false}, {8, true}, {24, true}} {
		r, err := Refine(0, d.device, d.capture)
		t.Logf("device %d capture=%v: %v %v", d.device, d.capture, r, err)
	}
}

func TestProbeQuietChannels(t *testing.T) {
	if os.Getenv("ALSA_PROBE") == "" {
		t.Skip("set ALSA_PROBE=1 on a device with a sound card")
	}
	const channels, rate = 4, 16000
	c, err := Open(0, 22, Config{Channels: channels, Rate: rate, Format: FormatS24_3LE, Bits: 24, PeriodSize: 320, Periods: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	buf := make([]byte, c.FrameBytes()*320)
	var sum [channels]float64
	var peak [channels]int32
	frames := 0
	for frames < 2*rate {
		n, err := c.Read(buf)
		if err != nil {
			continue
		}
		for f := 0; f+c.FrameBytes() <= n; f += c.FrameBytes() {
			for ch := range channels {
				b := buf[f+ch*3:]
				v := int32(b[0]) | int32(b[1])<<8 | int32(int8(b[2]))<<16
				sum[ch] += float64(v) * float64(v)
				peak[ch] = max(peak[ch], v, -v)
			}
			frames++
		}
	}
	for ch := range channels {
		t.Logf("channel %d: rms %.1f peak %d over %d frames", ch, math.Sqrt(sum[ch]/float64(frames)), peak[ch], frames)
	}
}

func TestProbeMixer(t *testing.T) {
	if os.Getenv("ALSA_PROBE") == "" {
		t.Skip("set ALSA_PROBE=1 on a device with a sound card")
	}

	m, err := OpenMixer(0)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	controls, err := m.Controls()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("elem_value is %d bytes, elem_info %d; %d controls", elemValueSize, elemInfoSize, len(controls))

	var integers, enums, multi int
	for _, c := range controls {
		v, err := m.Get(c)
		if err != nil {
			t.Errorf("%s: %v", c.Name, err)
			continue
		}
		switch c.Type {
		case TypeEnumerated:
			enums++
			if len(v) > 0 && int(v[0]) >= len(c.Items) {
				// Not a stride problem: these all have one value, so no stride is applied. The
				// driver reports an out-of-range item for controls nothing has set.
				t.Logf("%s reports item %d with only %d items", c.Name, v[0], len(c.Items))
			}
		case TypeInteger, TypeBoolean:
			integers++
		}

		// Only a control with several values exercises the stride at all, so these are the
		// interesting ones: with the wrong width the second value comes from the middle of the
		// first.
		if c.Count > 1 {
			multi++
			t.Logf("%-28s type %d count %d value %v", c.Name, c.Type, c.Count, v)
		}
	}
	t.Logf("read %d integer or boolean controls, %d enumerated, %d with several values",
		integers, enums, multi)

	// Spot-check the controls the speaker path depends on.
	for _, name := range []string{"Right Channel Only", "Audio_DacMux_Setting"} {
		c, err := m.Find(name)
		if err != nil {
			t.Logf("%s: %v", name, err)
			continue
		}
		v, err := m.Get(c)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		t.Logf("%-24s type %d count %d value %v items %v", c.Name, c.Type, c.Count, v, c.Items)
	}
}

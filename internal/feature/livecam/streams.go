package livecam

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/lib/hook"
	"github.com/ygelfand/echolocal/internal/setting"
)

const StreamGroup setting.Group = "Streams"

var (
	mainSizes    = []string{"1280x720", "864x480", "640x480"}
	qualities    = []string{"low", "standard", "high"}
	qualityScale = map[string]float64{"low": 0.5, "standard": 1, "high": 2}
)

const (
	keyframeMin = 1
	keyframeMax = 10
	bitrateMin  = 250_000
	bitrateMax  = 10_000_000
)

var StreamsChanged hook.Hook[[]int]

func defaultMainSize() string {
	b := component.Board()
	return fmt.Sprintf("%dx%d", b.CameraWidth, b.CameraHeight)
}

func parseSize(s string) (int, int) {
	w, h, _ := strings.Cut(s, "x")
	wi, _ := strconv.Atoi(w)
	hi, _ := strconv.Atoi(h)
	return wi, hi
}

func bitrateFor(k Knobs, w, h int) int {
	scale, ok := qualityScale[k.Quality]
	if !ok {
		scale = 1
	}
	return max(bitrateMin, min(int(float64(w*h*FPS/10)*scale), bitrateMax))
}

func toggle(name, icon string, field func(*Knobs) *bool) Knob {
	k := Knob{Name: name, Kind: setting.Toggle, Group: StreamGroup, Icon: icon,
		Read: func(k *Knobs) string { return setting.OnOff(*field(k)) }}
	k.Write = func(at *Knobs, v string) error {
		on, ok := setting.Boolean(v)
		if !ok {
			return k.Bad(v, "on or off")
		}
		*field(at) = on
		return nil
	}
	return k
}

func sizeLabels(values []string) []setting.Option {
	out := make([]setting.Option, len(values))
	for i, v := range values {
		out[i] = setting.Option{Value: v, Label: strings.Replace(v, "x", "×", 1)}
	}
	return out
}

func streamRows() []Knob {
	mainSize := Knob{Name: "main_size", Kind: setting.Choice, Group: StreamGroup, Icon: "mdi:aspect-ratio",
		Options: sizeLabels(mainSizes), Read: func(k *Knobs) string { return k.MainSize }}
	mainSize.Write = func(k *Knobs, v string) error {
		if !slices.Contains(mainSizes, v) {
			return mainSize.Bad(v, "")
		}
		k.MainSize = v
		return nil
	}

	keyframe := Knob{Name: "keyframe", Kind: setting.Number, Group: StreamGroup, Icon: "mdi:key-variant",
		Slider: true, Min: keyframeMin, Max: keyframeMax, Unit: "s",
		Read: func(k *Knobs) string { return strconv.Itoa(k.Keyframe) }}
	keyframe.Write = func(k *Knobs, s string) error {
		n, err := strconv.Atoi(s)
		if err != nil || n < keyframeMin || n > keyframeMax {
			return keyframe.Bad(s, fmt.Sprintf("between %d and %d", keyframeMin, keyframeMax))
		}
		k.Keyframe = n
		return nil
	}

	return []Knob{
		toggle("main_on", "mdi:video", func(k *Knobs) *bool { return &k.MainOn }),
		mainSize,
		keyframe,
		choice("quality", StreamGroup, "mdi:high-definition", words(qualities), func(k *Knobs) *string { return &k.Quality }),
	}
}

func Served() []int {
	k := Saved()
	var out []int
	if k.MainOn {
		out = append(out, 0)
	}
	return out
}

func shapeOf(k Knobs) string {
	return fmt.Sprintf("%s/%d/%s", k.MainSize, k.Keyframe, k.Quality)
}

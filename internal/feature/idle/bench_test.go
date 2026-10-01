package idle

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/visual"
)

func BenchmarkIdlePaint(b *testing.B) {
	palette := theme.Default()
	for _, at := range []struct {
		name  string
		kinds []visual.Kind
		face  config.Face
	}{
		{"orb", []visual.Kind{visual.Orb}, config.FacePlain},
		{"classic-vu", []visual.Kind{visual.ClassicVU}, config.FacePlain},
		{"classic-vu+orb", []visual.Kind{visual.ClassicVU, visual.Orb}, config.FacePlain},
		{"classic-vu+orb/noclock", []visual.Kind{visual.ClassicVU, visual.Orb}, config.FaceNone},
		{"clock-only", nil, config.FacePlain},
	} {
		b.Run(at.name, func(b *testing.B) {
			var slots []config.IdleVisual
			for _, k := range at.kinds {
				slots = append(slots, config.IdleVisual{Kind: string(k), Source: config.SourceBoth})
			}
			cfg := idled(func(c *config.Idle) { c.Face, c.Size = at.face, config.SizeMedium })
			img := ui.NewImage(1200, 1920, palette.Background)
			var st Stamps
			PaintWith(img, cfg, slots, nil, palette, noon, &st)
			b.ResetTimer()
			for range b.N {
				PaintWith(img, cfg, slots, nil, palette, noon, &st)
			}
		})
	}
}

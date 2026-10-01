package player

import (
	"github.com/ygelfand/echolocal/internal/feature/media"
	"github.com/ygelfand/echolocal/internal/ui"
	"github.com/ygelfand/echolocal/internal/ui/mark"
)

func cover(now media.Now) *ui.Image { return ui.Picture(now.Art, now.ArtID) }

func kindIcon(k media.Kind) ui.Icon {
	switch k {
	case media.FromBluetooth:
		return mark.Bluetooth
	case media.FromGroup:
		return mark.Speakers
	case media.FromCast:
		return mark.Chromecast
	}
	return mark.HomeAssistant
}

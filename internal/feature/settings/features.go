package settings

import (
	"fmt"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/bluetooth"
	"github.com/ygelfand/echolocal/internal/feature/sendspin"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

// featuresPage is what the device does at all, as opposed to how it looks or how loud it is.
//
// These are whole subsystems rather than settings on one: each opens a port, holds hardware, or
// listens for something. They were switches in Home Assistant and nowhere else, which leaves a
// device that has not been adopted, or whose Home Assistant is down, unable to be set up from its
// own screen.
func featuresPage() *shell.Page {
	return &shell.Page{
		Title: say.T("features.title"),
		Build: func() ([]widget.Row, []func(int)) {
			cfg := config.Get()

			return []widget.Row{
					{
						Label: say.T("features.sendspin"),
						Hint:  say.T("features.sendspin.hint"),
						Kind:  widget.Toggle,
						On:    cfg.Sendspin.Enabled,
					},
					{
						Label: say.T("features.proxy"),
						Hint:  say.T("features.proxy.hint"),
						Kind:  widget.Toggle,
						On:    cfg.Bluetooth.Proxy,
					},
				}, []func(int){
					func(int) { sendspin.Get().SetEnabled(!cfg.Sendspin.Enabled) },
					func(int) { bluetooth.Get().SetProxy(!cfg.Bluetooth.Proxy) },
				}
		},
	}
}

// featuresOn is how many are on, for the row that leads here.
func featuresOn() string {
	var on, all int

	cfg := config.Get()
	for _, yes := range []bool{cfg.Sendspin.Enabled, cfg.Bluetooth.Proxy} {
		all++
		if yes {
			on++
		}
	}
	return fmt.Sprintf("%d of %d", on, all)
}

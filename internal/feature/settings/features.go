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

type feature struct {
	label, hint func() string
	on          func(config.Config) bool
	set         func(bool)
	page        func() *shell.Page
}

var features = []feature{
	{
		label: func() string { return say.T("features.sendspin") },
		hint:  func() string { return say.T("features.sendspin.hint") },
		on:    func(c config.Config) bool { return c.Sendspin.Enabled },
		set:   func(on bool) { sendspin.Get().SetEnabled(on) },
	},
	{
		label: func() string { return say.T("features.proxy") },
		hint:  func() string { return say.T("features.proxy.hint") },
		on:    func(c config.Config) bool { return c.Bluetooth.Proxy },
		set:   func(on bool) { bluetooth.Get().SetProxy(on) },
	},
}

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

			rows := make([]widget.Row, 0, len(features))
			acts := make([]func(int), 0, len(features))
			for _, f := range features {
				if f.page != nil {
					rows = append(rows, widget.Row{Label: f.label(), Hint: f.hint(), Kind: widget.Chevron})
					acts = append(acts, open(f.page()))
					continue
				}
				on := f.on(cfg)
				rows = append(rows, widget.Row{Label: f.label(), Hint: f.hint(), Kind: widget.Toggle, On: on})
				acts = append(acts, func(int) { f.set(!on) })
			}
			return rows, acts
		},
	}
}

// featuresOn is how many are on, for the row that leads here.
func featuresOn() string {
	var on, all int

	cfg := config.Get()
	for _, f := range features {
		if f.on == nil {
			continue
		}
		all++
		if f.on(cfg) {
			on++
		}
	}
	return fmt.Sprintf("%d of %d", on, all)
}

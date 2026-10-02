package volume

import (
	"sync"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

// Page is the screen for setting every level, one slider each.
func Page() shell.View {
	pageOnce.Do(func() { page = levels{build()} })
	return page
}

var (
	pageOnce sync.Once
	page     levels
)

// levels is the page, saying it shows them.
type levels struct{ *shell.Page }

func (levels) Shows(config.Stream) bool { return true }

func build() *shell.Page {
	return &shell.Page{
		Title: say.T("settings.volume"),
		Build: func() ([]widget.Row, []func(int)) {
			streams := config.Streams()

			rows := make([]widget.Row, 0, len(streams))
			acts := make([]func(int), 0, len(streams))

			for _, s := range streams {
				rows = append(rows, widget.Row{
					Label: s.Label(),
					Kind:  widget.Slider,
					Level: Get().Level(s),
				})
				acts = append(acts, set(s))
			}
			return rows, acts
		},
	}
}

func set(s config.Stream) func(int) {
	return func(level int) { Get().Set(s, level) }
}

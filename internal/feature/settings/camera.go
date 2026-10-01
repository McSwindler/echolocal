//go:build board_checkers || board_cronos

package settings

import (
	"log/slog"
	"strconv"
	"time"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/livecam"
	"github.com/ygelfand/echolocal/internal/feature/rtspd"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/lib/say"
	"github.com/ygelfand/echolocal/internal/setting"
	"github.com/ygelfand/echolocal/internal/ui/widget"
)

func init() {
	sections = append(sections, section{label: "settings.camera", icon: icons.ImagePhotoCamera, page: cameraPage})
	pages["camera"] = cameraPage
	features = append(features, feature{
		label: func() string { return say.T("features.rtsp") },
		hint:  func() string { return say.T("features.rtsp.hint") },
		on:    func(c config.Config) bool { return c.RTSP.Enabled },
		set:   func(on bool) { rtspd.Get().SetEnabled(on) },
	})
}

// shown is the groups this panel offers: the ones with something in them.
func shown() []setting.Group {
	var out []setting.Group
	for _, g := range livecam.Table().Groups() {
		if len(livecam.Table().In(g)) > 0 {
			out = append(out, g)
		}
	}
	return out
}

func set() livecam.Knobs { return livecam.Saved() }

func save(name, value string) {
	if err := livecam.Set(name, value); err != nil {
		slog.Error("the camera setting could not be saved", "setting", name, "err", err)
	}
}

const resetWindow = 3 * time.Second

var resetArmed time.Time

func cameraPage() *shell.Page {
	return camPage(&shell.Page{
		Title:    say.T("settings.camera"),
		Aside:    camAside,
		AsideTap: camTap,
		Build: func() ([]widget.Row, []func(int)) {
			sh := set()

			var rows []widget.Row
			var taps []func(int)

			for _, g := range shown() {
				if in := livecam.Table().In(g); len(in) == 1 {
					row, tap := cameraRow(in[0], &sh)
					row.Label = livecam.Table().Title(g)
					rows = append(rows, row)
					taps = append(taps, tap)
					continue
				}
				rows = append(rows, widget.Row{
					Label: livecam.Table().Title(g),
					Kind:  widget.Chevron,
					Value: livecam.Table().Sums(g, &sh),
				})
				taps = append(taps, open(sectionPage(g)))
			}
			label := say.T("camera.reset")
			if time.Since(resetArmed) < resetWindow {
				label = say.T("camera.reset.confirm")
			}
			rows = append(rows, widget.Row{Label: label})
			taps = append(taps, func(int) {
				if time.Since(resetArmed) >= resetWindow {
					resetArmed = time.Now()
					time.AfterFunc(resetWindow, shell.Get().Redraw)
					shell.Get().Redraw()
					return
				}
				resetArmed = time.Time{}
				if err := livecam.Reset(); err != nil {
					slog.Error("resetting the camera settings", "err", err)
				}
				shell.Get().Redraw()
			})
			return rows, taps
		},
	})
}

func watching(p *shell.Page) *shell.Page {
	p.Aside, p.AsideTap = camAside, camTap
	return camPage(p)
}

func sectionPage(g setting.Group) *shell.Page {
	return watching(&shell.Page{
		Title: livecam.Table().Title(g),
		Build: func() ([]widget.Row, []func(int)) {
			sh := set()

			var rows []widget.Row
			var taps []func(int)

			for _, s := range livecam.Table().In(g) {
				row, tap := cameraRow(s, &sh)
				rows = append(rows, row)
				taps = append(taps, tap)
			}
			return rows, taps
		},
	})
}

func cameraRow(s livecam.Knob, sh *livecam.Knobs) (widget.Row, func(int)) {
	row := widget.Row{Label: s.Title()}
	idle := s.Dim(sh)

	switch s.Kind {
	case setting.Toggle:
		row.Kind = widget.Toggle
		row.On = s.On(sh)
		return row, dimmed(&row, idle, func(int) {
			save(s.Name, setting.OnOff(!row.On))
		})

	case setting.Number:
		row.Kind = widget.Slider
		row.Level = s.Percent(s.Level(sh))
		row.Snap = func(level int) int { return s.Percent(s.Raw(level)) }
		if s.Scale == setting.Linear {
			row.Value = s.Read(sh)
			if s.Unit != "" {
				row.Value += " " + s.Unit
			}
		}
		return row, dimmed(&row, idle, func(level int) {
			save(s.Name, strconv.Itoa(s.Raw(level)))
		})

	case setting.Choice:
		row.Kind = widget.Chevron
		row.Value = s.Shows(sh)
		return row, dimmed(&row, idle, open(choicePage(s)))
	}

	row.Kind = widget.Plain
	row.Value = s.Read(sh)
	return row, nil
}

// dimmed fades a row that has nothing to do and takes its action away.
func dimmed(row *widget.Row, idle bool, act func(int)) func(int) {
	if !idle {
		return act
	}
	row.Dim = true
	return nil
}

func choicePage(s livecam.Knob) *shell.Page {
	return watching(&shell.Page{
		Title: s.Title(),
		Build: func() ([]widget.Row, []func(int)) {
			cfg := set()
			at := s.Read(&cfg)

			rows := make([]widget.Row, 0, len(s.Options))
			taps := make([]func(int), 0, len(s.Options))

			for _, o := range s.Options {
				rows = append(rows, widget.Row{
					Label:  s.Label(o),
					Kind:   widget.Plain,
					Chosen: o.Value == at,
				})
				taps = append(taps, func(int) {
					save(s.Name, o.Value)
				})
			}
			return rows, taps
		},
	})
}

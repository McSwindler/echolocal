//go:build board_checkers || board_cronos || board_rook

package control

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/clock"
	"github.com/ygelfand/echolocal/internal/feature/idle"
	"github.com/ygelfand/echolocal/internal/feature/privacy"
	apptheme "github.com/ygelfand/echolocal/internal/feature/theme"
	"github.com/ygelfand/echolocal/internal/feature/visuals"
	"github.com/ygelfand/echolocal/internal/ui/theme"
	"github.com/ygelfand/echolocal/internal/ui/visual"
)

type setting struct {
	name string
	says func(config.Config) string
	use  func(string) error
}

func settings() []setting {
	c, i, v := clock.Get(), idle.Get(), visuals.Get()

	return []setting{
		{"clock.face", func(c config.Config) string { return string(c.Clock.Face) }, choose(config.Faces(), c.SetFace)},
		{"clock.position", func(c config.Config) string { return string(c.Clock.Position) }, choose(config.Positions(), c.SetPosition)},
		{"clock.size", func(c config.Config) string { return string(c.Clock.Size) }, choose(config.Sizes(), c.SetSize)},
		{"clock.color", func(c config.Config) string { return string(c.Clock.Ink) }, choose(config.Inks(), c.SetInk)},
		{"clock.date", func(c config.Config) string { return onOff(c.Clock.Date) }, toggle(c.SetDate)},
		{"clock.hours", func(c config.Config) string { return string(c.Screen.Hours) }, choose(config.HourFormats(), c.SetHours)},

		{"idle.after", func(c config.Config) string { return string(c.Idle.After) }, choose(config.Delays(), i.SetAfter)},
		{"idle.face", func(c config.Config) string { return string(c.Idle.Face) }, choose(config.IdleFaces(), i.SetFace)},
		{"idle.position", func(c config.Config) string { return string(c.Idle.Position) }, choose(config.Positions(), i.SetPosition)},
		{"idle.align", func(c config.Config) string { return string(c.Idle.Align) }, choose(config.Aligns(), i.SetAlign)},
		{"idle.size", func(c config.Config) string { return string(c.Idle.Size) }, choose(config.Sizes(), i.SetSize)},
		{"idle.visual1", func(c config.Config) string { return orNone(c.Idle.First.Kind) }, idleKind(0)},
		{"idle.visual1.source", func(c config.Config) string { return string(c.Idle.First.Source) },
			choose(config.Sources(), func(s config.Source) { i.SetSource(0, s) })},
		{"idle.visual2", func(c config.Config) string { return orNone(c.Idle.Second.Kind) }, idleKind(1)},
		{"idle.visual2.source", func(c config.Config) string { return string(c.Idle.Second.Source) },
			choose(config.Sources(), func(s config.Source) { i.SetSource(1, s) })},

		{"screen.theme", func(c config.Config) string { return c.Screen.Theme }, paint},
		{"screen.marks", func(c config.Config) string { return onOff(c.Screen.Marks) }, toggle(privacy.Get().SetMarks)},
		{"screen.logo", func(c config.Config) string { return onOff(c.Screen.Logo) }, toggle(c.SetLogo)},
		{"screen.visual", func(c config.Config) string { return c.Visual.Kind }, choose(visual.Built(), v.SetKind)},
		{"screen.visual.fps", func(c config.Config) string { return strconv.Itoa(c.Visual.MaxFPS) }, fpsStep},
		{"screen.visual.seed", func(c config.Config) string { return strconv.Itoa(c.Visual.Seed) }, number(0, visual.SeedMost, v.SetSeed)},
		{"screen.visual.label", func(c config.Config) string { return orNone(c.Visual.Label) }, words(v.SetLabel)},
		{"screen.visual.lift", func(c config.Config) string { return strconv.Itoa(c.Microphone.VisualizerLift) }, number(0, visuals.LiftMax, v.SetLift)},
	}
}

func setCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set [name] [value...]",
		Short: "List the screen's settings, read one, or change one",
		RunE: func(cmd *cobra.Command, args []string) error {
			said, err := set(args)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), said)
			return nil
		},
	}
}

func set(args []string) (string, error) {
	all := settings()

	if len(args) == 0 {
		cfg := config.Get()
		var out []string
		for _, s := range all {
			out = append(out, fmt.Sprintf("%-20s %s", s.name, s.says(cfg)))
		}
		return strings.Join(out, "\n"), nil
	}

	at := slices.IndexFunc(all, func(s setting) bool { return strings.EqualFold(s.name, args[0]) })
	if at < 0 {
		return "", fmt.Errorf("no such setting %q, try set with no arguments", args[0])
	}
	if len(args) == 1 {
		return all[at].says(config.Get()), nil
	}
	if err := all[at].use(strings.Join(args[1:], " ")); err != nil {
		return "", fmt.Errorf("%s: %w", all[at].name, err)
	}
	return all[at].says(config.Get()), nil
}

func idleKind(slot int) func(string) error {
	return func(s string) error {
		if strings.EqualFold(s, "none") || s == "" {
			idle.Get().SetKind(slot, "")
			return nil
		}
		for _, k := range visual.Built() {
			if strings.EqualFold(string(k), s) || strings.EqualFold(k.Label(), s) {
				idle.Get().SetKind(slot, string(k))
				return nil
			}
		}
		return fmt.Errorf("want none or one of %s", strings.Join(config.Labels(visual.Built()), ", "))
	}
}

func choose[T config.Labelled](values []T, use func(T)) func(string) error {
	return func(s string) error {
		for _, v := range values {
			if strings.EqualFold(v.Label(), s) || strings.EqualFold(fmt.Sprint(v), s) {
				use(v)
				return nil
			}
		}
		return fmt.Errorf("want one of %s", strings.Join(config.Labels(values), ", "))
	}
}

func words(use func(string)) func(string) error {
	return func(s string) error {
		use(s)
		return nil
	}
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func toggle(use func(bool)) func(string) error {
	return func(s string) error {
		switch strings.ToLower(s) {
		case "on", "true", "yes", "1":
			use(true)
		case "off", "false", "no", "0":
			use(false)
		default:
			return fmt.Errorf("want on or off")
		}
		return nil
	}
}

func number(lo, hi int, use func(int)) func(string) error {
	return func(s string) error {
		v, err := strconv.Atoi(s)
		if err != nil || v < lo || v > hi {
			return fmt.Errorf("want a number from %d to %d", lo, hi)
		}
		use(v)
		return nil
	}
}

func fpsStep(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil || !slices.Contains(config.MaxFPSSteps, v) {
		return fmt.Errorf("want one of %v", config.MaxFPSSteps)
	}
	visuals.Get().SetMaxFPS(v)
	return nil
}

func paint(s string) error {
	for _, t := range theme.All {
		if strings.EqualFold(t.Name, s) {
			apptheme.Get().Choose(t.Name)
			return nil
		}
	}
	var names []string
	for _, t := range theme.All {
		names = append(names, t.Name)
	}
	return fmt.Errorf("want one of %s", strings.Join(names, ", "))
}

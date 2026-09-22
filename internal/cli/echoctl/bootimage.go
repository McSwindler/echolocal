package echoctl

import (
	"context"
	"fmt"
	"io"
	"os"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"

	"github.com/ygelfand/echolocal/internal/host/device"
	"github.com/ygelfand/echolocal/internal/host/profile"
)

// profileFor is the install this device needs, or why there is not one.
func profileFor(d *device.Device) (profile.Profile, error) {
	name, err := d.Getprop("ro.product.device")
	if err != nil {
		return profile.Profile{}, fmt.Errorf("asking the device what it is: %w", err)
	}
	return profile.For(name)
}

// resolveBootImage is the image to write: the one named on the command line, or the one published
// for this board.
//
// The fetch is the only part of an install that needs the network. Nothing is kept afterwards, so it
// runs once per install rather than once per machine.
func resolveBootImage(ctx context.Context, out io.Writer, p profile.Profile, path string) ([]byte, string, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		return data, path, err
	}

	title := fmt.Sprintf("Fetching the %s boot image (%.1f MB)", p.Board, float64(p.Boot.Size)/(1<<20))

	data, from, err := fetchImage(ctx, out, title, p.Boot.Resolve)
	if err != nil {
		return nil, "", fmt.Errorf("%w\n\nA local image can be given with --boot-image", err)
	}
	return data, from, nil
}

// fetchImage runs a resolve behind a progress bar, or plain lines where there is no terminal to draw
// one on.
func fetchImage(ctx context.Context, out io.Writer, title string,
	resolve func(context.Context, func(float64)) ([]byte, string, error),
) ([]byte, string, error) {
	if !isTerminal() {
		fmt.Fprintln(out, title)
		var last int
		return resolve(ctx, func(f float64) {
			if pct := int(f * 100); pct >= last+10 {
				last = pct
				fmt.Fprintf(out, "  %d%%\n", pct)
			}
		})
	}

	m := &fetchModel{
		title: title,
		bar:   progress.New(progress.WithDefaultBlend(), progress.WithWidth(barWidth)),
	}
	prog := tea.NewProgram(m, tea.WithContext(ctx), tea.WithOutput(out))

	go func() {
		data, from, err := resolve(ctx, func(f float64) { prog.Send(fetchAt(f)) })
		prog.Send(fetchDone{data: data, from: from, err: err})
	}()

	final, err := prog.Run()
	if err != nil {
		return nil, "", err
	}
	done := final.(*fetchModel)
	return done.data, done.from, done.err
}

type (
	fetchAt   float64
	fetchDone struct {
		data []byte
		from string
		err  error
	}
)

// barWidth is the bar in cells, narrowed to fit a terminal that cannot hold it.
const barWidth = 40

type fetchModel struct {
	title string
	bar   progress.Model
	at    float64

	data []byte
	from string
	err  error
}

func (m *fetchModel) Init() tea.Cmd { return nil }

func (m *fetchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchAt:
		m.at = float64(msg)
		return m, nil
	case fetchDone:
		m.data, m.from, m.err = msg.data, msg.from, msg.err
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.bar.SetWidth(min(barWidth, max(msg.Width-10, 10)))
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = ErrCancelled
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *fetchModel) View() tea.View {
	return tea.NewView(styleTitle.Render(m.title) + "\n\n  " + m.bar.ViewAs(m.at) + "\n")
}

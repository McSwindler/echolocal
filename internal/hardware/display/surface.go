package display

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ygelfand/echolocal/internal/layout"
	"github.com/ygelfand/echolocal/internal/lib/surface"
)

const (
	uiLayer     = 1
	reconnect   = 3 * time.Second
	redialEvery = time.Second
	strandEvery = time.Second
)

const helperWait = 60 * time.Second

// open is the helper's layer on a board that carries the helper, and the framebuffer on one that does not.
func (d *Driver) open(ctx context.Context) (*Panel, error) {
	if _, err := os.Stat(layout.Surface); err != nil {
		p, err := Open(d.path)
		if err != nil {
			return nil, fmt.Errorf("display: %s: %w", d.path, err)
		}
		return p, nil
	}

	d.mu.Lock()
	wait := helperWait
	if d.opened {
		wait = 0
	}
	d.mu.Unlock()

	p, err := OpenSurface(surface.Socket, wait)
	var stop func()
	for err != nil {
		if stop == nil {
			slog.Error("display: the helper is not answering", "err", err)
			stop = d.strand()
		}
		select {
		case <-ctx.Done():
			stop()
			return nil, ctx.Err()
		case <-time.After(redialEvery):
		}
		p, err = OpenSurface(surface.Socket, 0)
	}
	if stop != nil {
		stop()
		slog.Info("display: helper back")
	}
	d.mu.Lock()
	d.opened = true
	d.mu.Unlock()
	return p, nil
}

// Stranded is what to show while the helper is gone.
func (d *Driver) Stranded(draw func(*Panel) error) {
	d.mu.Lock()
	d.stranded = draw
	d.mu.Unlock()
}

func (d *Driver) strand() (stop func()) {
	d.mu.Lock()
	draw, rot, brightness := d.stranded, d.rot, d.brightness
	d.mu.Unlock()
	if draw == nil {
		return func() {}
	}

	p, err := Open(d.path)
	if err != nil {
		slog.Error("display: no framebuffer for the stranded screen", "err", err)
		return func() {}
	}
	p.Turn(rot)
	if err := SetBacklight(brightness); err != nil {
		slog.Error("backlight", "err", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() {
		t := time.NewTicker(strandEvery)
		defer t.Stop()
		failed := false
		for {
			err := draw(p)
			if err == nil {
				p.toPanelOrder()
				err = p.Flip()
			}
			if err != nil && !failed {
				slog.Error("display: showing the stranded screen", "err", err)
				failed = true
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	})
	return func() {
		cancel()
		wg.Wait()
		p.Close()
	}
}

func OpenSurface(sock string, wait time.Duration) (*Panel, error) {
	c, err := surface.DialWait(sock, wait)
	if err != nil {
		return nil, err
	}
	p := &Panel{surf: c, sock: sock}
	if err := p.attach(c); err != nil {
		c.Close()
		return nil, err
	}
	p.rot = Mounted()
	p.Width, p.Height = p.rot.Size(p.fbW, p.fbH)
	return p, nil
}

// Helper is the connection to the SurfaceFlinger helper, or nil on the framebuffer.
func (d *Driver) Helper() *surface.Client {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.panel == nil {
		return nil
	}
	return d.panel.surf
}

func (p *Panel) attach(c *surface.Client) error {
	w, h := c.Width, c.Height
	l, err := c.Create(uiLayer, 0, 0, w, h, 0, 0)
	if err != nil {
		return fmt.Errorf("display: the UI layer: %w", err)
	}
	p.surf, p.layer = c, l
	c.Dropped.Listen(p.Dropped.Emit)
	p.fbW, p.fbH = w, h
	p.stride = l.Stride
	p.mem = l.Pixels
	p.back, p.front = 0, 0
	p.doubled = true
	return nil
}

func (p *Panel) flipSurface() error {
	x0, y0, x1, y1 := p.fbRect(p.clip)
	if x1 <= x0 || y1 <= y0 {
		return nil
	}
	p.seq++
	err := p.surf.Frame(uiLayer, p.seq, []surface.Rect{{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}})
	var st surface.Status
	if err == nil || errors.As(err, &st) {
		return err
	}
	return p.redial(err)
}

func (p *Panel) redial(cause error) error {
	kept := append([]byte(nil), p.mem...)
	p.surf.Close()
	p.mem, p.layer = nil, nil

	c, err := surface.DialWait(p.sock, reconnect)
	if err != nil {
		return fmt.Errorf("display: lost the helper (%v) and could not get it back: %w", cause, err)
	}
	if err := p.attach(c); err != nil {
		c.Close()
		return err
	}
	copy(p.mem, kept)
	p.seq++
	return p.surf.Frame(uiLayer, p.seq, []surface.Rect{{W: p.fbW, H: p.fbH}})
}

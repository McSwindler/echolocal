package display

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ygelfand/echolocal/internal/lib/surface"
)

const (
	uiLayer   = 1
	reconnect = 30 * time.Second
)

const helperWait = 60 * time.Second

// open is the helper's layer when it answers, and the framebuffer otherwise.
func (d *Driver) open() (*Panel, error) {
	if _, err := os.Stat(surface.Socket); err == nil {
		p, err := OpenSurface(surface.Socket, helperWait)
		if err == nil {
			return p, nil
		}
		slog.Error("the SurfaceFlinger helper would not answer, drawing on the framebuffer", "err", err)
	}

	p, err := Open(d.path)
	if err != nil {
		return nil, fmt.Errorf("display: %s: %w", d.path, err)
	}
	return p, nil
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

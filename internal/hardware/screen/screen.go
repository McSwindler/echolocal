// Package screen owns the panel on a board that has one.
//
// write(2) to the framebuffer node returns success and reaches nothing, so drawing is mmap with
// FBIOPAN_DISPLAY to say which page to fetch. Callers work in viewed coordinates and Orientation
// reconciles those with the panel, which is mounted sideways.
package screen

import (
	"fmt"
	"image"
	"os"
	"syscall"
	"unsafe"
)

const DefaultPath = "/dev/graphics/fb0"

// From linux/fb.h.
const (
	iocGetVScreenInfo = 0x4600
	iocPutVScreenInfo = 0x4601
	iocGetFScreenInfo = 0x4602
	iocPanDisplay     = 0x4606
	iocBlank          = 0x4611

	blankUnblank = 0
)

type bitfield struct{ Offset, Length, MSBRight uint32 }

// varInfo is fb_var_screeninfo. Its size is not in the ioctl number, so the layout has to match the
// kernel's exactly or the fields read as nonsense.
type varInfo struct {
	Xres, Yres               uint32
	XresVirtual, YresVirtual uint32
	Xoffset, Yoffset         uint32
	BitsPerPixel, Grayscale  uint32
	Red, Green, Blue, Transp bitfield
	Nonstd, Activate         uint32
	Height, Width            uint32
	AccelFlags               uint32
	Pixclock                 uint32
	LeftMargin, RightMargin  uint32
	UpperMargin, LowerMargin uint32
	HsyncLen, VsyncLen       uint32
	Sync, Vmode, Rotate      uint32
	Colorspace               uint32
	Reserved                 [4]uint32
}

// fixInfo is fb_fix_screeninfo.
type fixInfo struct {
	ID                            [16]byte
	SmemStart                     uint32
	SmemLen                       uint32
	Type, TypeAux, Visual         uint32
	Xpanstep, Ypanstep, Ywrapstep uint16
	_                             uint16
	LineLength                    uint32
	MmioStart                     uint32
	MmioLen, Accel                uint32
	Capabilities                  uint16
	Reserved                      [2]uint16
}

// Panel is the screen, held open.
type Panel struct {
	f    *os.File
	mem  []byte
	vari varInfo

	stride     int
	fbW, fbH   int
	rot        Orientation
	pageBytes  int
	pages      int
	back, page int

	// Width and Height are what a person sees, at this rotation.
	Width, Height int

	// clip is what drawing is being confined to, empty for all of it.
	clip image.Rectangle

	// BlankErr is what the unblank said, so a dark panel is distinguishable from a drawing mistake.
	BlankErr error
}

// Open takes the panel. Anything else driving the display has to be stopped first.
func Open(path string, rot Orientation) (*Panel, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("screen: open %s: %w", path, err)
	}

	p := &Panel{f: f, rot: rot}

	// Some kernels refuse this on an already-lit panel, which is not a reason to give up on it.
	p.BlankErr = p.ioctlValue(iocBlank, blankUnblank)

	if err := p.ioctl(iocGetVScreenInfo, unsafe.Pointer(&p.vari)); err != nil {
		f.Close()
		return nil, fmt.Errorf("screen: FBIOGET_VSCREENINFO: %w", err)
	}
	var fx fixInfo
	if err := p.ioctl(iocGetFScreenInfo, unsafe.Pointer(&fx)); err != nil {
		f.Close()
		return nil, fmt.Errorf("screen: FBIOGET_FSCREENINFO: %w", err)
	}
	if p.vari.BitsPerPixel != 32 {
		f.Close()
		return nil, fmt.Errorf("screen: panel is %d bpp, this expects 32", p.vari.BitsPerPixel)
	}

	p.stride = int(fx.LineLength)
	p.fbW, p.fbH = int(p.vari.Xres), int(p.vari.Yres)
	p.Width, p.Height = rot.Size(p.fbW, p.fbH)
	p.pageBytes = p.stride * p.fbH

	size := int(fx.SmemLen)
	if size == 0 {
		size = p.stride * int(p.vari.YresVirtual)
	}
	if size < p.pageBytes {
		f.Close()
		return nil, fmt.Errorf("screen: %d bytes of memory for a %d byte page", size, p.pageBytes)
	}

	mem, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("screen: mmap %d bytes: %w", size, err)
	}
	p.mem = mem

	// YresVirtual, not the mapping: only pages the controller can be panned to count.
	p.pages = max(min(size/p.pageBytes, int(p.vari.YresVirtual)/p.fbH), 1)

	// Drawing into the live page tears.
	p.page = int(p.vari.Yoffset) / p.fbH
	p.back = (p.page + 1) % p.pages

	return p, nil
}

// Close leaves on screen whatever was drawn.
func (p *Panel) Close() error {
	if p.mem != nil {
		syscall.Munmap(p.mem)
		p.mem = nil
	}
	return p.f.Close()
}

// Pages is how many buffers can be panned between. One means every draw is visible as it happens.
func (p *Panel) Pages() int { return p.pages }

// Viewed turns a panel position into the coordinates drawing uses, so a touch lands where the pixel
// was drawn.
func (p *Panel) Viewed(px, py int) (x, y int) { return p.rot.Viewed(px, py, p.fbW, p.fbH) }

// Clip confines drawing to a rectangle, which is how a repaint costs the part that changed rather
// than the screen. An empty rectangle accepts all of it.
func (p *Panel) Clip(x, y, w, h int) {
	p.clip = image.Rect(x, y, x+w, y+h)
}

// clipped reports whether a viewed pixel is being accepted.
func (p *Panel) clipped(x, y int) bool {
	return !p.clip.Empty() && !image.Pt(x, y).In(p.clip)
}

// Set paints one viewed pixel into the page being drawn. Anything outside the panel is dropped, so
// a caller may describe more than fits.
func (p *Panel) Set(x, y int, c Color) {
	if x < 0 || y < 0 || x >= p.Width || y >= p.Height || p.clipped(x, y) {
		return
	}
	px, py := p.rot.Panel(x, y, p.fbW, p.fbH)

	at := p.back*p.pageBytes + py*p.stride + px*4
	if at < 0 || at+4 > len(p.mem) {
		return
	}

	// RGBA in memory. The driver advertises r16 g8 b0, which on a little-endian machine would put
	// blue first, but a red patch comes out blue that way — so this is what the panel shows, not
	// what it says.
	p.mem[at+0] = c.R
	p.mem[at+1] = c.G
	p.mem[at+2] = c.B
	p.mem[at+3] = c.A
}

// At reads a viewed pixel back off the page being drawn, which antialiasing needs: the edge of a
// curve is a blend with whatever is already there.
func (p *Panel) At(x, y int) Color {
	if x < 0 || y < 0 || x >= p.Width || y >= p.Height {
		return Color{}
	}
	px, py := p.rot.Panel(x, y, p.fbW, p.fbH)

	at := p.back*p.pageBytes + py*p.stride + px*4
	if at < 0 || at+4 > len(p.mem) {
		return Color{}
	}
	return Color{R: p.mem[at+0], G: p.mem[at+1], B: p.mem[at+2], A: p.mem[at+3]}
}

// FillRect paints a viewed rectangle. Rotation means a viewed row is not always a run of memory, so
// this is pixel by pixel rather than a copy.
func (p *Panel) FillRect(x, y, w, h int, c Color) {
	for iy := max(y, 0); iy < min(y+h, p.Height); iy++ {
		for ix := max(x, 0); ix < min(x+w, p.Width); ix++ {
			p.Set(ix, iy, c)
		}
	}
}

// Fill paints the whole page one colour.
func (p *Panel) Fill(c Color) {
	page := p.mem[p.back*p.pageBytes : (p.back+1)*p.pageBytes]

	// One row, then copied out: far faster than writing each pixel.
	row := page[:p.fbW*4]
	for x := range p.fbW {
		row[x*4+0] = c.R
		row[x*4+1] = c.G
		row[x*4+2] = c.B
		row[x*4+3] = c.A
	}
	for y := 1; y < p.fbH; y++ {
		copy(page[y*p.stride:y*p.stride+p.fbW*4], row)
	}
}

// Flip shows what was drawn and turns the page.
func (p *Panel) Flip() error {
	if p.pages > 1 {
		p.vari.Yoffset = uint32(p.back * p.fbH)
	} else {
		p.vari.Yoffset = 0
	}
	p.vari.Activate = 0 // FB_ACTIVATE_NOW

	if err := p.ioctl(iocPanDisplay, unsafe.Pointer(&p.vari)); err != nil {
		return fmt.Errorf("screen: FBIOPAN_DISPLAY: %w", err)
	}

	if p.pages > 1 {
		p.page = p.back
		p.back = (p.back + 1) % p.pages
	}
	return nil
}

// Info describes the panel, for logs.
func (p *Panel) Info() string {
	s := fmt.Sprintf("%dx%d %s (fb %dx%d, stride %d, %d bpp, %d pages)",
		p.Width, p.Height, p.rot, p.fbW, p.fbH, p.stride, p.vari.BitsPerPixel, p.pages)
	if p.BlankErr != nil {
		s += fmt.Sprintf(", unblank: %v", p.BlankErr)
	}
	return s
}

func (p *Panel) ioctl(req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, p.f.Fd(), req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

func (p *Panel) ioctlValue(req, v uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, p.f.Fd(), req, v); e != 0 {
		return e
	}
	return nil
}

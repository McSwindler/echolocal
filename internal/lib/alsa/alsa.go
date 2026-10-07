//go:build !board_doppler

// Package alsa captures PCM audio by driving the /dev/snd ioctl interface directly.
//
// This exists instead of binding tinyalsa through cgo. The kernel ioctl ABI is stable and
// the subset a capture-only client needs is small, so the whole thing is a few hundred
// lines of Go and echod stays a static, cgo-free binary — no NDK image, no C toolchain.
//
// Several of these structures embed the kernel's unsigned long, so their size — and with it the
// ioctl number, which encodes that size — depends on the word size. The layouts below derive from
// it rather than assuming one, because a mismatch is not a subtle bug: the kernel rejects the call
// outright, or worse, reads the wrong bytes.
package alsa

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// Capture is an open PCM capture stream.
type Capture struct {
	f          *os.File
	cfg        Config
	frameBytes int
}

// Open configures and starts a capture stream. The open is non-blocking so a device someone else
// holds returns ErrBusy instead of waiting; reads block normally once it is ours.
func Open(card, device int, cfg Config) (*Capture, error) {
	path := fmt.Sprintf("/dev/snd/pcmC%dD%dc", card, device)
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if errors.Is(err, syscall.EBUSY) || errors.Is(err, syscall.EAGAIN) {
			return nil, fmt.Errorf("%w: %s", ErrBusy, path)
		}
		return nil, err
	}
	if err := clearNonBlock(f); err != nil {
		_ = f.Close()
		return nil, err
	}

	var p hwParams
	p.init()
	p.setMask(paramAccess, accessRWInterleaved)
	p.setMask(paramFormat, cfg.Format)
	p.setMask(paramSubformat, subformatStd)
	p.setInterval(paramSampleBits, uint32(cfg.Bits))
	p.setInterval(paramFrameBits, uint32(cfg.Bits*cfg.Channels))
	p.setInterval(paramChannels, uint32(cfg.Channels))
	p.setInterval(paramRate, uint32(cfg.Rate))
	p.setInterval(paramPeriodSize, uint32(cfg.PeriodSize))
	p.setInterval(paramPeriods, uint32(cfg.Periods))

	if err := ioctl(f.Fd(), ioctlHwParams, unsafe.Pointer(&p)); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("hw_params: %w", err)
	}
	if err := granted(&p, cfg); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := ioctlArgless(f.Fd(), ioctlPrepare); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("prepare: %w", err)
	}
	if err := ioctlArgless(f.Fd(), ioctlStart); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("start: %w", err)
	}

	return &Capture{
		f:          f,
		cfg:        cfg,
		frameBytes: cfg.Channels * cfg.Bits / 8,
	}, nil
}

// FrameBytes is the size of one interleaved frame across all channels.
func (c *Capture) FrameBytes() int { return c.frameBytes }

// Read fills buf with whole frames and returns the number of bytes read. On overrun the
// stream is re-prepared and restarted, and the call reports how many bytes were lost.
func (c *Capture) Read(buf []byte) (int, error) {
	frames := len(buf) / c.frameBytes
	if frames == 0 {
		return 0, fmt.Errorf("buffer smaller than one frame (%d bytes)", c.frameBytes)
	}
	x := xferi{
		buf:    uintptr(unsafe.Pointer(&buf[0])),
		frames: uintptr(frames),
	}
	if err := ioctl(c.f.Fd(), ioctlReadi, unsafe.Pointer(&x)); err != nil {
		if err == syscall.EPIPE {
			_ = ioctlArgless(c.f.Fd(), ioctlPrepare)
			_ = ioctlArgless(c.f.Fd(), ioctlStart)
			return 0, ErrOverrun
		}
		return 0, err
	}
	return int(x.result) * c.frameBytes, nil
}

func (c *Capture) Close() error {
	_ = ioctlArgless(c.f.Fd(), ioctlDrop)
	return c.f.Close()
}

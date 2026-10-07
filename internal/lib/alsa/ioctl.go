package alsa

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"syscall"
	"unsafe"
)

// longSize is the kernel's unsigned long, which is the word size: 8 on arm64, 4 on a 32-bit kernel.
// The ioctl numbers below encode the size of the struct they carry, so a wrong answer here is not a
// subtle bug — the kernel rejects every call.
const longSize = strconv.IntSize / 8

// struct snd_pcm_hw_params: flags, 8 masks of 32 bytes, 21 intervals of 12, six 32-bit scalars,
// fifo_size as an unsigned long, reserved[64]. Only fifo_size is word sized, so all the offsets
// hold on either ABI and only the total differs: 604 on 32-bit, 608 here.
const (
	maskOff     = 4
	maskSize    = 32
	intervalOff = maskOff + 8*maskSize // 260
	intervalLen = 12
	rmaskOff    = intervalOff + 21*intervalLen // 512
	infoOff     = rmaskOff + 8

	hwParamsSize = rmaskOff + 6*4 + longSize + 64 // 608
)

// Parameter indices. Masks are 0..2, intervals 8..19.
const (
	paramAccess    = 0
	paramFormat    = 1
	paramSubformat = 2
	firstMask      = paramAccess
	lastMask       = paramSubformat

	paramSampleBits = 8
	paramFrameBits  = 9
	paramChannels   = 10
	paramRate       = 11
	paramPeriodSize = 13
	paramPeriods    = 15
	firstInterval   = paramSampleBits
	lastInterval    = 19
)

const (
	accessRWInterleaved = 3
	subformatStd        = 0

	// FormatS16_LE is 16-bit little endian, the only format the playback codec accepts.
	// And the only format Gen 1 Echo capture accepts.
	FormatS16_LE = 2

	// FormatS24_3LE is 24 bits packed into 3 bytes — the only format the Echo Dot's
	// capture codec accepts.
	FormatS24_3LE = 32
)

// ioc builds an ioctl request number the same way asm-generic/ioctl.h does.
func ioc(dir, typ, nr, size uintptr) uintptr {
	return dir<<30 | size<<16 | typ<<8 | nr
}

var (
	ioctlHwParams = ioc(3, 'A', 0x11, hwParamsSize)
	ioctlPrepare  = ioc(0, 'A', 0x40, 0)
	ioctlStart    = ioc(0, 'A', 0x42, 0)
	ioctlDrop     = ioc(0, 'A', 0x43, 0)
	ioctlReadi    = ioc(2, 'A', 0x51, xferiSize)
)

type hwParams [hwParamsSize]byte

func (p *hwParams) set(off int, v uint32) { binary.LittleEndian.PutUint32(p[off:], v) }

// init opens every parameter to its full range so the driver can narrow them.
func (p *hwParams) init() {
	for i := range p {
		p[i] = 0
	}
	for m := firstMask; m <= lastMask; m++ {
		off := maskOff + m*maskSize
		p.set(off, ^uint32(0))
		p.set(off+4, ^uint32(0))
	}
	for n := firstInterval; n <= lastInterval; n++ {
		off := intervalOff + (n-firstInterval)*intervalLen
		p.set(off, 0)
		p.set(off+4, ^uint32(0))
	}
	p.set(rmaskOff, ^uint32(0))
	p.set(infoOff, ^uint32(0))
}

func (p *hwParams) setMask(param, bit int) {
	off := maskOff + param*maskSize
	for i := 0; i < maskSize/4; i++ {
		p.set(off+i*4, 0)
	}
	p.set(off+(bit>>5)*4, 1<<uint(bit&31))
}

// setInterval pins a parameter to one value. Bit 2 of the flags word is "integer".
func (p *hwParams) setInterval(param int, v uint32) {
	off := intervalOff + (param-firstInterval)*intervalLen
	p.set(off, v)
	p.set(off+4, v)
	p.set(off+8, 1<<2)
}

// interval reads a value back. The ioctl refines what it was given and returns what the hardware will
// actually do, which is not always what was asked for — and writing in a period the hardware is not
// using sounds broken rather than merely gappy.
func (p *hwParams) interval(param int) uint32 {
	off := intervalOff + (param-firstInterval)*intervalLen
	return binary.LittleEndian.Uint32(p[off:])
}

// snd_xferi: a signed long, a pointer and an unsigned long. Every member is word sized, so the struct
// has to be built from word-sized types — Go's int is the kernel's long on both ABIs. Writing result
// as int64 lays the pointer out four bytes late on a 32-bit kernel, which reads as EFAULT.
type xferi struct {
	result int
	buf    uintptr
	frames uintptr
}

const xferiSize = 3 * longSize

type Config struct {
	Channels   int
	Rate       int
	Format     int
	Bits       int // physical bits per sample: 24 for S24_3LE
	PeriodSize int
	Periods    int
}

func ioctl(fd, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

func ioctlArgless(fd, req uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, 0); e != 0 {
		return e
	}
	return nil
}

// ErrOverrun means the hardware ring wrapped before we read it; audio was lost.
var ErrOverrun = fmt.Errorf("alsa: capture overrun")

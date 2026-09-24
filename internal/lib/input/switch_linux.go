package input

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/unix"
)

// EVIOCGBIT(EV_SW, n) = _IOR('E', 0x20+EV_SW, n), EVIOCGSW(n) = _IOR('E', 0x1b, n), SW_MAX = 0x10.
const (
	swBytes     = 8
	eviocgbitSW = 2<<30 | swBytes<<16 | 'E'<<8 | (0x20 + EvSw)
	eviocgsw    = 2<<30 | swBytes<<16 | 'E'<<8 | 0x1b
)

func (d *Device) switchBits(req uintptr) (uint64, error) {
	var b [swBytes]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, d.f.Fd(), req, uintptr(unsafe.Pointer(&b[0]))); errno != 0 {
		return 0, errno
	}
	return binary.LittleEndian.Uint64(b[:]), nil
}

// HasSwitch reports whether this node carries a switch. A node without one answers EVIOCGSW with
// every bit clear.
func (d *Device) HasSwitch(code uint16) bool {
	bits, err := d.switchBits(eviocgbitSW)
	return err == nil && bits&(1<<code) != 0
}

// Switch is where a switch is now. A switch reports only when it moves.
func (d *Device) Switch(code uint16) (bool, error) {
	bits, err := d.switchBits(eviocgsw)
	if err != nil {
		return false, err
	}
	return bits&(1<<code) != 0, nil
}

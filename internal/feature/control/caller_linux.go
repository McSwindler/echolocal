package control

import (
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// shell is the uid adb shell runs as.
const shell = 2000

// caller is who is on the other end, and whether they may drive the device. An abstract socket has no
// permissions of its own: any process on the device can reach it.
func caller(conn net.Conn) (int, bool) {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return -1, false
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return -1, false
	}

	var cred *unix.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil || credErr != nil {
		return -1, false
	}

	uid := int(cred.Uid)
	return uid, uid == 0 || uid == shell || uid == os.Getuid()
}

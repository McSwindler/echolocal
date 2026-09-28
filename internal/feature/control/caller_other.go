//go:build !linux

package control

import "net"

func caller(net.Conn) (int, bool) { return -1, true }

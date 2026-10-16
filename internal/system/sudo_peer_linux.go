//go:build linux

package system

import (
	"errors"
	"net"

	"golang.org/x/sys/unix"
)

// askpassPeer obtains identity from the kernel, never from client-supplied data.
func askpassPeer(conn net.Conn) (uid, pid, session int, err error) {
	socket, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, 0, 0, errors.New("askpass requires a Unix socket")
	}
	raw, err := socket.SyscallConn()
	if err != nil {
		return 0, 0, 0, err
	}
	var credential *unix.Ucred
	var queryErr error
	if err = raw.Control(func(fd uintptr) {
		credential, queryErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return 0, 0, 0, err
	}
	if queryErr != nil {
		return 0, 0, 0, queryErr
	}
	session, err = unix.Getsid(int(credential.Pid))
	return int(credential.Uid), int(credential.Pid), session, err
}

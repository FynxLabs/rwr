//go:build darwin

package system

import (
	"errors"
	"net"

	"golang.org/x/sys/unix"
)

func askpassPeer(conn net.Conn) (uid, pid, session int, err error) {
	socket, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, 0, 0, errors.New("askpass requires a Unix socket")
	}
	raw, err := socket.SyscallConn()
	if err != nil {
		return 0, 0, 0, err
	}
	var credential *unix.Xucred
	var queryErr error
	if err = raw.Control(func(fd uintptr) {
		credential, queryErr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if queryErr == nil {
			pid, queryErr = unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
		}
	}); err != nil {
		return 0, 0, 0, err
	}
	if queryErr != nil {
		return 0, 0, 0, queryErr
	}
	session, err = unix.Getsid(pid)
	return int(credential.Uid), pid, session, err
}

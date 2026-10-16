//go:build !linux && !darwin && !windows

package system

import (
	"errors"
	"net"
)

func askpassPeer(net.Conn) (int, int, int, error) {
	return 0, 0, 0, errors.New("askpass peer validation is unsupported on this platform")
}

package tui

import "golang.org/x/sys/unix"

const (
	popupGetTermios = unix.TIOCGETA
	popupSetTermios = unix.TIOCSETA
	// FIONREAD = _IOR('f', 127, int), from Darwin's sys/filio.h. x/sys/unix
	// does not export this request; the payload is the platform's 32-bit int.
	popupInputQueued = 0x4004667f
)

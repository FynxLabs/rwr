package tui

import "golang.org/x/sys/unix"

const (
	popupGetTermios  = unix.TCGETS
	popupSetTermios  = unix.TCSETS
	popupInputQueued = unix.TIOCINQ
)

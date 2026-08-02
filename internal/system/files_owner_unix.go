//go:build !windows

package system

import (
	"os"
	"syscall"
)

// fileOwnedByEUID reports whether the file belongs to the effective user, so
// targetMode only inherits permissions from files the invoking user (or root,
// when elevated) actually owns.
func fileOwnedByEUID(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return int(st.Uid) == os.Geteuid()
}

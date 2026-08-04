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

// ownedByRootOrEUID reports whether the file or directory belongs to root or
// to the effective user. A provider directory owned by any other account is
// rewritable by that account regardless of its mode bits - a directory's owner
// can always chmod it open.
func ownedByRootOrEUID(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return st.Uid == 0 || int(st.Uid) == os.Geteuid()
}

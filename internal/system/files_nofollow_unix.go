//go:build !windows

package system

import "syscall"

// noFollow makes an open refuse a symlink as the final path component. The
// EXDEV copy fallback writes to a target in a directory that may be shared
// (staging fell back to the system temp dir because the target's own directory
// was not writable), where any local user can plant a symlink at the target
// name and redirect the write.
const noFollow = syscall.O_NOFOLLOW

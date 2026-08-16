//go:build !windows

package system

import (
	"os"
	"syscall"
)

// openFileNoFollow refuses a symlink as the final path component. The
// EXDEV copy fallback writes to a target in a directory that may be shared
// (staging fell back to the system temp dir because the target's own directory
// was not writable), where any local user can plant a symlink at the target
// name and redirect the write.
func openFileNoFollow(path string, flags int, mode os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flags|syscall.O_NOFOLLOW, mode) // #nosec G304 -- caller supplies an operator-controlled path; O_NOFOLLOW is the boundary
}

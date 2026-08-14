//go:build !windows

package system

import (
	"os/exec"
	"testing"
)

// Sanity: the helper is wired to real process groups, not silently a no-op.
//
// Unix-only, and in its own file rather than guarded by a runtime check:
// SysProcAttr.Setpgid does not exist in the Windows syscall package, so a
// runtime skip still fails to compile there. `GOOS=windows go build ./...`
// does not catch that, because it does not build tests - `go vet ./...` does,
// which is what CI runs.
func TestCommandsGetTheirOwnProcessGroup(t *testing.T) {
	built := exec.Command("true")
	intoOwnProcessGroup(built)
	if built.SysProcAttr == nil || !built.SysProcAttr.Setpgid {
		t.Fatal("commands are not placed in their own process group")
	}
}

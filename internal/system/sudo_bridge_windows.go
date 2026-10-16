//go:build windows

package system

import (
	"os/exec"

	"github.com/freehold-digital/rwr/internal/types"
)

func installSudoBridge(_ *exec.Cmd, _ types.Command) (*sudoBridge, error) { return nil, nil }

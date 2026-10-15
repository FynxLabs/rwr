package system

import (
	"github.com/freehold-digital/rwr/internal/types"
	"os/exec"
)

func installSudoBridge(_ *exec.Cmd, _ types.Command) (func(), error) { return func() {}, nil }

//go:build !linux && !darwin && !windows

package tui

import (
	"context"
	"fmt"
	"io"
	"os"
)

func stagePopupInput(_ context.Context, _, _ *os.File, _ io.Reader) error {
	return fmt.Errorf("supplied interactive input is unsupported on this platform; use non-interactive execution")
}

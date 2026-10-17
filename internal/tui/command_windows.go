//go:build windows

package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/charmbracelet/x/conpty"
	"golang.org/x/sys/windows"
)

type windowsPopupPTY struct {
	*conpty.ConPty
	process *os.Process
}

func startPopupPTY(ctx context.Context, cmd *exec.Cmd, width, height int) (popupPTY, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cmd.Stdin != nil {
		return nil, fmt.Errorf("Windows interactive commands with supplied stdin require non-interactive execution")
	}
	p, err := conpty.New(width, height, 0)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		closePopupIO(p)
		return nil, err
	}
	pid, handle, err := p.Spawn(cmd.Path, cmd.Args, &syscall.ProcAttr{Dir: cmd.Dir, Env: cmd.Env, Sys: cmd.SysProcAttr})
	if err != nil {
		_ = p.Close()
		return nil, err
	}
	process, err := os.FindProcess(pid)
	_ = windows.CloseHandle(windows.Handle(handle))
	if err != nil {
		_ = p.Close()
		return nil, err
	}
	cmd.Process = process
	return &windowsPopupPTY{ConPty: p, process: process}, nil
}

func (p *windowsPopupPTY) Wait() error {
	state, err := p.process.Wait()
	if err != nil {
		return err
	}
	if !state.Success() {
		return &exec.ExitError{ProcessState: state}
	}
	return nil
}
func (p *windowsPopupPTY) Kill() error { return p.process.Kill() }

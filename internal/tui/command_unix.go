//go:build !windows

package tui

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

type unixPopupPTY struct {
	*os.File
	cmd    *exec.Cmd
	mu     sync.Mutex
	closed bool
	slave  *os.File
}

func startPopupPTY(ctx context.Context, cmd *exec.Cmd, width, height int) (popupPTY, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, slave, err := pty.Open()
	if err != nil {
		return nil, err
	}
	// pty.Open uses os.NewFile on a blocking descriptor. Reopen a duplicate
	// as pollable so closing the popup interrupts a pending output read.
	fd, err := unix.FcntlInt(file.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		closePopupIO(file)
		closePopupIO(slave)
		return nil, err
	}
	if err := syscall.SetNonblock(fd, true); err != nil {
		closePopupIO(file)
		closePopupIO(slave)
		if closeErr := syscall.Close(fd); closeErr != nil {
			return nil, closeErr
		}
		return nil, err
	}
	closePopupIO(file)
	file = os.NewFile(uintptr(fd), "rwr-popup-pty") // #nosec G115 -- fd is a successful non-negative file descriptor
	p := &unixPopupPTY{File: file, slave: slave, cmd: cmd}
	if err := p.Resize(width, height); err != nil {
		closePopupIO(p)
		return nil, err
	}
	if cmd.Stdin != nil {
		if err := stagePopupInput(ctx, file, slave, cmd.Stdin); err != nil {
			closePopupIO(p)
			return nil, err
		}
	}
	cmd.Stdout, cmd.Stderr = slave, slave
	cmd.Stdin = slave
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Setsid creates the process group; combining it with Setpgid fails on
	// real RWR commands, which already request a separate process group.
	cmd.SysProcAttr.Setpgid, cmd.SysProcAttr.Pgid = false, 0
	cmd.SysProcAttr.Setsid, cmd.SysProcAttr.Setctty, cmd.SysProcAttr.Ctty = true, true, 1
	if err := ctx.Err(); err != nil {
		closePopupIO(p)
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		closePopupIO(p)
		return nil, err
	}
	return p, nil
}

func (p *unixPopupPTY) Resize(width, height int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return os.ErrClosed
	}
	return pty.Setsize(p.File, &pty.Winsize{Cols: uint16(width), Rows: uint16(height)}) // #nosec G115 -- bounded terminal dimensions
}

func (p *unixPopupPTY) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	closePopupIO(p.slave)
	return p.File.Close()
}

func (p *unixPopupPTY) Wait() error {
	err := p.cmd.Wait()
	// Release the parent's slave endpoint so the output reader sees EOF.
	closePopupIO(p.slave)
	return err
}
func (p *unixPopupPTY) Kill() error { return syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL) }

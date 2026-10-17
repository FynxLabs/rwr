//go:build !windows

package tui

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

type unixPopupPTY struct {
	*os.File
	cmd      *exec.Cmd
	mu       sync.Mutex
	closed   bool
	slave    *os.File
	input    *os.File
	inputTTY *os.File
}

func startPopupPTY(cmd *exec.Cmd, width, height int) (popupPTY, error) {
	file, slave, err := pty.Open()
	if err != nil {
		return nil, err
	}
	// pty.Open uses os.NewFile on a blocking descriptor. Reopen a duplicate
	// as pollable so closing the popup interrupts a pending output read.
	fd, err := syscall.Dup(int(file.Fd()))
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
	initial := cmd.Stdin
	cmd.Stdout, cmd.Stderr = slave, slave
	cmd.Stdin = slave
	var inputRead *os.File
	if initial != nil {
		inputRead, p.input, err = os.Pipe()
		if err != nil {
			closePopupIO(p)
			return nil, err
		}
		cmd.Stdin = inputRead
		// A separate open keeps the child's slave blocking while allowing
		// the parent's stdin bridge to be interrupted on process exit.
		p.inputTTY, err = os.OpenFile(slave.Name(), os.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0) // #nosec G304 -- PTY device path allocated by pty.Open, never operator input
		if err != nil {
			closePopupIO(inputRead)
			closePopupIO(p)
			return nil, err
		}
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Setsid creates the process group; combining it with Setpgid fails on
	// real RWR commands, which already request a separate process group.
	cmd.SysProcAttr.Setpgid, cmd.SysProcAttr.Pgid = false, 0
	cmd.SysProcAttr.Setsid, cmd.SysProcAttr.Setctty, cmd.SysProcAttr.Ctty = true, true, 1
	err = cmd.Start()
	if inputRead != nil {
		closePopupIO(inputRead)
	}
	if err != nil {
		closePopupIO(p)
		return nil, err
	}
	if initial != nil {
		// Pre-fed sudo passwords go through a pipe, never the PTY input
		// queue where terminal echo could expose them before sudo starts.
		go func() {
			if _, err := io.Copy(p.input, initial); err != nil {
				return
			}
			if _, err := io.Copy(p.input, p.inputTTY); err != nil {
				return
			}
		}()
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
	if p.input != nil {
		closePopupIO(p.input)
	}
	if p.inputTTY != nil {
		closePopupIO(p.inputTTY)
	}
	return p.File.Close()
}

func (p *unixPopupPTY) Wait() error {
	err := p.cmd.Wait()
	// Release the parent's slave endpoint so the output reader sees EOF.
	closePopupIO(p.slave)
	if p.input != nil {
		closePopupIO(p.input)
	}
	if p.inputTTY != nil {
		closePopupIO(p.inputTTY)
	}
	return err
}
func (p *unixPopupPTY) Kill() error { return syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL) }

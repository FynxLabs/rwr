//go:build linux || darwin

package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// stagePopupInput queues complete initial lines before the child or the popup
// input pump starts. Echo stays disabled until every byte has crossed the line
// discipline; restoring it immediately after Write could echo deferred input.
func stagePopupInput(ctx context.Context, master, slave *os.File, source io.Reader) (err error) {
	const maxInitialInput = 4096
	data, err := io.ReadAll(io.LimitReader(source, maxInitialInput+1))
	defer clear(data)
	if err != nil {
		return fmt.Errorf("reading initial tool input: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if len(data) > maxInitialInput || !bytes.HasSuffix(data, []byte{'\n'}) {
		return fmt.Errorf("initial interactive input must be newline-terminated and at most %d bytes; use non-interactive execution for other input", maxInitialInput)
	}
	for _, b := range data {
		if b == 127 || b < 32 && b != '\n' && b != '\t' {
			return fmt.Errorf("initial interactive input contains terminal control characters; use non-interactive execution")
		}
	}
	fd := int(slave.Fd())
	original, err := unix.IoctlGetTermios(fd, popupGetTermios)
	if err != nil {
		return fmt.Errorf("reading tool terminal settings: %w", err)
	}
	quiet := *original
	quiet.Lflag &^= unix.ECHO | unix.ECHONL
	if err := unix.IoctlSetTermios(fd, popupSetTermios, &quiet); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, unix.IoctlSetTermios(fd, popupSetTermios, original)) }()
	deadline := time.Now().Add(time.Second)
	if err := master.SetWriteDeadline(deadline); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, master.SetWriteDeadline(time.Time{})) }()
	if _, err := master.Write(data); err != nil {
		return fmt.Errorf("staging initial tool input: %w", err)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		queued, err := unix.IoctlGetInt(fd, popupInputQueued)
		if err != nil {
			return err
		}
		if queued == len(data) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("initial input does not fit the tool's terminal buffer intact; use non-interactive execution")
		}
		time.Sleep(time.Millisecond)
	}
}

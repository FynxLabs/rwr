package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/types"
)

type popupPTY interface {
	io.ReadWriteCloser
	Resize(int, int) error
	Wait() error
	Kill() error
}

type popupSize struct{ width, height int }
type commandPopup struct {
	req      reporting.TerminalReq
	terminal *vt.SafeEmulator
	input    chan tea.Msg
	resize   chan popupSize
	ctx      context.Context
	cancel   context.CancelFunc
}
type commandDone struct {
	popup *commandPopup
	err   error
}

func (m *Model) commandSize() popupSize {
	return popupSize{width: m.promptContentWidth(), height: max(1, min(16, m.height-10))}
}

func (m *Model) initCommand(req reporting.TerminalReq) tea.Cmd {
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	size := m.commandSize()
	p := &commandPopup{req: req, ctx: ctx, cancel: cancel, terminal: vt.NewSafeEmulator(size.width, size.height), input: make(chan tea.Msg, 256), resize: make(chan popupSize, 1)}
	m.command = p
	m.state = Prompting
	m.pinned, m.cursor, m.scrollOffset = false, m.live, 0
	return func() tea.Msg { return commandDone{popup: p, err: p.run(size)} }
}

func (p *commandPopup) run(size popupSize) error {
	if !reporting.TryClaim(p.req.Claim) {
		p.cancel()
		return reporting.ErrPromptUnavailable
	}
	err := p.runClaimed(size)
	p.req.Done <- err
	return err
}

func (p *commandPopup) runClaimed(size popupSize) error {
	defer p.cancel()
	if err := p.ctx.Err(); err != nil {
		return err
	}
	cmd := p.req.Cmd
	cmd.Stdout, cmd.Stderr = nil, nil
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	for i := len(cmd.Env) - 1; i >= 0; i-- {
		if strings.HasPrefix(cmd.Env[i], "TERM=") {
			cmd.Env = append(cmd.Env[:i], cmd.Env[i+1:]...)
		}
	}
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	inputPipe, ok := p.terminal.InputPipe().(io.Closer)
	if !ok {
		return fmt.Errorf("terminal emulator has no closable input")
	}
	defer closePopupIO(inputPipe)
	child, err := startPopupPTY(p.ctx, cmd, size.width, size.height)
	if err != nil {
		return err
	}
	defer closePopupIO(child)
	if p.req.Started != nil {
		p.req.Started(cmd)
	}

	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-p.ctx.Done():
		case <-reporting.TerminalLost():
		case <-stopped:
			return
		}
		p.cancel()
		select {
		case <-stopped:
			return
		default:
		}
		if err := child.Kill(); err != nil {
			log.Debug("Tool already exited during cancellation")
		}
		closePopupIO(child)
		closePopupIO(inputPipe)
	}()
	go func() {
		// Drain emulator input independently of the child. SendKey holds the
		// emulator lock while writing; a tool that stops reading must never
		// block View on that lock and prevent cancellation of the popup.
		chunks := make(chan []byte, 16)
		go func() {
			defer close(chunks)
			buf := make([]byte, 4096)
			for {
				n, err := p.terminal.Read(buf)
				if n > 0 {
					chunk := append([]byte(nil), buf[:n]...)
					select {
					case chunks <- chunk:
					case <-p.ctx.Done():
						return
					default:
						log.Warn("Tool input buffer full; canceling the tool")
						p.cancel()
						return
					}
				}
				if err != nil {
					return
				}
			}
		}()
		for chunk := range chunks {
			if _, err := child.Write(chunk); err != nil {
				p.cancel()
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case size := <-p.resize:
				if err := child.Resize(size.width, size.height); err != nil {
					log.Debug("Tool terminal closed before resize")
				}
			case msg := <-p.input:
				switch msg := msg.(type) {
				case tea.KeyPressMsg:
					p.terminal.SendKey(uv.KeyPressEvent(msg))
				case tea.PasteMsg:
					p.terminal.Paste(msg.Content)
				}
			}
		}
	}()
	// Hide the embedded emulator's promoted WriteString method: it bypasses
	// SafeEmulator's lock when io.WriteString is used by the secret scrubber.
	var output io.Writer = struct{ io.Writer }{p.terminal}
	if !p.req.Private {
		if log := reporting.CommandOutputWriter(reporting.SrcStdout); log != nil {
			output = io.MultiWriter(output, log)
		}
	}
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		writer := types.NewSecretWriter(output, append(types.ResolvedSecrets(), p.req.Secrets...))
		if _, err := io.Copy(writer, child); err != nil {
			log.Debug("Tool terminal output stream closed")
		}
		writer.Flush()
	}()
	err = child.Wait()
	select {
	case <-outputDone:
	case <-time.After(500 * time.Millisecond):
		closePopupIO(child)
		<-outputDone
	}
	if p.ctx.Err() != nil {
		return reporting.ErrPromptCancelled
	}
	return err
}

func closePopupIO(closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Debug("Tool terminal stream already closed")
	}
}

func (m *Model) resizeCommand() {
	size := m.commandSize()
	m.command.terminal.Resize(size.width, size.height)
	select {
	case <-m.command.resize:
	default:
	}
	m.command.resize <- size
}

func (m *Model) commandKey(msg tea.Msg) {
	if key, ok := msg.(tea.KeyPressMsg); ok && (key.String() == "ctrl+esc" || key.String() == "ctrl+c") {
		m.command.cancel()
		return
	}
	select {
	case m.command.input <- msg:
	default:
		log.Warn("Tool input queue full; canceling the tool")
		m.command.cancel()
	}
}

func (m *Model) viewCommandDialog(background string) (string, *tea.Cursor) {
	p := m.command
	body := style(m.theme.Modal).Bold(true).Render("INTERACTIVE TOOL · "+sanitizePromptText(filepath.Base(p.req.Cmd.Path))) + "\n\n" +
		p.terminal.Render() + "\n\n" + style(m.theme.Subtext).Render("Input goes to this tool · Ctrl-C cancel")
	pos := p.terminal.CursorPosition()
	return m.viewPromptDialog(background, body, &tea.Position{X: pos.X, Y: pos.Y + 2})
}

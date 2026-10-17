package system

import "os/exec"

// sudoBridge binds askpass access to the session of exactly one child command.
// Closing ready publishes session to handlers after Start has assigned its PID.
type sudoBridge struct {
	ready   chan struct{}
	session int
	stop    func()
}

func (b *sudoBridge) Run(command *exec.Cmd) error {
	if b == nil {
		return command.Run()
	}
	if err := command.Start(); err != nil {
		return err
	}
	b.Started(command)
	return command.Wait()
}

// Started binds askpass peers to the child session after either the captured
// executor or the popup PTY starts the process. It never reads child state early.
func (b *sudoBridge) Started(command *exec.Cmd) {
	if b != nil {
		b.session = command.Process.Pid
		close(b.ready)
	}
}

func (b *sudoBridge) Close() {
	if b != nil {
		b.stop()
	}
}

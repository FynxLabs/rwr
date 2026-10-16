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
	b.session = command.Process.Pid
	close(b.ready)
	return command.Wait()
}

func (b *sudoBridge) Close() {
	if b != nil {
		b.stop()
	}
}

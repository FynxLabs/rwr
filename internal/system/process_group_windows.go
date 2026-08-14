package system

import "os/exec"

// Windows has no process groups in the POSIX sense and no signals to send one.
// Killing the child is the whole of what is available here, which is what
// os/exec does by default when a command's context is cancelled.
func intoOwnProcessGroup(*exec.Cmd) {}

func terminateProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

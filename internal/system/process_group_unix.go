//go:build !windows

package system

import (
	"os/exec"
	"syscall"
)

// intoOwnProcessGroup puts the child in a process group of its own, so
// cancelling can kill the group rather than just the process rwr spawned.
//
// This matters for every command rwr actually runs: `brew install` is a shell
// that forks curl, git and the installer; `sudo pacman` is sudo with pacman
// underneath. Killing only the direct child orphans the real work, which keeps
// running with the terminal it inherited - the process rwr was asked to stop
// carries on writing to the operator's screen.
func intoOwnProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends a signal to the child's whole process group.
//
// A negative pid means "the group led by this pid", which is why the child had
// to be given its own group above: negating rwr's own group would kill rwr.
func killProcessGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, sig); err != nil {
		// The group may already be gone, or the child may have failed to get
		// its own group; fall back to the process itself rather than leaving
		// it running.
		return cmd.Process.Signal(sig)
	}
	return nil
}

// terminateProcessGroup is what a cancelled command runs.
//
// SIGKILL rather than SIGTERM, because the operator asked for the run to stop
// now and a package manager that traps SIGTERM to finish its transaction is
// exactly the thing that would ignore a polite request. WaitDelay in the
// caller bounds how long the pipes are held afterwards.
func terminateProcessGroup(cmd *exec.Cmd) error {
	return killProcessGroup(cmd, syscall.SIGKILL)
}

//go:build unix

package editor

import (
	"os/exec"
	"syscall"
)

// ownProcessGroup starts cmd in a process group of its own, so stopping it
// stops what it started too: the compiler make runs, the tests go test runs.
// Signalling the shell alone would leave those running on, orphaned.
func ownProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// interruptGroup asks cmd's group to stop, as C-c at a terminal does.
func interruptGroup(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

// killGroup stops cmd's group outright.
func killGroup(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

// sigintGroup sends C-c to cmd's group alone, for a shell that should go on
// after what it was running stops.
func sigintGroup(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
}

// ownSession starts cmd in a session of its own, with no controlling
// terminal. An interactive shell in a process group of nem's session would be
// a background job of nem's terminal, and stopped the moment it touched it.
// Being its own group leader, it is still stopped by signalling -pid.
func ownSession(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

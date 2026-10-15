//go:build !unix

package editor

import "os/exec"

// ownProcessGroup does nothing where there are no process groups to join.
func ownProcessGroup(*exec.Cmd) {}

// interruptGroup stops cmd; without process groups, what it started may
// outlive it.
func interruptGroup(cmd *exec.Cmd) { _ = cmd.Process.Kill() }

// killGroup stops cmd.
func killGroup(cmd *exec.Cmd) { _ = cmd.Process.Kill() }

// sigintGroup stops cmd; without signals there is no gentler way, and the
// shell goes with what it was running.
func sigintGroup(cmd *exec.Cmd) { _ = cmd.Process.Kill() }

// ownSession does nothing where there are no sessions.
func ownSession(*exec.Cmd) {}

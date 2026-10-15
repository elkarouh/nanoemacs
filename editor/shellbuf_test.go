package editor

import (
	"strings"
	"testing"

	"github.com/Borderliner/nem/text"
)

// M-t enters a shell, RET runs a line in it, and M-t again returns.
func TestShellToggleRunsCommands(t *testing.T) {
	e, scr := newTestEditor(t, "hello")
	e.shell = "/bin/sh"
	origin := e.Buf()
	press(t, e, "M-t")
	shell := e.Buf()
	if e.shellOf(shell) == nil {
		t.Fatalf("M-t did not enter a shell buffer")
	}
	for _, r := range "echo nem-shell-ok" {
		// A space is written SPC in a key spec, as the keymap has it.
		spec := string(r)
		if r == ' ' {
			spec = "SPC"
		}
		press(t, e, spec)
	}
	press(t, e, "RET")
	// The line typed is in the buffer once; the shell's answer makes twice.
	pump(t, e, scr, func() bool { return strings.Count(shell.String(), "nem-shell-ok") >= 2 })

	// What the shell printed is not editable.
	if err := shell.Insert(text.Pos{}, []rune("x")); err == nil {
		t.Errorf("inserting before the prompt should be refused")
	}

	press(t, e, "M-t")
	if e.Buf() != origin {
		t.Fatalf("M-t in the shell should return to the buffer it came from")
	}
	press(t, e, "M-t")
	if e.Buf() != shell {
		t.Fatalf("M-t should return to the same shell")
	}
}

// Typing at the prompt keeps its place when output arrives.
func TestShellOutputKeepsInputBeingTyped(t *testing.T) {
	e, _ := newTestEditor(t)
	press(t, e, "M-t")
	b := e.Buf()
	st := e.shellOf(b)
	for _, r := range "ls" {
		press(t, e, string(r))
	}
	e.shellFeed(b, st, "output\nprompt$ ")
	if got := b.String(); got != "output\nprompt$ ls" {
		t.Fatalf("buffer = %q", got)
	}
	if p := e.Active().Pt; p != b.End() {
		t.Errorf("point = %+v, want the end of the input %+v", p, b.End())
	}
	press(t, e, "M-p")
	if got := b.String(); got != "output\nprompt$ ls" {
		t.Errorf("M-p with no history changed the buffer: %q", got)
	}
}

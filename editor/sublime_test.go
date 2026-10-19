package editor

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// sublimeCalls records what open-in-sublime would have passed to subl, and
// makes it succeed or fail as the test wants. The real subl is never run:
// TestMain replaces it.
type sublimeCalls struct {
	args [][]string
	err  error
}

// fakeSubl installs a recording subl for the test and returns the record. The
// original is put back when the test ends.
func fakeSubl(t *testing.T) *sublimeCalls {
	t.Helper()
	c := &sublimeCalls{}
	prev := sublimeRun
	sublimeRun = func(args ...string) error {
		c.args = append(c.args, slices.Clone(args))
		return c.err
	}
	t.Cleanup(func() { sublimeRun = prev })
	return c
}

// Shift-F5 sends the file to Sublime at the line point is on, as FILE:LINE.
func TestShiftF5OpensTheBufferInSublimeAtPointsLine(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeSubl(t)
	p := visitFileForEmacs(t, e, "one\ntwo\nthree\n")
	e.active.Pt.Line = 2

	press(t, e, "S-<f5>")

	want := []string{p + ":3"}
	if len(c.args) != 1 || !slices.Equal(c.args[0], want) {
		t.Fatalf("subl called with %q, want one call with %q", c.args, want)
	}
}

// A modified buffer is not sent: Sublime would read the file from disk.
func TestShiftF5RefusesAModifiedBuffer(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeSubl(t)
	visitFileForEmacs(t, e, "saved\n")
	press(t, e, "x")

	press(t, e, "S-<f5>")

	if len(c.args) != 0 {
		t.Errorf("subl called for a modified buffer: %q", c.args)
	}
	wantEcho(t, e, "unsaved changes")
}

// A buffer with no file has nothing to send.
func TestShiftF5NeedsAFile(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeSubl(t)

	press(t, e, "S-<f5>")

	if len(c.args) != 0 {
		t.Errorf("subl called with no file: %q", c.args)
	}
	wantEcho(t, e, "no file")
}

// In a directory listing, Shift-F5 sends the directory, which Sublime opens as
// a folder.
func TestShiftF5OnAListingOpensTheDirectory(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeSubl(t)
	dir := t.TempDir()
	listed(t, e, dir)

	press(t, e, "S-<f5>")

	if len(c.args) != 1 || !slices.Equal(c.args[0], []string{dir}) {
		t.Fatalf("subl called with %q, want one call with the directory %q", c.args, dir)
	}
}

// A failure from subl reaches the command's caller.
func TestShiftF5ReportsAFailure(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeSubl(t)
	c.err = errors.New("subl: not found")
	visitFileForEmacs(t, e, "text\n")

	err := e.dispatch("open-in-sublime")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("dispatch error = %v, want subl's reason", err)
	}
}

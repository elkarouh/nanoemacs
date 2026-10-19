package editor

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// emacsCalls records what open-in-emacs would have sent to emacsclient, and
// makes it succeed or fail as the test wants. The real emacsclient is never
// run: TestMain replaces it.
type emacsCalls struct {
	args [][]string
	err  error
}

// fakeEmacsclient installs a recording emacsclient for the test and returns the
// record. The original is put back when the test ends.
func fakeEmacsclient(t *testing.T) *emacsCalls {
	t.Helper()
	c := &emacsCalls{}
	prev := emacsclientRun
	emacsclientRun = func(args ...string) error {
		c.args = append(c.args, slices.Clone(args))
		return c.err
	}
	t.Cleanup(func() { emacsclientRun = prev })
	return c
}

// visitFileForEmacs writes a file with the given text, visits it, and returns
// its absolute path.
func visitFileForEmacs(t *testing.T, e *Editor, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := e.OpenFile(p)
	if err != nil {
		t.Fatal(err)
	}
	e.active.Visit(b)
	return p
}

// Shift-F4 sends the file to Emacs at the line point is on, counted from one,
// and without waiting for Emacs to be finished with it.
func TestShiftF4OpensTheBufferInEmacsAtPointsLine(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeEmacsclient(t)
	p := visitFileForEmacs(t, e, "one\ntwo\nthree\n")
	e.active.Pt.Line = 2

	press(t, e, "S-<f4>")

	want := []string{"-c", "-n", "+3", p}
	if len(c.args) != 1 || !slices.Equal(c.args[0], want) {
		t.Fatalf("emacsclient called with %q, want one call with %q", c.args, want)
	}
}

// A file that has unsaved edits is not sent: Emacs would read the file from
// disk and show the older text. The buffer is left as it is.
func TestShiftF4RefusesAModifiedBuffer(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeEmacsclient(t)
	visitFileForEmacs(t, e, "saved\n")
	press(t, e, "x")

	press(t, e, "S-<f4>")

	if len(c.args) != 0 {
		t.Errorf("emacsclient called for a modified buffer: %q", c.args)
	}
	wantEcho(t, e, "unsaved changes")
}

// A buffer with no file, such as the scratch buffer, has nothing to send.
func TestShiftF4NeedsAFile(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeEmacsclient(t)

	press(t, e, "S-<f4>")

	if len(c.args) != 0 {
		t.Errorf("emacsclient called with no file: %q", c.args)
	}
	wantEcho(t, e, "no file")
}

// In a directory listing, Shift-F4 sends the directory, which Emacs shows in
// dired - the same thing open-externally does for the system app.
func TestShiftF4OnAListingOpensTheDirectory(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeEmacsclient(t)
	dir := t.TempDir()
	listed(t, e, dir)

	press(t, e, "S-<f4>")

	want := []string{"-c", "-n", dir}
	if len(c.args) != 1 || !slices.Equal(c.args[0], want) {
		t.Fatalf("emacsclient called with %q, want one call with %q", c.args, want)
	}
}

// When emacsclient fails - no server running, say - the reason reaches the
// command's caller rather than being swallowed.
func TestShiftF4ReportsAFailure(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeEmacsclient(t)
	c.err = errors.New("emacsclient: can't find socket; have you started the server?")
	visitFileForEmacs(t, e, "text\n")

	err := e.dispatch("open-in-emacs")
	if err == nil || !strings.Contains(err.Error(), "can't find socket") {
		t.Fatalf("dispatch error = %v, want emacsclient's reason", err)
	}
}

// Plain F4 is still the keyboard-macro key and must not reach emacsclient: the
// Shift-F4 binding is a separate key, not a change to F4.
func TestPlainF4DoesNotOpenInEmacs(t *testing.T) {
	e, _ := newTestEditor(t)
	c := fakeEmacsclient(t)
	visitFileForEmacs(t, e, "text\n")

	press(t, e, "<f4>")
	if len(c.args) != 0 {
		t.Fatalf("plain F4 reached emacsclient: %q", c.args)
	}
}

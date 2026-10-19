package editor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Borderliner/nem/command"
)

// Shift-F4 hands the current buffer to Emacs. It runs emacsclient, which talks
// to an Emacs that is already running as a server (M-x server-start, or
// emacs --daemon). Nothing is started here: if no server answers, the error
// says so and the buffer is left alone.

// emacsclientTimeout is how long emacsclient may take to hand over a file. It
// normally returns at once, since -n does not wait for the file to be closed.
const emacsclientTimeout = 5 * time.Second

// emacsclientRun runs emacsclient with args. It is a variable so the test
// binary can replace it, as systemOpen is replaced: a test run must never talk
// to a real Emacs on the developer's desktop. See TestMain.
var emacsclientRun = runEmacsclient

// runEmacsclient runs the emacsclient on the PATH with args and reports how it
// failed, if it did.
func runEmacsclient(args ...string) error {
	bin, err := exec.LookPath("emacsclient")
	if err != nil {
		return errors.New("emacsclient is not installed: nem looks for it on the PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), emacsclientTimeout)
	defer cancel()
	// Stdin, stdout and stderr are left unset - the null device - except that
	// stderr is captured, so emacsclient never takes over nem's terminal and
	// its complaint can be shown on the echo line.
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("emacsclient did not answer within %v", emacsclientTimeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i]
		}
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("emacsclient: %s", msg)
	}
	return nil
}

// openInEmacs sends path to the running Emacs, at line when line is positive.
// Lines are counted from one, as emacsclient's +LINE is.
func (e *Editor) openInEmacs(path string, line int) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	// -c opens a frame for the file if Emacs has none showing, so it appears
	// on screen rather than loading in the background. -n returns at once
	// rather than waiting for the buffer to be done with.
	args := []string{"-c", "-n"}
	if line > 0 {
		args = append(args, fmt.Sprintf("+%d", line))
	}
	args = append(args, abs)
	if err := emacsclientRun(args...); err != nil {
		return err
	}
	if line > 0 {
		e.Echo("Sent %s:%d to Emacs", filepath.Base(abs), line)
	} else {
		e.Echo("Sent %s to Emacs", filepath.Base(abs))
	}
	return nil
}

// registerEmacsCommands adds open-in-emacs.
func registerEmacsCommands(e *Editor, reg *command.Registry) error {
	c := command.Command{
		Name:        "open-in-emacs",
		Doc:         "Open this buffer's file in a running Emacs, at point's line. A listing opens its directory. Needs emacsclient and a server.",
		Interactive: true,
		Fn: func(command.Env) error {
			b := e.active.Buf
			if st := e.diredOf(b); st != nil {
				return e.openInEmacs(st.dir, 0)
			}
			if b.Path() == "" {
				e.Echo("This buffer has no file to open in Emacs; save it first (C-x C-w)")
				return nil
			}
			// Emacs reads the file from disk, so unsaved edits would not be
			// there. Say so rather than show Emacs an older version.
			if b.Modified() {
				e.Echo("This buffer has unsaved changes; save it first (C-x C-s)")
				return nil
			}
			return e.openInEmacs(b.Path(), e.active.Pt.Line+1)
		},
	}
	return reg.Register(c)
}

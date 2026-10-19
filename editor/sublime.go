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

// Shift-F5 hands the current buffer to Sublime Text through its subl command-
// line tool. subl opens the file in a running Sublime, or starts Sublime if it
// is not running. Nothing else is started here.

// sublimeTimeout is how long subl may take. It returns at once after handing
// over a file, unless it is asked to wait, which nem never does.
const sublimeTimeout = 5 * time.Second

// sublimeRun runs subl with args. It is a variable so the test binary can
// replace it, as emacsclientRun is replaced: a test run must never launch the
// developer's Sublime. See TestMain.
var sublimeRun = runSubl

// runSubl runs the subl on the PATH with args and reports how it failed, if it
// did.
func runSubl(args ...string) error {
	bin, err := exec.LookPath("subl")
	if err != nil {
		return errors.New("subl is not installed: nem looks for it on the PATH (in Sublime, install the 'subl' command)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), sublimeTimeout)
	defer cancel()
	// As with emacsclient, stdin and stdout are left as the null device and
	// stderr is captured, so subl never takes over nem's terminal.
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("subl did not answer within %v", sublimeTimeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i]
		}
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("subl: %s", msg)
	}
	return nil
}

// openInSublime sends path to Sublime Text, at line when line is positive.
// subl takes FILE:LINE, with the line counted from one.
func (e *Editor) openInSublime(path string, line int) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	arg := abs
	if line > 0 {
		arg = fmt.Sprintf("%s:%d", abs, line)
	}
	if err := sublimeRun(arg); err != nil {
		return err
	}
	if line > 0 {
		e.Echo("Sent %s:%d to Sublime Text", filepath.Base(abs), line)
	} else {
		e.Echo("Sent %s to Sublime Text", filepath.Base(abs))
	}
	return nil
}

// registerSublimeCommands adds open-in-sublime.
func registerSublimeCommands(e *Editor, reg *command.Registry) error {
	c := command.Command{
		Name:        "open-in-sublime",
		Doc:         "Open this buffer's file in Sublime Text, at point's line. A listing opens its directory. Needs the subl command.",
		Interactive: true,
		Fn: func(command.Env) error {
			b := e.active.Buf
			if st := e.diredOf(b); st != nil {
				return e.openInSublime(st.dir, 0)
			}
			if b.Path() == "" {
				e.Echo("This buffer has no file to open in Sublime Text; save it first (C-x C-w)")
				return nil
			}
			// Sublime reads the file from disk, so unsaved edits would not be
			// there. Say so rather than show an older version.
			if b.Modified() {
				e.Echo("This buffer has unsaved changes; save it first (C-x C-s)")
				return nil
			}
			return e.openInSublime(b.Path(), e.active.Pt.Line+1)
		},
	}
	return reg.Register(c)
}

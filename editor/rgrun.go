package editor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Borderliner/nem/command"
)

// rgRunTimeout is how long rg may run before it is stopped.
const rgRunTimeout = 30 * time.Second

// rgRunName is the results buffer's name. One is kept and reused, as fd's is.
const rgRunName = "*rg*"

// rgRun runs rg with args, its arguments as typed, in dir, and shows the
// matching lines in a read-only buffer, one match per line with its file and
// line number. Nothing found shows nothing, as fd's search does.
func (e *Editor) rgRun(dir, args string) error {
	bin, err := exec.LookPath("rg")
	if err != nil {
		return errors.New("rg is not installed: nem looks for rg on the PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), rgRunTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", bin+" --color never --line-number --with-filename "+args)
	cmd.Dir = dir
	out, err := cmd.Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		// rg exits 1 when nothing matched, which is not a failure.
		e.Echo("No matches for rg %s in %s", args, dir)
		return nil
	}
	if err != nil {
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return fmt.Errorf("rg: %s", vcFirstLine(string(ee.Stderr), nil))
		}
		return fmt.Errorf("rg: %w", err)
	}
	matches := strings.TrimRight(string(out), "\n")
	if matches == "" {
		e.Echo("No matches for rg %s in %s", args, dir)
		return nil
	}

	b, ok := e.byName[rgRunName]
	if !ok {
		b = e.NewBuffer(e.uniqueName(rgRunName))
	}
	body := fmt.Sprintf("rg %s in %s\n\n%s\n", args, dir, matches)
	if err := fillReadOnly(b, body); err != nil {
		return err
	}
	e.active.Visit(b)
	e.Echo("%d matching lines for rg %s", strings.Count(matches, "\n")+1, args)
	return nil
}

// registerRgRunCommands adds dired-rg, which runs rg from the dired buffer's
// directory with the arguments typed, as dired-fd runs fd.
func registerRgRunCommands(e *Editor, reg *command.Registry) error {
	cmd := command.Command{
		Name: "dired-rg",
		Doc:  "Run rg, with the arguments typed, in the directory of this dired buffer, and list the matching lines.",
		Fn: func(command.Env) error {
			st := e.diredOf(e.active.Buf)
			if st == nil {
				return errors.New("not a dired buffer")
			}
			args, err := e.ReadString(command.ReadOpts{Prompt: "rg: ", History: "rg"})
			if err != nil {
				return err
			}
			return e.rgRun(st.dir, args)
		},
	}
	cmd.Interactive = true
	return reg.Register(cmd)
}

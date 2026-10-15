package editor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/keymap"
	"github.com/Borderliner/nem/text"
)

// fd runs the fd file finder from a dired buffer's directory and lists what it
// finds in a read-only buffer, one path per line relative to that directory.
// RET or f visits the path at point, and a directory found is listed in dired.
// The listing is not a dired buffer: fd's results span subdirectories, and a
// dired buffer lists one directory.

// fdName is the results buffer's name. One is kept and reused, as the search
// results are.
const fdName = "*fd*"

// fdHeaderLines is how many lines come before the paths: the search's heading
// and a blank line.
const fdHeaderLines = 2

// fdTimeout is how long fd may run before it is stopped.
const fdTimeout = 30 * time.Second

// fdState is what makes a buffer fd's results.
type fdState struct {
	dir     string
	pattern string
}

// fdBindings is the results' keymap. The buffer is read-only, so printable
// keys are free to take.
var fdBindings = []struct{ Spec, Command string }{
	{"RET", "fd-find-file"},
	{"f", "fd-find-file"},
	{"n", "next-line"},
	{"p", "previous-line"},
	{"q", "quit-window"},
}

// newFdKeymap builds the fd results' keymap from fdBindings.
func newFdKeymap() (*keymap.Map, error) {
	m := keymap.New()
	for _, b := range fdBindings {
		if err := bindSpec(m, b.Spec, b.Command); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// fdOf returns b's fd results, or nil when b holds none.
func (e *Editor) fdOf(b *text.Buffer) *fdState {
	if b == nil {
		return nil
	}
	return e.fds[b]
}

// fdBinary is the fd command on this system: fd, or fdfind where the Debian
// package installs it under that name.
func fdBinary() (string, error) {
	for _, name := range []string{"fd", "fdfind"} {
		if _, err := exec.LookPath(name); err == nil {
			return name, nil
		}
	}
	return "", errors.New("fd is not installed: nem looks for fd and fdfind on the PATH")
}

// fdRun runs fd with pattern, its arguments as typed, in dir, and shows the
// paths it found. Nothing found shows nothing, as a search does.
func (e *Editor) fdRun(dir, pattern string) error {
	bin, err := fdBinary()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), fdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", bin+" --color never "+pattern)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return fmt.Errorf("fd: %s", vcFirstLine(string(ee.Stderr), nil))
		}
		return fmt.Errorf("fd: %w", err)
	}
	paths := strings.TrimRight(string(out), "\n")
	if paths == "" {
		e.Echo("No files found by fd %s in %s", pattern, dir)
		return nil
	}

	b, ok := e.byName[fdName]
	if !ok || e.fdOf(b) == nil {
		b = e.NewBuffer(e.uniqueName(fdName))
	}
	e.fds[b] = &fdState{dir: dir, pattern: pattern}
	if err := fillReadOnly(b, fmt.Sprintf("fd %s in %s\n\n%s\n", pattern, dir, paths)); err != nil {
		return err
	}

	// The results take this window in a tab of their own, not a split beside
	// the dired buffer they came from.
	e.active.Visit(b)
	for _, w := range e.tree.Windows() {
		if w.Buf == b {
			w.Pt = text.Pos{Line: fdHeaderLines}
			w.SetTop(0)
		}
	}
	e.Echo("%d found by fd %s", strings.Count(paths, "\n")+1, pattern)
	return nil
}

// fillReadOnly replaces b's text with s and leaves b read-only, unmodified:
// the text is a rendering, not an edit to keep.
func fillReadOnly(b *text.Buffer, s string) error {
	b.SetReadOnly(false)
	defer b.SetReadOnly(true)
	if b.End() != (text.Pos{}) {
		if err := b.Delete(text.Pos{}, b.End()); err != nil {
			return err
		}
	}
	if err := b.Insert(text.Pos{}, []rune(s)); err != nil {
		return err
	}
	b.SetModified(false)
	return nil
}

// fdFindFile visits the path on the line at point: a file in the buffer, or a
// directory in dired.
func (e *Editor) fdFindFile() error {
	b := e.active.Buf
	st := e.fdOf(b)
	if st == nil {
		return errors.New("not a fd results buffer")
	}
	line := e.active.Pt.Line
	if line < fdHeaderLines || line >= b.NumLines() {
		return nil
	}
	p := filepath.Join(st.dir, b.Line(line).String())
	info, err := os.Stat(p)
	if err != nil {
		return err
	}
	if info.IsDir() {
		db, err := e.Dired(p)
		if err != nil {
			return err
		}
		e.active.Visit(db)
		return nil
	}
	return e.visitFile(p)
}

// registerFdCommands adds dired-fd, which runs fd from the dired buffer's
// directory, and fd-find-file, which visits a path in the results.
func registerFdCommands(e *Editor, reg *command.Registry) error {
	cmds := []command.Command{
		{Name: "dired-fd", Doc: "Run fd, with the arguments typed, in the directory of this dired buffer, and list the files it finds.",
			Fn: func(command.Env) error {
				st := e.diredOf(e.active.Buf)
				if st == nil {
					return errors.New("not a dired buffer")
				}
				pattern, err := e.ReadString(command.ReadOpts{Prompt: "fd: ", History: "fd"})
				if err != nil {
					return err
				}
				return e.fdRun(st.dir, pattern)
			}},
		{Name: "dired-rg", Doc: "Search the files under this dired buffer's directory with rg, as the pattern is typed.",
			Fn: func(command.Env) error {
				st := e.diredOf(e.active.Buf)
				if st == nil {
					return errors.New("not a dired buffer")
				}
				return e.rgLive(st.dir)
			}},
		{Name: "fd-find-file", Doc: "Visit the file at point in the fd results, or list the directory there in dired.",
			Fn: func(command.Env) error { return e.fdFindFile() }},
	}
	for _, c := range cmds {
		c.Interactive = true
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

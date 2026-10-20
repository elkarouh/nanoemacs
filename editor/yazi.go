package editor

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Borderliner/nem/command"
)

// M-x yazi hands the terminal to the yazi file manager. nem is suspended while
// yazi runs, and what yazi's chooser picks - files or a directory - is opened
// when it quits: a file is visited, a directory is listed in dired. Quitting
// yazi without choosing anything changes nothing.

// yaziRun runs yazi on start with its chooser writing to chooser, on nem's
// terminal. It is a variable so the test binary can replace it: a test run
// must never start a real yazi. See TestMain.
var yaziRun = runYazi

func runYazi(start, chooser string) error {
	bin, err := exec.LookPath("yazi")
	if err != nil {
		return errors.New("yazi is not installed: nem looks for it on the PATH")
	}
	cmd := exec.Command(bin, "--chooser-file="+chooser, start)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yazi: %w", err)
	}
	return nil
}

// yaziStart is where yazi opens: the active buffer's file, so yazi reveals it,
// else its directory, else the working directory.
func (e *Editor) yaziStart() string {
	b := e.active.Buf
	if st := e.diredOf(b); st != nil {
		if en, ok := e.entryAtPoint(st); ok {
			return filepath.Join(st.dir, en.Name)
		}
		return st.dir
	}
	if p := b.Path(); p != "" {
		return p
	}
	if d := e.bufferDir(b); d != "" {
		return d
	}
	wd, _ := os.Getwd()
	return wd
}

// yazi runs yazi on the active buffer and opens what it chose.
func (e *Editor) yazi() error { return e.yaziAt(e.yaziStart()) }

// StartYazi is nem with no file named: yazi on dir, before the first frame. A
// failure, or yazi quitting with nothing chosen, leaves the buffer shown as it
// is, with the failure on the echo line.
func (e *Editor) StartYazi(dir string) {
	if err := e.yaziAt(dir); err != nil {
		e.Echo("%v", err)
	}
}

// yaziAt runs yazi on start and opens what it chose.
func (e *Editor) yaziAt(start string) error {
	f, err := os.CreateTemp("", "nem-yazi-*")
	if err != nil {
		return err
	}
	chooser := f.Name()
	f.Close()
	defer os.Remove(chooser)

	if e.scr != nil {
		if err := e.scr.Suspend(); err != nil {
			return err
		}
	}
	runErr := yaziRun(start, chooser)
	if e.scr != nil {
		if err := e.scr.Resume(); err != nil {
			return err
		}
		e.scr.Sync()
	}
	if runErr != nil {
		return runErr
	}

	f, err = os.Open(chooser)
	if err != nil {
		return err
	}
	defer f.Close()
	var paths []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if p := strings.TrimSpace(sc.Text()); p != "" {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	for i, p := range paths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			b, err := e.Dired(p)
			if err != nil {
				return err
			}
			if i == 0 {
				e.active.Visit(b)
			}
			continue
		}
		b, err := e.OpenFile(p)
		if err != nil {
			return err
		}
		if i == 0 {
			e.VisitOpened(b)
		}
	}
	return nil
}

// registerYaziCommands adds yazi.
func registerYaziCommands(e *Editor, reg *command.Registry) error {
	return reg.Register(command.Command{
		Name:        "yazi",
		Doc:         "Run the yazi file manager on this buffer's file or directory, then open what it chose. Needs yazi on the PATH.",
		Interactive: true,
		Fn:          func(command.Env) error { return e.yazi() },
	})
}

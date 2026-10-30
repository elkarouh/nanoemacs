package editor

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/sysopen"
	"github.com/Borderliner/nem/text"
)

// M-x yazi hands the terminal to the yazi file manager. nem is suspended while
// yazi runs, and what yazi's chooser picks - files or a directory - is opened
// when it quits: a file is visited, a directory is listed in dired. Quitting
// yazi without choosing anything changes nothing.

// yaziCwdFile is where yazi writes the directory it ended in, beside chooser.
func yaziCwdFile(chooser string) string { return chooser + ".cwd" }

// yaziRun runs yazi on start with its chooser writing to chooser, and the
// directory it ended in to yaziCwdFile(chooser), on nem's terminal. It is a variable so the test binary can replace it: a test run
// must never start a real yazi. See TestMain.
var yaziRun = runYazi

func runYazi(start, chooser string) error {
	bin, err := exec.LookPath("yazi")
	if err != nil {
		return errors.New("yazi is not installed: nem looks for it on the PATH")
	}
	cmd := exec.Command(bin, "--chooser-file="+chooser, "--cwd-file="+yaziCwdFile(chooser), start)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yazi: %w", err)
	}
	return nil
}

// yaziStart is the directory yazi opens in: the active buffer's file's, or the
// one a dired buffer lists, else the working directory.
func (e *Editor) yaziStart() string {
	b := e.active.Buf
	if d := e.bufferDir(b); d != "" {
		return d
	}
	wd, _ := os.Getwd()
	return wd
}

// ExitDir is the directory a shell should be left in when nem exits: the one
// the active buffer lists or its file is in, else the working directory.
//
// Where yazi last ended counts as the latest directory visited, until another
// buffer is shown.
func (e *Editor) ExitDir() string {
	if e.yaziEnd != "" && e.active.Buf == e.yaziEndBuf {
		return e.yaziEnd
	}
	return e.yaziStart()
}

// yazi runs yazi on the active buffer and opens what it chose.
func (e *Editor) yazi() error { return e.yaziAt(e.yaziStart()) }

// selectOtherWindow selects another window, splitting the frame first when
// there is only one.
func (e *Editor) selectOtherWindow() error {
	if len(e.tree.Windows()) < 2 {
		return e.SplitWindow(true)
	}
	e.OtherWindow(1)
	return nil
}

// yaziOtherWindow runs yazi on the active buffer and opens what it chose in
// the other window, which stays where it is if nothing is chosen.
func (e *Editor) yaziOtherWindow() error {
	e.yaziOther = true
	defer func() { e.yaziOther = false }()
	return e.yaziAt(e.yaziStart())
}

// StartYazi is nem with no file named: yazi on dir, before the first frame. It
// reports whether nem should quit instead of starting: yazi was quit without
// choosing anything, as with q. When yazi is missing or fails, dir is listed in
// dired instead, with the failure on the echo line.
func (e *Editor) StartYazi(dir string) (quit bool) {
	before := e.active.Buf
	err := e.yaziAt(dir)
	if err == nil && e.active.Buf == before {
		return true
	}
	if err != nil {
		defer e.Echo("%v", err)
	}
	if e.active.Buf != before {
		return false
	}
	if e.yaziEnd != "" {
		dir = e.yaziEnd // where yazi was left
	}
	if b, derr := e.Dired(dir); derr == nil {
		e.active.Visit(b)
	}
	return false
}

// yaziPause is how long the refusal of a binary file stays on the terminal
// before yazi comes back over it. A variable so tests need not wait.
var yaziPause = 1500 * time.Millisecond

// yaziChosen reads the paths yazi's chooser file holds.
func yaziChosen(chooser string) []string {
	f, err := os.Open(chooser)
	if err != nil {
		return nil
	}
	defer f.Close()
	var paths []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if p := strings.TrimSpace(sc.Text()); p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

// nonTextNames lists the base names of the files among paths that are not text.
func nonTextNames(paths []string) []string {
	var out []string
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			continue
		}
		if isText, err := sysopen.IsText(p); err == nil && !isText && !isArchive(p) {
			out = append(out, filepath.Base(p))
		}
	}
	return out
}

// yaziAt runs yazi on start and opens what it chose.
func (e *Editor) yaziAt(start string) error {
	e.yaziChosen = false
	f, err := os.CreateTemp("", "nem-yazi-*")
	if err != nil {
		return err
	}
	chooser := f.Name()
	f.Close()
	defer os.Remove(chooser)
	defer os.Remove(yaziCwdFile(chooser))

	if e.scr != nil {
		if err := e.scr.Suspend(); err != nil {
			return err
		}
	}
	// yazi is run again, where it was left, while all it chose is not text:
	// there is nothing to open, and the way out is another choice.
	var paths []string
	var runErr error
	for {
		runErr = yaziRun(start, chooser)
		if runErr != nil {
			break
		}
		paths = yaziChosen(chooser)
		// Books go to the system's app, or else a terminal reader, if there is
		// one, and yazi comes back after it, as after a picture.
		graphical, bin := e.ext.available() == nil, epubReader()
		if onlyEPUBs(paths) && (graphical || bin != "") {
			for _, p := range paths {
				var err error
				if graphical {
					err = e.launch(p)
				} else {
					err = epubRun(bin, p)
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nnem: %v\n", err)
					time.Sleep(yaziPause)
				}
			}
			if d, err := os.ReadFile(yaziCwdFile(chooser)); err == nil {
				if dir := strings.TrimSpace(string(d)); dir != "" {
					start = dir
				}
			}
			os.Truncate(chooser, 0)
			paths = nil
			continue
		}
		// Pictures and PDFs are shown, in terminals that can, and yazi comes back
		// over them, as it does after a refused binary file.
		if len(paths) > 0 && e.canShowImages() && pictureOrPDF(paths) && e.scr != nil {
			if err := e.scr.Resume(); err != nil {
				return err
			}
			for _, p := range paths {
				if err := e.showPicture(p); err != nil {
					e.Echo("%v", err)
				}
			}
			if err := e.scr.Suspend(); err != nil {
				return err
			}
			if d, err := os.ReadFile(yaziCwdFile(chooser)); err == nil {
				if dir := strings.TrimSpace(string(d)); dir != "" {
					start = dir
				}
			}
			os.Truncate(chooser, 0)
			paths = nil
			continue
		}
		binaries := nonTextNames(paths)
		if len(paths) == 0 || len(binaries) != len(paths) {
			break
		}
		fmt.Fprintf(os.Stderr, "\nnem: not a text file: %s\n", strings.Join(binaries, ", "))
		time.Sleep(yaziPause)
		if d, err := os.ReadFile(yaziCwdFile(chooser)); err == nil {
			if dir := strings.TrimSpace(string(d)); dir != "" {
				start = dir
			}
		}
		os.Truncate(chooser, 0)
		paths = nil
	}
	if e.scr != nil {
		if err := e.scr.Resume(); err != nil {
			return err
		}
		e.scr.Sync()
	}
	if runErr != nil {
		return runErr
	}
	// Where yazi was left, for ExitDir, unless what it chose is opened below.
	if d, err := os.ReadFile(yaziCwdFile(chooser)); err == nil {
		before := e.active.Buf
		defer func() {
			if dir := strings.TrimSpace(string(d)); dir != "" && e.active.Buf == before {
				e.yaziEnd, e.yaziEndBuf = dir, before
			}
		}()
	}

	e.yaziChosen = len(paths) > 0
	if len(paths) == 0 {
		return nil
	}
	// What yazi chose is opened, the first of it shown. A file that is not
	// text is refused, whatever the setting for such files: choosing one here
	// is a mistake, not a request for the system's app. An archive is listed
	// instead, as its table of contents.
	shown := false
	var refused []string
	for _, p := range paths {
		var b *text.Buffer
		var err error
		if fi, serr := os.Stat(p); serr == nil && fi.IsDir() {
			b, err = e.Dired(p)
		} else if isArchive(p) {
			b, err = e.archiveList(p)
		} else if isText, terr := sysopen.IsText(p); terr == nil && !isText {
			refused = append(refused, filepath.Base(p))
			continue
		} else {
			b, err = e.OpenFile(p)
		}
		if err != nil {
			return err
		}
		if !shown {
			if e.yaziOther {
				if err := e.selectOtherWindow(); err != nil {
					return err
				}
			}
			e.VisitOpened(b)
			shown = true
		}
	}
	if len(refused) > 0 {
		return fmt.Errorf("not a text file: %s", strings.Join(refused, ", "))
	}
	return nil
}

// quitIfNothingChosen ends the session when yazi was quit without choosing
// anything, as q does: yazi was the way around nem, and leaving it is leaving
// both. Modified buffers are offered for saving first, as C-x C-c does.
func (e *Editor) quitIfNothingChosen(env command.Env) error {
	if e.yaziChosen {
		return nil
	}
	return env.Run("save-buffers-kill-terminal")
}

// yaziKillBuffer closes the active buffer, asking first if it is modified, then
// runs yazi in the directory it was in. It is for leaving a file for another
// one without keeping the first open. When it is the only buffer there is
// nothing to close it into, so it stays.
func (e *Editor) yaziKillBuffer() error {
	e.yaziChosen = true // unless yazi runs and is quit with nothing chosen
	b := e.active.Buf
	start := e.yaziStart()
	if b.Modified() && b.Path() != "" {
		c, err := e.ReadChar(fmt.Sprintf("Buffer %s modified; kill anyway? (y/n) ", e.BufferName(b)), []rune{'y', 'n'})
		if err != nil {
			return err
		}
		if c != 'y' {
			e.Echo("Buffer not killed")
			return nil
		}
	}
	if len(e.buffers) > 1 {
		if err := e.KillBuffer(b); err != nil {
			return err
		}
	}
	return e.yaziAt(start)
}

// registerYaziCommands adds yazi and yazi-kill-buffer.
func registerYaziCommands(e *Editor, reg *command.Registry) error {
	for _, c := range []command.Command{
		{Name: "yazi", Doc: "Run the yazi file manager in this buffer's directory, then open what it chose; quitting yazi with nothing chosen quits nem. Needs yazi on the PATH.",
			Interactive: true, Fn: func(env command.Env) error {
				if err := e.yazi(); err != nil {
					return err
				}
				return e.quitIfNothingChosen(env)
			}},
		{Name: "yazi-other-window", Doc: "Run the yazi file manager, then open what it chose in the other window, splitting the frame if there is only one. Needs yazi on the PATH.",
			Interactive: true, Fn: func(command.Env) error { return e.yaziOtherWindow() }},
		{Name: "yazi-kill-buffer", Doc: "Kill this buffer, confirming if it is modified, then run yazi in its directory and open what it chose. Quitting yazi with nothing chosen quits nem.",
			Interactive: true, Fn: func(env command.Env) error {
				if err := e.yaziKillBuffer(); err != nil {
					return err
				}
				return e.quitIfNothingChosen(env)
			}},
	} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

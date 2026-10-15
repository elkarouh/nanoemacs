package editor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/keymap"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/view"
)

// The shell buffer: M-t goes to *shell*, an interactive shell in an ordinary
// buffer, and M-t again comes back to the buffer it was entered from. It is
// emacs's M-x shell: type a command after the prompt, RET sends it, and what
// the shell prints arrives in the buffer below.
//
// The shell is the user's own ($SHELL, or the shell setting) run with -i, on
// pipes rather than a terminal. That is why there is no screen to emulate -
// the buffer is the screen - and why programs that want a terminal (vim,
// htop, less, a password prompt) do not work in it.
//
// The buffer is read-only up to shellState.mark, where the last output ended;
// everything after it is the input being typed. Output arriving while input
// is half typed is placed before it.

const shellBufferName = "*shell*"

var errNotShell = errors.New("not a shell buffer")

// errShellPrompt refuses an edit that reaches back into what the shell
// printed.
var errShellPrompt = errors.New("text is read-only before the prompt")

// shellState is what makes a buffer a shell.
type shellState struct {
	proc    *process
	stdin   io.WriteCloser
	running bool
	dir     string

	// mark is where the input being typed starts: just past the last output.
	mark text.Pos

	// hist is the lines sent, oldest first. histAt is the one M-p and M-n
	// are showing, len(hist) when none is; draft is what was being typed
	// when they began.
	hist   []string
	histAt int
	draft  string
}

// guard is the buffer's edit guard: the shell's output is not editable.
func (st *shellState) guard(from, _ text.Pos, _ []rune) error {
	if posBefore(from, st.mark) {
		return errShellPrompt
	}
	return nil
}

func posBefore(a, b text.Pos) bool {
	return a.Line < b.Line || (a.Line == b.Line && a.Col < b.Col)
}

// shellBindings is the shell buffer's keymap, over the global one.
var shellBindings = []struct{ Spec, Command string }{
	{"RET", "shell-send-input"},
	{"C-c C-c", "shell-interrupt"},
	{"C-d", "shell-delete-or-eof"},
	{"C-a", "shell-beginning-of-line"},
	{"M-p", "shell-previous-input"},
	{"M-n", "shell-next-input"},
}

// newShellKeymap builds the shell buffer's keymap from shellBindings.
func newShellKeymap() (*keymap.Map, error) {
	m := keymap.New()
	for _, b := range shellBindings {
		if err := bindSpec(m, b.Spec, b.Command); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// shellOf returns b's shell state, or nil when b is not a shell.
func (e *Editor) shellOf(b *text.Buffer) *shellState {
	if b == nil {
		return nil
	}
	return e.shells[b]
}

// registerShellBufferCommands adds M-t and the shell buffer's commands.
func registerShellBufferCommands(e *Editor, reg *command.Registry) error {
	inShell := func(fn func(b *text.Buffer, st *shellState) error) func(command.Env) error {
		return func(command.Env) error {
			b := e.active.Buf
			st := e.shellOf(b)
			if st == nil {
				return errNotShell
			}
			return fn(b, st)
		}
	}
	// orElse runs the ordinary command name where point is not in the
	// input, or where the shell command has nothing to say.
	orElse := func(env command.Env, name string) error {
		c, ok := e.reg.Lookup(name)
		if !ok {
			return nil
		}
		return c.Fn(env)
	}
	for _, c := range []command.Command{
		{Name: "shell-toggle", Doc: "Go to the shell buffer, or back to the buffer it was entered from.",
			Fn: func(command.Env) error { return e.shellToggle() }},
		{Name: "shell-send-input", Doc: "Send the line being typed to the shell.",
			Fn: inShell(func(b *text.Buffer, st *shellState) error { return e.shellSend(b, st) })},
		{Name: "shell-interrupt", Doc: "Interrupt what the shell is running, as C-c does in a terminal.",
			Fn: inShell(func(b *text.Buffer, st *shellState) error {
				if !st.running {
					e.Echo("The shell is not running")
					return nil
				}
				sigintGroup(st.proc.cmd)
				return nil
			})},
		{Name: "shell-delete-or-eof", Doc: "Delete the character after point, or at an empty prompt end the shell's input.",
			Fn: func(env command.Env) error {
				b := e.active.Buf
				st := e.shellOf(b)
				if st == nil {
					return errNotShell
				}
				if st.running && b.End() == b.ClampPos(st.mark) && e.active.Pt == b.End() {
					return st.stdin.Close()
				}
				return orElse(env, "delete-char")
			}},
		{Name: "shell-beginning-of-line", Doc: "Move to the start of the input on the prompt's line, or of the line elsewhere.",
			Fn: func(env command.Env) error {
				b := e.active.Buf
				st := e.shellOf(b)
				if st == nil {
					return errNotShell
				}
				if mark := b.ClampPos(st.mark); e.active.Pt.Line == mark.Line {
					e.active.Pt = mark
					e.active.GoalCol = view.GoalColUnset
					return nil
				}
				return orElse(env, "move-beginning-of-line")
			}},
		{Name: "shell-previous-input", Doc: "Replace the input with the one sent before it.",
			Fn: inShell(func(b *text.Buffer, st *shellState) error { return e.shellHistory(b, st, -1) })},
		{Name: "shell-next-input", Doc: "Replace the input with the one sent after it.",
			Fn: inShell(func(b *text.Buffer, st *shellState) error { return e.shellHistory(b, st, 1) })},
	} {
		c.Interactive = true
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// shellToggle is M-t. From anywhere else it goes to the shell, starting one
// if there is none; from the shell it goes back.
func (e *Editor) shellToggle() error {
	w := e.active
	if e.shellOf(w.Buf) != nil {
		return e.shellLeave()
	}
	dir := e.shellDir()
	b, ok := e.byName[shellBufferName]
	if !ok || e.shellOf(b) == nil {
		b = e.NewBuffer(e.uniqueName(shellBufferName))
		st := &shellState{dir: dir}
		b.SetEditGuard(st.guard)
		e.shells[b] = st
	}
	st := e.shells[b]
	e.shellPrev = w.Buf
	w.Visit(b)
	if !st.running {
		if err := e.shellStart(b, st, dir); err != nil {
			return err
		}
	}
	w.Pt = b.End()
	w.GoalCol = view.GoalColUnset
	return nil
}

// shellLeave goes back from the shell: to the buffer M-t came from, or, when
// that is gone, the one visited most recently before the shell.
func (e *Editor) shellLeave() error {
	w := e.active
	prev := e.shellPrev
	if prev == nil || prev == w.Buf || !slices.Contains(e.buffers, prev) {
		prev = nil
		for _, c := range e.buffers {
			if c != w.Buf && e.shellOf(c) == nil {
				prev = c
				break
			}
		}
	}
	if prev == nil {
		e.Echo("No other buffer to go back to")
		return nil
	}
	w.Visit(prev)
	return nil
}

// shellStart runs the user's shell in dir, for b to talk to.
func (e *Editor) shellStart(b *text.Buffer, st *shellState, dir string) error {
	prog := e.shell
	if prog == "" {
		prog = os.Getenv("SHELL")
	}
	if prog == "" {
		prog = "/bin/sh"
	}
	p, in, err := e.startInteractive(prog, dir)
	if err != nil {
		return err
	}
	st.proc, st.stdin, st.running, st.dir = p, in, true, dir
	st.mark = b.End()
	return nil
}

// startInteractive runs prog as an interactive shell in dir, giving back the
// way to write to it. Its output and errors come to the loop as a command's
// do.
func (e *Editor) startInteractive(prog, dir string) (*process, io.WriteCloser, error) {
	cmd := exec.Command(prog)
	if runtime.GOOS != "windows" {
		// bash's line editor echoes what it reads even from a pipe, which
		// would show every line twice: the buffer already has what was typed.
		if filepath.Base(prog) == "bash" {
			cmd.Args = append(cmd.Args, "--noediting")
		}
		cmd.Args = append(cmd.Args, "-i")
	}
	cmd.Dir = dir
	// No terminal: a pager would wait for keys nobody can send, and colour
	// meant for one is noise in a buffer.
	cmd.Env = append(os.Environ(), "TERM=dumb", "PAGER=cat", "GIT_PAGER=cat")
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	r, w, err := os.Pipe()
	if err != nil {
		in.Close()
		return nil, nil, err
	}
	cmd.Stdout, cmd.Stderr = w, w
	ownSession(cmd)
	if err := cmd.Start(); err != nil {
		r.Close()
		w.Close()
		in.Close()
		return nil, nil, err
	}
	w.Close() // the child has its own copy; ours would keep the pipe open
	return e.pump(cmd, r), in, nil
}

// shellWoke takes in what the shell has for the loop.
func (e *Editor) shellWoke(b *text.Buffer, st *shellState) {
	out, done := st.proc.take()
	if len(out) > 0 {
		e.shellFeed(b, st, shellClean(out))
	}
	if done && st.running {
		st.running = false
		_ = st.stdin.Close()
		code, _ := st.proc.result()
		e.shellFeed(b, st, fmt.Sprintf("\n[Shell exited with code %d. RET starts a new one.]\n", code))
	}
}

// shellClean turns what the shell printed into plain text, and drops the two
// complaints an interactive shell makes at finding no terminal, which are
// true and of no use.
func shellClean(out []byte) string {
	lines := strings.Split(cleanOutput(out), "\n")
	keep := lines[:0]
	for _, l := range lines {
		if strings.HasPrefix(l, "bash: cannot set terminal process group") ||
			strings.HasPrefix(l, "bash: no job control in this shell") {
			continue
		}
		keep = append(keep, l)
	}
	return strings.Join(keep, "\n")
}

// shellFeed appends output to b, ahead of any input being typed, and moves
// the mark past it. Windows with point in the input keep it there.
func (e *Editor) shellFeed(b *text.Buffer, st *shellState, out string) {
	if out == "" {
		return
	}
	oldMark := b.ClampPos(st.mark)
	lineStart := text.Pos{Line: oldMark.Line}
	// The line the mark is on is rewritten along with what follows it, so
	// what is before the mark on it is carried over.
	tail := slices.Clone(b.Text(lineStart, oldMark))
	pending := b.Text(oldMark, b.End())
	tail = append(tail, []rune(out)...)
	newMark := advancePos(lineStart, tail)

	ws := e.tree.Windows()
	offs := map[int]int{}
	for i, w := range ws {
		if w.Buf == b && !posBefore(w.Pt, oldMark) {
			offs[i] = len(b.Text(oldMark, b.ClampPos(w.Pt)))
		}
	}

	b.RegenerateTail(oldMark.Line, append(tail, pending...))
	// The history was of text that has just been rewritten.
	b.ResetUndo()
	b.SetModified(false)
	st.mark = newMark
	for i, off := range offs {
		ws[i].Pt = advancePos(newMark, pending[:min(off, len(pending))])
	}
}

// shellSend is RET: the input goes to the shell, and the line it was on ends.
func (e *Editor) shellSend(b *text.Buffer, st *shellState) error {
	if !st.running {
		if err := e.shellStart(b, st, st.dir); err != nil {
			return err
		}
	}
	input := string(b.Text(b.ClampPos(st.mark), b.End()))
	if err := b.Insert(b.End(), []rune{'\n'}); err != nil {
		return err
	}
	st.mark = b.End()
	b.ResetUndo()
	b.SetModified(false)
	if strings.TrimSpace(input) != "" && (len(st.hist) == 0 || st.hist[len(st.hist)-1] != input) {
		st.hist = append(st.hist, input)
	}
	st.histAt = len(st.hist)
	for _, w := range e.tree.Windows() {
		if w.Buf == b {
			w.Pt = b.End()
			w.GoalCol = view.GoalColUnset
		}
	}
	if _, err := io.WriteString(st.stdin, input+"\n"); err != nil {
		e.Echo("Shell: %v", err)
	}
	return nil
}

// shellHistory is M-p (dir -1) and M-n (dir 1): the input is replaced by an
// earlier or later one sent, and by what was being typed after the last.
func (e *Editor) shellHistory(b *text.Buffer, st *shellState, dir int) error {
	if len(st.hist) == 0 {
		e.Echo("No input sent yet")
		return nil
	}
	mark := b.ClampPos(st.mark)
	if st.histAt >= len(st.hist) {
		st.histAt = len(st.hist)
		st.draft = string(b.Text(mark, b.End()))
	}
	at := st.histAt + dir
	if at < 0 {
		e.Echo("Beginning of input history")
		return nil
	}
	if at > len(st.hist) {
		e.Echo("End of input history")
		return nil
	}
	st.histAt = at
	line := st.draft
	if at < len(st.hist) {
		line = st.hist[at]
	}
	if end := b.End(); mark != end {
		if err := b.Delete(mark, end); err != nil {
			return err
		}
	}
	if err := b.Insert(mark, []rune(line)); err != nil {
		return err
	}
	e.active.Pt = b.End()
	e.active.GoalCol = view.GoalColUnset
	return nil
}

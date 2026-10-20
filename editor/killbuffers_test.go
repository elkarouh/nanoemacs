package editor

import (
	"github.com/gdamore/tcell/v2"
	"os"
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T, e *Editor, names ...string) {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte(n+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := e.OpenFile(p); err != nil {
			t.Fatal(err)
		}
	}
}

// runCommand runs name through M-x, answering its prompts with the characters
// in answers.
func runCommand(t *testing.T, e *Editor, scr tcell.SimulationScreen, name, answers string) {
	t.Helper()
	feed(t, scr, txt(name), key(t, "RET"), txt(answers))
	press(t, e, "M-x")
}

func TestKillAllBuffersLeavesScratch(t *testing.T) {
	e, scr := newTestEditor(t)
	openTemp(t, e, "a.txt", "b.txt", "c.txt")
	runCommand(t, e, scr, "kill-all-buffers", "y")
	if len(e.buffers) != 1 || e.byName["*scratch*"] == nil {
		t.Errorf("buffers left = %d, want just *scratch*", len(e.buffers))
	}
}

func TestKillAllBuffersAnswerNoKeepsThem(t *testing.T) {
	e, scr := newTestEditor(t)
	openTemp(t, e, "a.txt", "b.txt")
	n := len(e.buffers)
	runCommand(t, e, scr, "kill-all-buffers", "n")
	if len(e.buffers) != n {
		t.Errorf("buffers = %d, want %d", len(e.buffers), n)
	}
}

func TestKillSomeBuffersAsksForEach(t *testing.T) {
	e, scr := newTestEditor(t)
	openTemp(t, e, "a.txt", "b.txt", "c.txt")
	// Answers follow the buffer order: kill the first, keep the second, stop.
	runCommand(t, e, scr, "kill-some-buffers", "ynq")
	left := 0
	for _, b := range e.buffers {
		if filepath.Ext(b.Path()) == ".txt" {
			left++
		}
	}
	if left != 2 {
		t.Errorf("txt buffers left = %d, want 2", left)
	}
}

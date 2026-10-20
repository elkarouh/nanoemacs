package editor

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeYazi makes yaziRun choose paths, as yazi's chooser file would hold them.
func fakeYazi(t *testing.T, paths ...string) {
	t.Helper()
	old := yaziRun
	t.Cleanup(func() { yaziRun = old })
	yaziRun = func(start, chooser string) error {
		body := ""
		for _, p := range paths {
			body += p + "\n"
		}
		return os.WriteFile(chooser, []byte(body), 0o644)
	}
}

func TestYaziOpensTheChosenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeYazi(t, path)
	e, _ := newTestEditor(t)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if e.active.Buf.Path() != path {
		t.Errorf("active buffer is %q, want %q", e.active.Buf.Path(), path)
	}
}

func TestYaziOpensAChosenDirectoryInDired(t *testing.T) {
	dir := t.TempDir()
	fakeYazi(t, dir)
	e, _ := newTestEditor(t)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if st := e.diredOf(e.active.Buf); st == nil || st.dir != dir {
		t.Errorf("active buffer is not dired on %s", dir)
	}
}

func TestYaziWithNothingChosenChangesNothing(t *testing.T) {
	fakeYazi(t)
	e, _ := newTestEditor(t)
	before := e.active.Buf
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if e.active.Buf != before {
		t.Error("the active buffer changed")
	}
}

func TestYaziOpensInTheDirectoryOfTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got string
	old := yaziRun
	t.Cleanup(func() { yaziRun = old })
	yaziRun = func(start, chooser string) error { got = start; return nil }
	e, _ := newTestEditor(t)
	b, err := e.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(b)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("yazi started on %q, want %q", got, dir)
	}
}

func TestExitDirIsWhereYaziWasLeft(t *testing.T) {
	dir, other := t.TempDir(), t.TempDir()
	old := yaziRun
	t.Cleanup(func() { yaziRun = old })
	yaziRun = func(start, chooser string) error {
		return os.WriteFile(yaziCwdFile(chooser), []byte(other+"\n"), 0o644)
	}
	e, _ := newTestEditor(t)
	b, err := e.Dired(dir)
	if err != nil {
		t.Fatal(err)
	}
	e.active.Visit(b)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if got := e.ExitDir(); got != other {
		t.Errorf("ExitDir = %q, want %q", got, other)
	}
	// Showing another buffer takes over.
	third := t.TempDir()
	b2, _ := e.Dired(third)
	e.active.Visit(b2)
	if got := e.ExitDir(); got != third {
		t.Errorf("after switching buffers ExitDir = %q, want %q", got, third)
	}
}

func TestYaziKillBufferClosesTheBufferThenOpensWhatWasChosen(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fakeYazi(t, b)
	e, _ := newTestEditor(t)
	ba, err := e.OpenFile(a)
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(ba)
	if err := e.yaziKillBuffer(); err != nil {
		t.Fatal(err)
	}
	if e.active.Buf.Path() != b {
		t.Errorf("active buffer is %q, want %q", e.active.Buf.Path(), b)
	}
	for _, buf := range e.buffers {
		if buf.Path() == a {
			t.Error("a.txt is still open")
		}
	}
}

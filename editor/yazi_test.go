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

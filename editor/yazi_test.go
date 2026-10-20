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

func TestYaziComesBackAfterABinaryFile(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "blob.bin")
	if err := os.WriteFile(bin, []byte{0, 1, 2, 0, 255, 254, 0, 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	oldPause, oldRun := yaziPause, yaziRun
	t.Cleanup(func() { yaziPause, yaziRun = oldPause, oldRun })
	yaziPause = 0
	var starts []string
	yaziRun = func(start, chooser string) error {
		starts = append(starts, start)
		if len(starts) == 1 {
			// The first run chooses the binary file, from another directory.
			if err := os.WriteFile(yaziCwdFile(chooser), []byte(dir+"\n"), 0o644); err != nil {
				return err
			}
			return os.WriteFile(chooser, []byte(bin+"\n"), 0o644)
		}
		return nil // quit without choosing
	}
	e, _ := newTestEditor(t)
	before := e.active.Buf
	n := len(e.buffers)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if len(starts) != 2 || starts[1] != dir {
		t.Errorf("yazi ran %v, want a second run in %s", starts, dir)
	}
	if e.active.Buf != before || len(e.buffers) != n {
		t.Error("the binary file was opened")
	}
}

func TestYaziRefusesABinaryChosenAmongText(t *testing.T) {
	dir := t.TempDir()
	bin, txt := filepath.Join(dir, "blob.bin"), filepath.Join(dir, "a.txt")
	if err := os.WriteFile(bin, []byte{0, 1, 2, 0, 255, 254, 0, 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(txt, []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeYazi(t, bin, txt)
	e, _ := newTestEditor(t)
	if err := e.yazi(); err == nil {
		t.Error("the binary file was accepted without a word")
	}
	if e.active.Buf.Path() != txt {
		t.Errorf("active buffer is %q, want the text file", e.active.Buf.Path())
	}
}

func TestStartYaziQuitsWhenYaziIsQuitWithNothingChosen(t *testing.T) {
	fakeYazi(t)
	e, _ := newTestEditor(t)
	before := e.active.Buf
	if !e.StartYazi(t.TempDir()) {
		t.Error("nem would start after yazi was quit with q")
	}
	if e.active.Buf != before {
		t.Error("a buffer was opened")
	}
}

func TestStartYaziOpensWhatWasChosenAndStarts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeYazi(t, path)
	e, _ := newTestEditor(t)
	if e.StartYazi(dir) {
		t.Error("nem quit after a file was chosen")
	}
	if e.active.Buf.Path() != path {
		t.Errorf("active buffer is %q, want %q", e.active.Buf.Path(), path)
	}
}

func TestStartYaziFallsBackToDiredWhenYaziFails(t *testing.T) {
	dir := t.TempDir()
	old := yaziRun
	t.Cleanup(func() { yaziRun = old })
	yaziRun = func(string, string) error { return os.ErrNotExist }
	e, _ := newTestEditor(t)
	if e.StartYazi(dir) {
		t.Fatal("nem quit although yazi failed")
	}
	if st := e.diredOf(e.active.Buf); st == nil || st.dir != dir {
		t.Error("the directory was not listed in dired")
	}
}

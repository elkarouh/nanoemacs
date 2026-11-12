package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestYaziCommandQuitsNemWhenYaziIsQuitWithNothingChosen(t *testing.T) {
	fakeYazi(t)
	e, scr := newTestEditor(t)
	feed(t, scr, txt("yazi"), key(t, "RET"))
	press(t, e, "M-x")
	if !e.Quitting() {
		t.Error("nem is still running after yazi was quit with nothing chosen")
	}
}

func TestYaziCommandStaysWhenAFileIsChosen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeYazi(t, path)
	e, scr := newTestEditor(t)
	feed(t, scr, txt("yazi"), key(t, "RET"))
	press(t, e, "M-x")
	if e.Quitting() {
		t.Error("nem quit although a file was chosen")
	}
}

func TestYaziListsAnArchiveInsteadOfRefusingIt(t *testing.T) {
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar is not installed")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "inner.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	arc := filepath.Join(dir, "a.tar.gz")
	if out, err := exec.Command("tar", "-czf", arc, "-C", dir, "inner.txt").CombinedOutput(); err != nil {
		t.Fatalf("tar: %v: %s", err, out)
	}
	fakeYazi(t, arc)
	e, _ := newTestEditor(t)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	b := e.active.Buf
	if !b.ReadOnly() || !strings.Contains(b.String(), "inner.txt") {
		t.Errorf("archive buffer = %q (read-only %v), want a listing with inner.txt", b.String(), b.ReadOnly())
	}
}

func TestYaziStartsInTheDirectoryOfAnArchiveListing(t *testing.T) {
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar is not installed")
	}
	dir := t.TempDir()
	arc := filepath.Join(dir, "a.tar")
	if err := os.WriteFile(filepath.Join(dir, "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("tar", "-cf", arc, "-C", dir, "x").CombinedOutput(); err != nil {
		t.Fatalf("tar: %v: %s", err, out)
	}
	e, _ := newTestEditor(t)
	b, err := e.archiveList(arc)
	if err != nil {
		t.Fatal(err)
	}
	e.active.Visit(b)
	if got := e.yaziStart(); got != dir {
		t.Errorf("yazi starts in %q, want %q", got, dir)
	}
}

// yazi-other-window opens what yazi chose in the other window, splitting the
// frame when there is only one, and leaves the first window alone.
func TestYaziOtherWindowOpensInTheOtherWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeYazi(t, path)
	e, _ := newTestEditor(t, "scratch")
	first, orig := e.active, e.active.Buf

	if err := e.yaziOtherWindow(); err != nil {
		t.Fatal(err)
	}
	if len(e.tree.Windows()) != 2 {
		t.Fatalf("got %d windows, want 2", len(e.tree.Windows()))
	}
	if first.Buf != orig {
		t.Error("the first window's buffer changed")
	}
	if e.active == first || e.active.Buf.Path() != path {
		t.Errorf("the other window shows %q, want %q", e.active.Buf.Path(), path)
	}
	if e.yaziOther {
		t.Error("yaziOther left set")
	}
}

func TestYaziOtherWindowWithNothingChosenDoesNotSplit(t *testing.T) {
	fakeYazi(t)
	e, _ := newTestEditor(t)
	if err := e.yaziOtherWindow(); err != nil {
		t.Fatal(err)
	}
	if len(e.tree.Windows()) != 1 {
		t.Error("the frame was split although nothing was chosen")
	}
}

func TestYaziHandsAnOfficeFileToTheSystemApp(t *testing.T) {
	dir := t.TempDir()
	var docs []string
	for _, name := range []string{"report.docx", "sheet.xlsx"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte{0x50, 0x4b, 0, 3, 0, 0, 0, 0}, 0o644); err != nil {
			t.Fatal(err)
		}
		docs = append(docs, p)
	}
	oldOpen, oldAvail, oldRun := systemOpen, systemAvailable, yaziRun
	t.Cleanup(func() { systemOpen, systemAvailable, yaziRun = oldOpen, oldAvail, oldRun })
	var opened []string
	systemAvailable = func() error { return nil }
	systemOpen = func(path string, _ func(error)) error {
		opened = append(opened, path)
		return nil
	}
	runs := 0
	yaziRun = func(start, chooser string) error {
		runs++
		return os.WriteFile(chooser, []byte(docs[0]+"\n"+docs[1]+"\n"), 0o644)
	}
	e, _ := newTestEditor(t)
	before, n := e.active.Buf, len(e.buffers)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if runs != 1 {
		t.Errorf("yazi ran %d times, want once: it must not come back over an office file", runs)
	}
	if len(opened) != 2 || opened[0] != docs[0] || opened[1] != docs[1] {
		t.Errorf("system app opened %v, want %v", opened, docs)
	}
	if e.active.Buf != before || len(e.buffers) != n {
		t.Error("an office file was opened as a buffer")
	}
	if !e.yaziChosen {
		t.Error("yaziChosen is false: quitting nem would be wrong after a choice")
	}
}

// A program chosen in yazi is offered for running, not refused, and running it
// is a plain subprocess in the foreground: the system's opener is never asked.
func TestYaziOffersToRunAProgram(t *testing.T) {
	dir := t.TempDir()
	prog := filepath.Join(dir, "gui-app")
	if err := os.WriteFile(prog, []byte{0x7f, 'E', 'L', 'F', 0, 0, 0, 0}, 0o755); err != nil {
		t.Fatal(err)
	}
	oldOpen, oldAvail, oldRun, oldYazi := systemOpen, systemAvailable, systemRunWait, yaziRun
	t.Cleanup(func() { systemOpen, systemAvailable, systemRunWait, yaziRun = oldOpen, oldAvail, oldRun, oldYazi })
	systemAvailable = func() error { return nil }
	systemOpen = func(path string, _ func(error)) error {
		t.Errorf("the system opener was asked for %s; a program must be run directly", path)
		return nil
	}
	var ran []string
	systemRunWait = func(path string) error {
		ran = append(ran, path)
		return nil
	}
	runs := 0
	yaziRun = func(start, chooser string) error {
		runs++
		if runs > 1 {
			return os.WriteFile(chooser, nil, 0o644) // back in yazi, nothing chosen
		}
		return os.WriteFile(chooser, []byte(prog+"\n"), 0o644)
	}
	e, _ := newTestEditor(t)
	feedAnswer(t, e, "r")
	if err := e.yazi(); err != nil {
		t.Fatalf("yazi: %v", err)
	}
	if runs != 2 {
		t.Errorf("yazi ran %d times, want twice: once to choose, once to come back to after the program", runs)
	}
	if len(ran) != 1 || ran[0] != prog {
		t.Errorf("ran %v, want [%s]", ran, prog)
	}
}

// Answering n to the offer leaves the program alone and brings yazi back.
func TestYaziRefusesAProgramWhenAskedNo(t *testing.T) {
	dir := t.TempDir()
	prog := filepath.Join(dir, "gui-app")
	if err := os.WriteFile(prog, []byte{0x7f, 'E', 'L', 'F', 0, 0, 0, 0}, 0o755); err != nil {
		t.Fatal(err)
	}
	oldRun, oldYazi := systemRunWait, yaziRun
	t.Cleanup(func() { systemRunWait, yaziRun = oldRun, oldYazi })
	systemRunWait = func(path string) error {
		t.Errorf("%s was run after n", path)
		return nil
	}
	runs := 0
	yaziRun = func(start, chooser string) error {
		runs++
		if runs > 1 {
			return os.WriteFile(chooser, nil, 0o644)
		}
		return os.WriteFile(chooser, []byte(prog+"\n"), 0o644)
	}
	e, _ := newTestEditor(t)
	feedAnswer(t, e, "n")
	if err := e.yazi(); err != nil {
		t.Fatalf("yazi: %v", err)
	}
	if runs != 2 {
		t.Errorf("yazi ran %d times, want twice: back in yazi after the answer n", runs)
	}
}

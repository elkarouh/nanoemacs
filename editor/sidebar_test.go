package editor

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// sideProject makes a directory with a file, a subdirectory holding another,
// and opens the first file.
func sideProject(t *testing.T) (e *Editor, dir string) {
	t.Helper()
	dir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.txt", "sub/b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(f+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e, _ = newTestEditor(t)
	b, err := e.OpenFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(b)
	return e, dir
}

func sideText(e *Editor) string { return bufferString(e.sideBuf) }

// gotoSideLine puts point on the first panel line that contains s.
func gotoSideLine(t *testing.T, e *Editor, s string) {
	t.Helper()
	for i, l := range strings.Split(sideText(e), "\n") {
		if strings.Contains(l, s) {
			e.active.Pt.Line, e.active.Pt.Col = i, 0
			return
		}
	}
	t.Fatalf("no panel line contains %q in:\n%s", s, sideText(e))
}

func TestSidebarTogglesAsAWindow(t *testing.T) {
	e, _ := sideProject(t)
	main := e.active
	press(t, e, "C-b")
	if !e.sideOpen() || e.active != e.side {
		t.Fatal("C-b did not open the panel and move into it")
	}
	if got := sideText(e); !strings.Contains(got, "Tree") || !strings.Contains(got, "Bookmarks") {
		t.Errorf("panel lacks its tabs:\n%s", got)
	}
	press(t, e, "C-b")
	if e.sideOpen() || e.active != main {
		t.Error("C-b in the panel did not close it and go back")
	}
}

func TestSidebarTreeExpandsAndOpensFiles(t *testing.T) {
	e, dir := sideProject(t)
	main := e.active
	press(t, e, "C-b")
	if got := sideText(e); !strings.Contains(got, "a.txt") || !strings.Contains(got, "sub/") || strings.Contains(got, "b.txt") {
		t.Fatalf("tree should list a.txt and a closed sub/:\n%s", got)
	}
	gotoSideLine(t, e, "sub/")
	press(t, e, "RET")
	if !strings.Contains(sideText(e), "b.txt") {
		t.Fatalf("RET did not expand sub/:\n%s", sideText(e))
	}
	gotoSideLine(t, e, "b.txt")
	press(t, e, "RET")
	if e.active != main || e.active.Buf.Path() != filepath.Join(dir, "sub", "b.txt") {
		t.Errorf("RET on a file showed %q in the main window", e.active.Buf.Path())
	}
	if !e.sideOpen() {
		t.Error("opening a file closed the panel")
	}
}

func TestSidebarBookmarksFiles(t *testing.T) {
	e, dir := sideProject(t)
	press(t, e, "C-b")
	gotoSideLine(t, e, "a.txt")
	press(t, e, "b")
	want := filepath.Join(dir, "a.txt")
	if !slices.Contains(e.mem.Bookmarks, want) {
		t.Fatalf("bookmarks = %v, want %s in them", e.mem.Bookmarks, want)
	}
	press(t, e, "TAB")
	if !strings.Contains(sideText(e), "a.txt") {
		t.Fatalf("the Bookmarks tab does not list a.txt:\n%s", sideText(e))
	}
	gotoSideLine(t, e, "a.txt")
	press(t, e, "d")
	if len(e.mem.Bookmarks) != 0 {
		t.Errorf("d left the bookmark: %v", e.mem.Bookmarks)
	}
}

func TestBookmarkFileCommandTogglesTheCurrentFile(t *testing.T) {
	e, dir := sideProject(t)
	press(t, e, "C-x", "r", "f")
	if !slices.Contains(e.mem.Bookmarks, filepath.Join(dir, "a.txt")) {
		t.Fatalf("C-x r f did not bookmark the file: %v", e.mem.Bookmarks)
	}
	press(t, e, "C-x", "r", "f")
	if len(e.mem.Bookmarks) != 0 {
		t.Errorf("a second C-x r f left it: %v", e.mem.Bookmarks)
	}
}

func TestSidebarGoesAwayWithOtherWindowsDeleted(t *testing.T) {
	e, _ := sideProject(t)
	press(t, e, "C-b", "C-x", "o") // into the panel, then back out
	press(t, e, "C-x", "1")
	if e.sideOpen() {
		t.Error("C-x 1 left the panel")
	}
	press(t, e, "C-b")
	if !e.sideOpen() {
		t.Error("C-b did not open the panel again")
	}
}

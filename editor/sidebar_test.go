package editor

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Borderliner/nem/text"
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
	if got := sideText(e); !strings.Contains(got, "Tree") || !strings.Contains(got, "Favs") || !strings.Contains(got, "Bufs") || !strings.Contains(got, "Outline") {
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

func TestShiftArrowsMoveBetweenPanes(t *testing.T) {
	e, _ := sideProject(t)
	main := e.active
	press(t, e, "C-b") // into the panel, on the left
	press(t, e, "S-<right>")
	if e.active != main {
		t.Fatal("S-<right> from the panel did not reach the file's window")
	}
	press(t, e, "S-<left>")
	if e.active != e.side {
		t.Fatal("S-<left> did not go back to the panel")
	}
	press(t, e, "S-<left>")
	wantEcho(t, e, "No window to the left")
	if e.active != e.side {
		t.Error("moved with no window there")
	}
}

func TestShiftArrowsPickTheWindowOnThatSide(t *testing.T) {
	e, _ := sideProject(t)
	press(t, e, "C-x", "3") // two panes side by side
	right := e.active
	press(t, e, "S-<left>")
	left := e.active
	if left == right {
		t.Fatal("S-<left> stayed in place")
	}
	press(t, e, "S-<right>")
	if e.active != right {
		t.Error("S-<right> did not come back")
	}
}

func TestDirectoryBookmarksPersistAcrossSessions(t *testing.T) {
	state := t.TempDir()
	e, _ := newTestEditor(t)
	if err := e.UseStateDir(state); err != nil {
		t.Fatal(err)
	}
	e.setDirBookmark("proj", "/work/proj")
	e.setDirBookmark("old", "/work/old")
	e.deleteDirBookmark("old")

	e2, _ := newTestEditor(t)
	if err := e2.UseStateDir(state); err != nil {
		t.Fatal(err)
	}
	if got := e2.bookmarks["proj"]; got != "/work/proj" {
		t.Errorf("proj = %q after a restart, want /work/proj", got)
	}
	if _, ok := e2.bookmarks["old"]; ok {
		t.Error("a deleted bookmark came back")
	}
}

func TestSidebarBufferAndOutlineTabs(t *testing.T) {
	e, dir := sideProject(t)
	goFile := filepath.Join(dir, "m.go")
	if err := os.WriteFile(goFile, []byte("package m\n\nfunc One() {}\n\nfunc Two() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := e.OpenFile(goFile)
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(b)
	main := e.active
	press(t, e, "C-b", "3")
	if got := sideText(e); !strings.Contains(got, "a.txt") || !strings.Contains(got, "m.go") {
		t.Fatalf("the Buffers tab lacks the open files:\n%s", got)
	}
	gotoSideLine(t, e, "a.txt")
	press(t, e, "RET")
	if e.active != main || e.active.Buf.Path() != filepath.Join(dir, "a.txt") {
		t.Fatalf("RET on a buffer showed %q", e.active.Buf.Path())
	}
	press(t, e, "C-b", "4") // the outline of a.txt: nothing to list
	press(t, e, "2")
	press(t, e, "3")
	gotoSideLine(t, e, "m.go")
	press(t, e, "RET")
	press(t, e, "C-b", "4")
	if got := sideText(e); !strings.Contains(got, "One") || !strings.Contains(got, "Two") {
		t.Fatalf("the Outline tab lacks the definitions:\n%s", got)
	}
	gotoSideLine(t, e, "Two")
	press(t, e, "RET")
	if e.active != main || e.active.Pt.Line != 4 {
		t.Errorf("RET on a definition put point on line %d, want 4", e.active.Pt.Line)
	}
}

func TestSidebarOutlineFollowsTheFileInFront(t *testing.T) {
	e, dir := sideProject(t)
	goFile := filepath.Join(dir, "m.go")
	if err := os.WriteFile(goFile, []byte("package m\n\nfunc Only() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	press(t, e, "C-b", "4", "S-<right>")
	b, _ := e.OpenFile(goFile)
	e.VisitOpened(b)
	e.Redraw()
	if !strings.Contains(sideText(e), "Only") {
		t.Errorf("the outline did not follow the new file:\n%s", sideText(e))
	}
}

func TestSidebarFavoritesHoldLinksAndTools(t *testing.T) {
	e, scr := newTestEditor(t)
	if err := e.UseStateDir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	press(t, e, "C-b", "2")
	feed(t, scr, txt("w"), txt("Go"), key(t, "RET"), txt("https://go.dev"), key(t, "RET"))
	press(t, e, "a")
	feed(t, scr, txt("t"), txt("Test"), key(t, "RET"), txt("go test ./..."), key(t, "RET"))
	press(t, e, "a")
	got := sideText(e)
	for _, want := range []string{"Files", "Directories", "WWW", "Tools", "Go", "Test"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the Favorites tab lacks %q:\n%s", want, got)
		}
	}
	if e.mem.Links["Go"] != "https://go.dev" || e.mem.Tools["Test"] != "go test ./..." {
		t.Errorf("links %v tools %v", e.mem.Links, e.mem.Tools)
	}
	gotoSideLine(t, e, "WWW")
	press(t, e, "RET") // folds the group shut
	if strings.Contains(sideText(e), "Go") && strings.Contains(sideText(e), "https") {
		t.Error("the WWW group did not fold")
	}
	press(t, e, "RET")
	gotoSideLine(t, e, "  Go")
	press(t, e, "d")
	if _, ok := e.mem.Links["Go"]; ok {
		t.Error("d did not remove the link")
	}
}

func TestSidebarWithIconsAndClicks(t *testing.T) {
	e, _ := sideProject(t)
	e.SetIcons(true)
	press(t, e, "C-b")
	if !strings.ContainsRune(sideText(e), '\uf0f6') && !strings.ContainsRune(sideText(e), '\uf016') {
		t.Errorf("no file icon in the tree:\n%s", sideText(e))
	}
	e.Redraw()
	cols := e.sidebar.tabCols
	e.sidebarClick(text.Pos{Line: 0, Col: text.RuneIdx(cols[sideBuffers][0] + 1)})
	if e.sidebar.tab != sideBuffers {
		t.Errorf("clicking the third tab showed tab %d", e.sidebar.tab)
	}
}

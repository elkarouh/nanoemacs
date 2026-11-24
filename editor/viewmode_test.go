package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Borderliner/nem/keymap"
)

// yaziFile writes a text file of lines in a temporary directory, and returns
// its path, for yazi to choose.
func yaziFile(t *testing.T, lines int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a.txt")
	var sb strings.Builder
	for i := range lines {
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestYaziOpensAFileReadOnly(t *testing.T) {
	path := yaziFile(t, 1)
	fakeYazi(t, path)
	e, _ := newTestEditor(t)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if !e.active.Buf.ReadOnly() {
		t.Fatal("a file opened from yazi is editable, want read-only")
	}
	before := e.Buf().String()
	press(t, e, "x")
	wantText(t, e, before)
}

func TestViewEditMakesTheFileEditable(t *testing.T) {
	path := yaziFile(t, 1)
	fakeYazi(t, path)
	e, _ := newTestEditor(t)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	press(t, e, "e")
	if e.active.Buf.ReadOnly() {
		t.Fatal("e did not make the file editable")
	}
	press(t, e, "x")
	if !strings.HasPrefix(e.Buf().String(), "xline") {
		t.Errorf("typing after e did not change the file: %q", e.Buf().String())
	}
}

func TestOpenFileAlreadyOpenKeepsItsState(t *testing.T) {
	path := yaziFile(t, 1)
	fakeYazi(t, path)
	e, _ := newTestEditor(t)
	if _, err := e.OpenFile(path); err != nil {
		t.Fatal(err)
	}
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	if e.active.Buf.ReadOnly() {
		t.Error("a file already open became read-only when chosen again in yazi")
	}
}

func TestKeypadNineAndSevenAreBoundInTheView(t *testing.T) {
	// The screen in tests has no height to page by, so this checks the bindings
	// the way they are read: 9 is PgUp's command, and 7 is Home's.
	if got := newTestViewKeymap(t).Where("scroll-down-command"); !containsString(got, "9") {
		t.Errorf("9 is not bound to scroll-down-command: %v", got)
	}
	if got := newTestViewKeymap(t).Where("move-beginning-of-line"); !containsString(got, "7") {
		t.Errorf("7 is not bound to move-beginning-of-line: %v", got)
	}
}

func TestSevenGoesHomeInTheView(t *testing.T) {
	path := yaziFile(t, 3)
	fakeYazi(t, path)
	e, _ := newTestEditor(t)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	before := e.Buf().String()
	press(t, e, "C-e", "7")
	wantPt(t, e, 0, 0)
	wantText(t, e, before)
}

// newTestViewKeymap is the view keymap, built as an editor builds it.
func newTestViewKeymap(t *testing.T) *keymap.Map {
	t.Helper()
	m, err := newViewKeymap()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

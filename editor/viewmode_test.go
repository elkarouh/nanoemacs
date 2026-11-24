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

func TestKeypadKeysAreBoundInTheViewAndThePreview(t *testing.T) {
	view := newTestViewKeymap(t)
	preview, err := newPreviewKeymap()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range keypadBindings {
		for name, m := range map[string]*keymap.Map{"view": view, "preview": preview} {
			if got := m.Where(b.Command); !containsString(got, b.Spec) {
				t.Errorf("%s: %s is not bound to %s: %v", name, b.Spec, b.Command, got)
			}
		}
	}
}

func TestSevenGoesToTheStartInTheView(t *testing.T) {
	path := yaziFile(t, 3)
	fakeYazi(t, path)
	e, _ := newTestEditor(t)
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	before := e.Buf().String()
	press(t, e, "C-n", "C-n", "C-e", "7")
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

func TestKillingAViewedFileForgetsIt(t *testing.T) {
	path := yaziFile(t, 1)
	fakeYazi(t, path)
	e, _ := newTestEditor(t)
	if _, err := e.OpenFile(filepath.Join(t.TempDir(), "other.txt")); err != nil {
		t.Fatal(err)
	}
	if err := e.yazi(); err != nil {
		t.Fatal(err)
	}
	b := e.active.Buf
	if !e.viewing[b] {
		t.Fatal("the yazi file is not in view mode")
	}
	if err := e.KillBuffer(b); err != nil {
		t.Fatal(err)
	}
	if e.viewing[b] {
		t.Error("a killed buffer is still remembered as viewed")
	}
}

func TestCtrlHomeAndEndAreBoundInTheView(t *testing.T) {
	m := newTestViewKeymap(t)
	if got := m.Where("beginning-of-buffer"); !containsString(got, "C-<home>") {
		t.Errorf("C-<home> is not bound to beginning-of-buffer in the view: %v", got)
	}
	if got := m.Where("end-of-buffer"); !containsString(got, "C-<end>") {
		t.Errorf("C-<end> is not bound to end-of-buffer in the view: %v", got)
	}
}

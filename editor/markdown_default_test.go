package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Borderliner/nem/text"
)

// TestMarkdownFileOpensAsPreview: a .md file opens in its preview, and q turns
// the preview back into the editable file.
func TestMarkdownFileOpensAsPreview(t *testing.T) {
	fakeTool(t, "mdcat", `printf 'RENDERED\n'`)
	e, _ := newTestEditor(t)
	path := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(path, []byte("# Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := e.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(src)

	if e.previewOf(e.active.Buf) == nil {
		t.Fatalf("active buffer is %q, not a Markdown preview", e.active.Buf.Path())
	}
	if got := bufferString(e.active.Buf); got != "RENDERED\n" {
		t.Errorf("preview shows %q, want the rendering", got)
	}

	press(t, e, "q")
	if e.previewOf(e.active.Buf) != nil {
		t.Fatal("q left the preview showing")
	}
	if e.active.Buf != src {
		t.Errorf("q showed a different buffer, want the source file's")
	}
	if got := bufferString(e.active.Buf); got != "# Notes" {
		t.Errorf("after q the buffer reads %q, want the file's text", got)
	}
}

// TestOtherFilesOpenUnchanged: only .md files open as previews.
func TestOtherFilesOpenUnchanged(t *testing.T) {
	fakeTool(t, "mdcat", `printf 'RENDERED\n'`)
	e, _ := newTestEditor(t)
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("plain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := e.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(src)
	if e.active.Buf != src || e.previewOf(src) != nil {
		t.Errorf("a .txt file opened as a preview, want the file itself")
	}
}

// TestMarkdownOpensAsFileWithoutMdcat: with no mdcat the .md file opens as its
// text, since the preview is only a convenience.
func TestMarkdownOpensAsFileWithoutMdcat(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	e, _ := newTestEditor(t)
	path := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(path, []byte("# Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := e.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(src)
	if e.active.Buf != src {
		t.Errorf("without mdcat the .md file did not open as its text")
	}
}

// bufferString is the whole text of b.
func bufferString(b *text.Buffer) string {
	return string(b.Text(text.Pos{}, b.End()))
}

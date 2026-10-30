package editor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// onlyTool puts a fake tool named name on an otherwise empty PATH, so that no
// other renderer, installed on the machine running the tests, can be found.
// The script runs cat as /bin/cat, as the PATH holds nothing else.
func onlyTool(t *testing.T, name, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake tools are shell scripts")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

// The HTML preview hands the page to lynx on its stdin, and shows what lynx
// prints in the source's place.
func TestHTMLPreviewShowsLynxOutput(t *testing.T) {
	onlyTool(t, "lynx", `printf 'LYNX:'; /bin/cat`)
	e, _ := newTestEditor(t, "<p>hi</p>")
	if err := e.showRendered(e.Buf(), htmlRender); err != nil {
		t.Fatal(err)
	}
	if e.previewOf(e.active.Buf) == nil {
		t.Fatalf("active buffer is %q, not a preview", e.BufferName(e.active.Buf))
	}
	if got := bufferString(e.active.Buf); got != "LYNX:<p>hi</p>" {
		t.Errorf("preview shows %q, want the page as lynx printed it", got)
	}
}

// Without lynx the preview says so, and the buffer is left as it is.
func TestHTMLPreviewNeedsLynx(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	e, _ := newTestEditor(t, "<p>hi</p>")
	before := e.active.Buf
	if err := e.showRendered(e.Buf(), htmlRender); err == nil {
		t.Fatal("no error without lynx")
	}
	if e.active.Buf != before || e.previewOf(before) != nil {
		t.Error("the buffer changed although no preview was made")
	}
}

// C-x x h shows the page as the preview, as a user would press it.
func TestHTMLPreviewKey(t *testing.T) {
	onlyTool(t, "lynx", `printf 'LYNX:'; /bin/cat`)
	e, _ := newTestEditor(t, "<p>hi</p>")
	press(t, e, "C-x x h")
	if e.previewOf(e.active.Buf) == nil {
		t.Fatalf("C-x x h did not show a preview of %q", e.BufferName(e.Buf()))
	}
	if got := bufferString(e.active.Buf); got != "LYNX:<p>hi</p>" {
		t.Errorf("preview shows %q, want the page as lynx printed it", got)
	}
}

// While a preview is showing, the directory of its source still counts: a
// shell left by quitting nem with the preview up must be in the file's
// directory, not the working directory.
func TestPreviewKeepsTheSourceDirectory(t *testing.T) {
	onlyTool(t, "lynx", `printf 'LYNX:'; /bin/cat`)
	e, _ := newTestEditor(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "page.html")
	if err := os.WriteFile(path, []byte("<p>hi</p>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := e.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	e.active.Visit(src)
	if err := e.showRendered(src, htmlRender); err != nil {
		t.Fatal(err)
	}
	if e.active.Buf == src {
		t.Fatal("the preview is not showing")
	}
	if got := e.ExitDir(); got != dir {
		t.Errorf("ExitDir with the preview showing = %q, want %q", got, dir)
	}
}

// An .html file opens as its preview, as an .md file does, and q turns the
// preview into the source to edit.
func TestHTMLFileOpensAsPreview(t *testing.T) {
	onlyTool(t, "lynx", `printf 'LYNX:'; /bin/cat`)
	e, _ := newTestEditor(t)
	path := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(path, []byte("<p>hi</p>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := e.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(src)

	if e.previewOf(e.active.Buf) == nil {
		t.Fatalf("active buffer is %q, not an HTML preview", e.BufferName(e.active.Buf))
	}
	if got := bufferString(e.active.Buf); got != "LYNX:<p>hi</p>" {
		t.Errorf("preview shows %q, want lynx's output", got)
	}
	press(t, e, "q")
	if e.active.Buf != src {
		t.Errorf("q showed %q, want the source", e.BufferName(e.active.Buf))
	}
}

// Without lynx an .html file opens as its text rather than failing.
func TestHTMLFileOpensAsTextWithoutLynx(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	e, _ := newTestEditor(t)
	path := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(path, []byte("<p>hi</p>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := e.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	e.VisitOpened(src)
	if e.active.Buf != src {
		t.Errorf("active buffer is %q, want the source", e.BufferName(e.active.Buf))
	}
}

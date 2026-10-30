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

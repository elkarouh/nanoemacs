package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// pdfTurn is the page each key goes to. Arrows and Page keys move a page and
// stay within the PDF; Home and End go to its ends; any other key does not turn.
func TestPDFTurn(t *testing.T) {
	tests := []struct {
		page  int
		key   tcell.Key
		want  int
		turns bool
	}{
		{1, tcell.KeyRight, 2, true},
		{2, tcell.KeyDown, 3, true},
		{5, tcell.KeyPgDn, 5, true}, // stays on the last page
		{3, tcell.KeyPgDn, 4, true},
		{2, tcell.KeyLeft, 1, true},
		{1, tcell.KeyUp, 1, true}, // stays on the first page
		{3, tcell.KeyPgUp, 2, true},
		{2, tcell.KeyHome, 1, true},
		{2, tcell.KeyEnd, 5, true},
		{2, tcell.KeyEscape, 2, false}, // closes the viewer
		{2, tcell.KeyRune, 2, false},
	}
	for _, tt := range tests {
		got, turned := pdfTurn(tt.page, 5, tt.key)
		if got != tt.want || turned != tt.turns {
			t.Errorf("pdfTurn(%d, 5, %v) = %d, %v; want %d, %v", tt.page, tt.key, got, turned, tt.want, tt.turns)
		}
	}
}

func TestPDFPagesReadsPdfinfo(t *testing.T) {
	fakeTool(t, "pdfinfo", `printf 'Title:          paper\nPages:          7\nPage size:      612 x 792 pts\n'`)
	n, err := pdfPages(filepath.Join(t.TempDir(), "paper.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 7 {
		t.Errorf("pages = %d, want 7", n)
	}
}

// renderPDFPage asks pdftoppm for the one page, and returns the PNG it wrote.
// The fake records its arguments and writes a PNG beside the base name it gets.
func TestRenderPDFPageAsksForThatPage(t *testing.T) {
	dir := t.TempDir()
	args := filepath.Join(dir, "args")
	src := filepath.Join(dir, "src.png")
	if err := os.WriteFile(src, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeTool(t, "pdftoppm", `echo "$@" > '`+args+`'; for last; do :; done; cp '`+src+`' "$last.png"`)
	png, err := renderPDFPage("paper.pdf", 4, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(png, "page-4.png") {
		t.Errorf("rendered %q, want page-4.png", png)
	}
	if data, err := os.ReadFile(png); err != nil || string(data) != "PNG" {
		t.Fatalf("reading the page: %q, %v", data, err)
	}
	got, _ := os.ReadFile(args)
	if !strings.Contains(string(got), "-f 4 -l 4") {
		t.Errorf("pdftoppm was given %q, want the one page 4", got)
	}
}

func TestPDFViewerNeedsPoppler(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := pdfPages("paper.pdf"); err == nil {
		t.Fatal("no error without pdfinfo")
	}
}

func TestIsPDFFileIgnoresCase(t *testing.T) {
	if !isPDFFile("Paper.PDF") || isPDFFile("paper.txt") {
		t.Error("isPDFFile misjudged an extension")
	}
}

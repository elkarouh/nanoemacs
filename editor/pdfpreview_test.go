package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// The viewer's keys, as the events a terminal sends for them.
func TestPDFKeysAreEmacsLike(t *testing.T) {
	tests := []struct {
		name string
		ev   *tcell.EventKey
		want pdfAct
	}{
		{"C-n next", tcell.NewEventKey(tcell.KeyCtrlN, 0, tcell.ModCtrl), pdfNext},
		{"C-v next", tcell.NewEventKey(tcell.KeyCtrlV, 0, tcell.ModCtrl), pdfNext},
		{"Page Down next", tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone), pdfNext},
		{"C-p prev", tcell.NewEventKey(tcell.KeyCtrlP, 0, tcell.ModCtrl), pdfPrev},
		{"M-v prev", tcell.NewEventKey(tcell.KeyRune, 'v', tcell.ModAlt), pdfPrev},
		{"M-< first", tcell.NewEventKey(tcell.KeyRune, '<', tcell.ModAlt), pdfFirst},
		{"M-> last", tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModAlt), pdfLast},
		{"M-g goes to a page", tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModAlt), pdfGoto},
		{"g goes to a page too", tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone), pdfGoto},
		{"+ zooms in", tcell.NewEventKey(tcell.KeyRune, '+', tcell.ModNone), pdfZoomIn},
		{"- zooms out", tcell.NewEventKey(tcell.KeyRune, '-', tcell.ModNone), pdfZoomOut},
		{"0 fits the width", tcell.NewEventKey(tcell.KeyRune, '0', tcell.ModNone), pdfFit},
		{"a letter closes", tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone), pdfClose},
		{"Escape closes", tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), pdfClose},
	}
	for _, tt := range tests {
		if got := pdfKeyAct(tt.ev); got != tt.want {
			t.Errorf("%s: act = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// Moves stay within the PDF's first and last page.
func TestPDFTurnStaysWithinThePages(t *testing.T) {
	tests := []struct {
		page int
		act  pdfAct
		want int
	}{
		{1, pdfNext, 2},
		{5, pdfNext, 5},
		{1, pdfPrev, 1},
		{3, pdfPrev, 2},
		{2, pdfFirst, 1},
		{2, pdfLast, 5},
		{2, pdfZoomIn, 2}, // not a move
	}
	for _, tt := range tests {
		if got := pdfTurn(tt.page, 5, tt.act); got != tt.want {
			t.Errorf("pdfTurn(%d, 5, %d) = %d, want %d", tt.page, tt.act, got, tt.want)
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

// renderPDFPage asks pdftoppm for the one page, at the size asked for, and
// returns the PNG it wrote. The fake records its arguments and writes a PNG
// beside the base name it gets last, as the real one does with -singlefile.
func TestRenderPDFPageAsksForThatPageAtThatSize(t *testing.T) {
	dir := t.TempDir()
	args := filepath.Join(dir, "args")
	src := filepath.Join(dir, "src.png")
	if err := os.WriteFile(src, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeTool(t, "pdftoppm", `echo "$@" > '`+args+`'; for last; do :; done; cp '`+src+`' "$last.png"`)

	png, err := renderPDFPage("paper.pdf", 4, pdfSize{dpi: 150}, filepath.Join(dir, "p4"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(png, "p4.png") {
		t.Errorf("rendered %q, want p4.png", png)
	}
	got, _ := os.ReadFile(args)
	if !strings.Contains(string(got), "-f 4 -l 4") || !strings.Contains(string(got), "-r 150") {
		t.Errorf("pdftoppm was given %q, want page 4 at 150 dpi", got)
	}

	if _, err := renderPDFPage("paper.pdf", 4, pdfSize{dpi: 150, fitWidth: 900}, filepath.Join(dir, "f4")); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(args)
	if !strings.Contains(string(got), "-scale-to-x 900 -scale-to-y -1") {
		t.Errorf("pdftoppm was given %q, want the page scaled to 900 pixels wide", got)
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

package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// The viewer's keys, as the events a terminal sends for them.
func TestPDFKeys(t *testing.T) {
	tests := []struct {
		name string
		ev   *tcell.EventKey
		want pdfAct
	}{
		{"Right next", tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), pdfNext},
		{"Down next", tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), pdfNext},
		{"Page Down next", tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone), pdfNext},
		{"space next", tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone), pdfNext},
		{"C-f next", tcell.NewEventKey(tcell.KeyCtrlF, 0, tcell.ModCtrl), pdfNext},
		{"C-v next", tcell.NewEventKey(tcell.KeyCtrlV, 0, tcell.ModCtrl), pdfNext},
		{"Left prev", tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), pdfPrev},
		{"Page Up prev", tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone), pdfPrev},
		{"C-b prev", tcell.NewEventKey(tcell.KeyCtrlB, 0, tcell.ModCtrl), pdfPrev},
		{"Home first", tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone), pdfFirst},
		{"M-< first", tcell.NewEventKey(tcell.KeyRune, '<', tcell.ModAlt), pdfFirst},
		{"End last", tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone), pdfLast},
		{"M-> last", tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModAlt), pdfLast},
		{"a letter closes", tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone), pdfClose},
		{"Escape closes", tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), pdfClose},
		{"C-n closes", tcell.NewEventKey(tcell.KeyCtrlN, 0, tcell.ModCtrl), pdfClose},
		{"M-g closes", tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModAlt), pdfClose},
		{"0 closes", tcell.NewEventKey(tcell.KeyRune, '0', tcell.ModNone), pdfClose},
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

// renderPDFPage asks pdftoppm for the one page and returns the PNG it wrote.
// The fake records its arguments and writes a PNG beside the base name it gets
// last, as the real one does with -singlefile.
func TestRenderPDFPageAsksForThatPage(t *testing.T) {
	dir := t.TempDir()
	args := filepath.Join(dir, "args")
	src := filepath.Join(dir, "src.png")
	if err := os.WriteFile(src, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeTool(t, "pdftoppm", `echo "$@" > '`+args+`'; for last; do :; done; cp '`+src+`' "$last.png"`)

	png, err := renderPDFPage("paper.pdf", 4, filepath.Join(dir, "p4"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(png, "p4.png") {
		t.Errorf("rendered %q, want p4.png", png)
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

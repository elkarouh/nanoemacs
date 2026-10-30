package editor

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Borderliner/nem/text"
)

func TestEPUBFiles(t *testing.T) {
	if !isEPUBFile("Book.EPUB") || isEPUBFile("book.pdf") {
		t.Error("isEPUBFile misjudged an extension")
	}
}

// makeEPUB writes a one-chapter EPUB to a new file in a temporary directory,
// made by pandoc, and returns its path. The test is skipped without pandoc.
func makeEPUB(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc is not installed")
	}
	book := filepath.Join(t.TempDir(), "book.epub")
	cmd := exec.Command("pandoc", "--output="+book)
	cmd.Stdin = strings.NewReader("# Chapter\n\nHello, reader.\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pandoc could not make a test EPUB: %v\n%s", err, out)
	}
	return book
}

func TestEPUBToHTML(t *testing.T) {
	html, err := epubHTML(makeEPUB(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "Hello, reader.") {
		t.Errorf("the HTML does not hold the book's text: %s", html)
	}
}

func TestOpenEPUBShowsItsText(t *testing.T) {
	e, _ := newTestEditor(t)
	b, err := e.OpenFile(makeEPUB(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b.Text(text.Pos{}, b.End())); !strings.Contains(got, "Hello, reader.") {
		t.Errorf("the EPUB's buffer does not show its text: %q", got)
	}
	if b.Path() != "" {
		t.Errorf("the converted book has a file behind it: %q", b.Path())
	}
}

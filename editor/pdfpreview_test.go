package editor

import (
	"os"
	"path/filepath"
	"testing"
)

// The fake pdftoppm writes a PNG beside the base name it is given last, as the
// real one does with -singlefile.
const fakePdftoppm = `for last; do :; done; printf PNG > "$last.png"`

func TestPDFPageRendersAPNGAndCleansUp(t *testing.T) {
	fakeTool(t, "pdftoppm", fakePdftoppm)
	png, cleanup, err := pdfPage(filepath.Join(t.TempDir(), "paper.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(png) != ".png" {
		t.Errorf("rendered %q, want a PNG", png)
	}
	if data, err := os.ReadFile(png); err != nil || string(data) != "PNG" {
		t.Fatalf("reading the page: %q, %v", data, err)
	}
	dir := filepath.Dir(png)
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the page's directory survived cleanup: %v", err)
	}
}

func TestPDFPageNeedsPdftoppm(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, _, err := pdfPage("paper.pdf"); err == nil {
		t.Fatal("no error without pdftoppm")
	}
}

func TestIsPDFFileIgnoresCase(t *testing.T) {
	if !isPDFFile("Paper.PDF") || isPDFFile("paper.txt") {
		t.Error("isPDFFile misjudged an extension")
	}
}

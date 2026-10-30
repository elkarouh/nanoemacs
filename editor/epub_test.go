package editor

import (
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
)

func TestEPUBFiles(t *testing.T) {
	if !isEPUBFile("Book.EPUB") || isEPUBFile("book.pdf") {
		t.Error("isEPUBFile misjudged an extension")
	}
}

// makeEPUB writes an EPUB of markdown to a new file, made by pandoc, and returns
// its path. Files the markdown names are read from dir. The test is skipped
// without pandoc.
func makeEPUB(t *testing.T, dir, markdown string) string {
	t.Helper()
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc is not installed")
	}
	book := filepath.Join(dir, "book.epub")
	cmd := exec.Command("pandoc", "--resource-path="+dir, "--output="+book)
	cmd.Stdin = strings.NewReader(markdown)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pandoc could not make a test EPUB: %v\n%s", err, out)
	}
	return book
}

// writePNG writes a solid red w by h PNG to dir/pic.png.
func writePNG(t *testing.T, dir string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+3] = 0xff, 0xff
	}
	f, err := os.Create(filepath.Join(dir, "pic.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestEPUBToHTML(t *testing.T) {
	book := makeEPUB(t, t.TempDir(), "# Chapter\n\nHello, reader.\n")
	html, files, err := epubHTML(book, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "Hello, reader.") {
		t.Errorf("the HTML does not hold the book's text: %s", html)
	}
	if len(files) != 0 {
		t.Errorf("a book with no pictures gave %d picture files", len(files))
	}
}

func TestOpenEPUBShowsItsText(t *testing.T) {
	e, _ := newTestEditor(t)
	b, err := e.OpenFile(makeEPUB(t, t.TempDir(), "# Chapter\n\nHello, reader.\n"))
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

func TestEPUBPicturesKeepTheirRows(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, dir, 800, 400)
	book := makeEPUB(t, dir, "# Chapter\n\nBefore.\n\n![pic](pic.png)\n\nAfter.\n")
	e, _ := newTestEditor(t)
	b, err := e.OpenFile(book)
	if err != nil {
		t.Fatal(err)
	}
	st := e.previewOf(b)
	if st == nil {
		t.Fatal("the EPUB opened without its preview")
	}
	if len(st.pictures) != 1 {
		t.Fatalf("got %d pictures, want 1", len(st.pictures))
	}
	p := st.pictures[0]
	// An 800 by 400 picture is 2:1, so at the preview width of 80 cells it is
	// 20 rows: 400 pixels tall at 20 pixels a row.
	if p.Cols != previewWidth || p.Rows != 20 || p.png == nil {
		t.Fatalf("picture box %d by %d, png %d bytes; want %d by 20", p.Cols, p.Rows, len(p.png), previewWidth)
	}
	got := string(b.Text(text.Pos{}, b.End()))
	if strings.Contains(got, "NEMPICTURE") {
		t.Errorf("a picture marker was left in the preview: %q", got)
	}
	lines := strings.Split(got, "\n")
	for i := p.Line; i < p.Line+p.Rows; i++ {
		if lines[i] != "" {
			t.Errorf("line %d is in the picture's rows but holds %q", i, lines[i])
		}
	}
}

func TestPictureSequenceDrawsOnlyWhereItFits(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, dir, 100, 50)
	book := makeEPUB(t, dir, "# Chapter\n\n![pic](pic.png)\n")
	e, _ := newTestEditor(t)
	b, err := e.OpenFile(book)
	if err != nil {
		t.Fatal(err)
	}
	e.active.Visit(b)
	st := e.previewOf(b)
	if len(st.pictures) != 1 || st.pictures[0].Rows == 0 {
		t.Fatal("the test book has no picture to draw")
	}
	const w, h = 80, 24

	seq := e.pictureSequence(w, h, false, true)
	if !strings.Contains(seq, "a=t,") || !strings.Contains(seq, "a=p,") {
		t.Errorf("the picture is not sent and placed: %q", seq)
	}
	if again := e.pictureSequence(w, h, false, true); strings.Contains(again, "a=t,") || !strings.Contains(again, "a=p,") {
		t.Errorf("a redraw should place the picture again without sending it: %q", again)
	}
	if seq := e.pictureSequence(w, h, true, true); !strings.Contains(seq, ui.KittyClearPlacements) || strings.Contains(seq, "a=p,") {
		t.Errorf("a panel should clear the pictures and draw none: %q", seq)
	}
	if seq := e.pictureSequence(w, h, false, false); seq != "" {
		t.Errorf("without kitty nothing is drawn: %q", seq)
	}
}

package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Borderliner/nem/ui"
	"github.com/gdamore/tcell/v2"
)

// A PDF chosen in yazi opens in a viewer that draws one page at a time, each
// rendered by pdftoppm, over the window. The arrow keys and Page Up and Page
// Down turn pages, Home and End go to the first and last, and any other key
// closes the viewer back to yazi. It needs the same terminal as a picture;
// elsewhere yazi refuses a PDF as any binary file.

// isPDFFile reports whether path names a PDF.
func isPDFFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".pdf")
}

// pictureOrPDF reports whether every path is a picture or a PDF, which yazi
// shows over the window rather than refusing.
func pictureOrPDF(paths []string) bool {
	for _, p := range paths {
		if !isImageFile(p) && !isPDFFile(p) {
			return false
		}
	}
	return true
}

// showPicture shows a picture, or a PDF in its viewer, over the window.
func (e *Editor) showPicture(path string) error {
	if isPDFFile(path) {
		return e.browsePDF(path)
	}
	return e.showImage(path)
}

// browsePDF shows the PDF at path page by page, until a key that does not turn
// a page closes it.
func (e *Editor) browsePDF(path string) error {
	pages, err := pdfPages(path)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "nem-pdf-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	page := 1
	for {
		png, err := renderPDFPage(path, page, dir)
		if err != nil {
			return err
		}
		if _, _, err := e.drawImage(png); err != nil {
			return err
		}
		e.Echo("%s, page %d of %d - arrows and Page keys turn, any other key closes", filepath.Base(path), page, pages)
		e.Redraw()
		key, ok := e.nextKey()
		ui.WriteTTY(e.scr, ui.KittyClear)
		next, turned := pdfTurn(page, pages, key)
		if !ok || !turned {
			break
		}
		page = next
	}
	e.Echo("")
	e.scr.Sync()
	return nil
}

// pdfTurn is the page a key goes to from page, of pages. Arrows and Page keys
// turn one page, staying within the first and last, and Home and End go to
// those. turned is false for a key that does not turn pages.
func pdfTurn(page, pages int, key tcell.Key) (next int, turned bool) {
	switch key {
	case tcell.KeyRight, tcell.KeyDown, tcell.KeyPgDn:
		return min(page+1, pages), true
	case tcell.KeyLeft, tcell.KeyUp, tcell.KeyPgUp:
		return max(page-1, 1), true
	case tcell.KeyHome:
		return 1, true
	case tcell.KeyEnd:
		return pages, true
	}
	return page, false
}

// pdfPages reports how many pages the PDF at path has, as pdfinfo says.
func pdfPages(path string) (int, error) {
	pdfinfo, err := exec.LookPath("pdfinfo")
	if err != nil {
		return 0, fmt.Errorf("the PDF view needs pdfinfo and pdftoppm, from poppler, which are not on the PATH")
	}
	out, err := exec.Command(pdfinfo, path).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("pdfinfo: %s", vcFirstLine(string(out), err))
	}
	for _, line := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(line, "Pages:"); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				return n, nil
			}
		}
	}
	return 0, fmt.Errorf("pdfinfo did not say how many pages %s has", filepath.Base(path))
}

// renderPDFPage renders page of the PDF at path to a PNG in dir, with pdftoppm,
// and returns the PNG's path.
func renderPDFPage(path string, page int, dir string) (string, error) {
	pdftoppm, err := exec.LookPath("pdftoppm")
	if err != nil {
		return "", fmt.Errorf("the PDF view needs pdftoppm, from poppler, which is not on the PATH")
	}
	n := strconv.Itoa(page)
	base := filepath.Join(dir, "page-"+n)
	out, err := exec.Command(pdftoppm, "-png", "-singlefile", "-f", n, "-l", n,
		"-r", "150", path, base).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pdftoppm: %s", vcFirstLine(string(out), err))
	}
	return base + ".png", nil
}

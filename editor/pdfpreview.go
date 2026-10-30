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
// rendered by pdftoppm, over the window. Its keys follow emacs: C-n, C-f, C-v
// and Page Down go to the next page, C-p, C-b, M-v and Page Up to the previous,
// M-< and M-> to the first and last, and M-g (or plain g, for a terminal that
// does not send Meta) goes to a page the number of which is typed. + and - zoom, and 0 fits the page to the window's width. Any other
// key closes the viewer back to yazi. It needs the same terminal as a picture;
// elsewhere yazi refuses a PDF as any binary file.

// pdfAct is what a key does in the PDF viewer.
type pdfAct int

const (
	pdfClose pdfAct = iota
	pdfNext
	pdfPrev
	pdfFirst
	pdfLast
	pdfGoto
	pdfZoomIn
	pdfZoomOut
	pdfFit
)

// pdfDefaultDPI is the resolution a page is rendered at when it is not fitted
// to the window's width.
const pdfDefaultDPI = 150

// pdfKeyAct is what a key press does in the PDF viewer.
func pdfKeyAct(ev *tcell.EventKey) pdfAct {
	alt := ev.Modifiers()&tcell.ModAlt != 0
	if alt {
		switch ev.Rune() {
		case 'v':
			return pdfPrev
		case '<':
			return pdfFirst
		case '>':
			return pdfLast
		case 'g':
			return pdfGoto
		}
		return pdfClose
	}
	switch ev.Key() {
	case tcell.KeyRight, tcell.KeyDown, tcell.KeyPgDn, tcell.KeyCtrlN, tcell.KeyCtrlF, tcell.KeyCtrlV:
		return pdfNext
	case tcell.KeyLeft, tcell.KeyUp, tcell.KeyPgUp, tcell.KeyCtrlP, tcell.KeyCtrlB:
		return pdfPrev
	case tcell.KeyHome:
		return pdfFirst
	case tcell.KeyEnd:
		return pdfLast
	case tcell.KeyRune:
		switch ev.Rune() {
		case ' ':
			return pdfNext
		case '+', '=':
			return pdfZoomIn
		case '-':
			return pdfZoomOut
		case '0':
			return pdfFit
		case 'g':
			return pdfGoto
		}
	}
	return pdfClose
}

// pdfTurn is the page an action goes to from page, of pages. Moves stay within
// the first and last page.
func pdfTurn(page, pages int, act pdfAct) int {
	switch act {
	case pdfNext:
		return min(page+1, pages)
	case pdfPrev:
		return max(page-1, 1)
	case pdfFirst:
		return 1
	case pdfLast:
		return pages
	}
	return page
}

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

// pdfSize is how a page is rendered: at dpi, or, when fitWidth is more than
// zero, scaled to fitWidth pixels wide.
type pdfSize struct {
	dpi      int
	fitWidth int
}

// browsePDF shows the PDF at path page by page, until a key that is not a page
// key closes it.
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
	// The picture stays up while keys are read, the page prompt included, and
	// is cleared once the viewer closes.
	defer ui.WriteTTY(e.scr, ui.KittyClear)

	page, size, seq := 1, pdfSize{dpi: pdfDefaultDPI}, 0
	for {
		seq++
		png, err := renderPDFPage(path, page, size, filepath.Join(dir, strconv.Itoa(seq)))
		if err != nil {
			return err
		}
		// One picture at a time: the last one is cleared before the next is drawn.
		ui.WriteTTY(e.scr, ui.KittyClear)
		if _, _, err := e.drawImage(png); err != nil {
			return err
		}
		e.Echo("%s, page %d of %d - n p M-< M-> M-g + - 0; any other key closes", filepath.Base(path), page, pages)
		e.Redraw()
		ev, ok := e.nextKeyEvent()
		if !ok {
			break
		}
		act := pdfKeyAct(ev)
		switch act {
		case pdfClose:
			e.scr.Sync()
			e.Echo("")
			return nil
		case pdfGoto:
			if n, ok := e.askPage(pages); ok {
				page = n
			}
		case pdfZoomIn:
			size = pdfSize{dpi: min(size.dpi*5/4+1, 600)}
		case pdfZoomOut:
			size = pdfSize{dpi: max(size.dpi*4/5, 30)}
		case pdfFit:
			size = e.fitSize()
		default:
			page = pdfTurn(page, pages, act)
		}
	}
	e.Echo("")
	e.scr.Sync()
	return nil
}

// fitSize is a page scaled to the window's width, or at the default resolution
// where the terminal does not report its cell size.
func (e *Editor) fitSize() pdfSize {
	w, _ := e.scr.Size()
	if cw, _ := ui.CellPixels(e.scr); cw > 0 {
		return pdfSize{dpi: pdfDefaultDPI, fitWidth: w * cw}
	}
	return pdfSize{dpi: pdfDefaultDPI}
}

// askPage reads the number of a page, typed and ended with Enter, and reports
// it when it is one of the pages. Escape or C-g gives up.
func (e *Editor) askPage(pages int) (int, bool) {
	num := ""
	for {
		e.Echo("Go to page (1-%d): %s", pages, num)
		e.Redraw()
		ev, ok := e.nextKeyEvent()
		if !ok {
			return 0, false
		}
		switch {
		case ev.Key() == tcell.KeyEnter:
			n, err := strconv.Atoi(num)
			if err != nil || n < 1 || n > pages {
				e.Echo("no page %q in a PDF of %d pages", num, pages)
				return 0, false
			}
			return n, true
		case ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlG:
			return 0, false
		case ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2:
			if num != "" {
				num = num[:len(num)-1]
			}
		case ev.Key() == tcell.KeyRune && ev.Rune() >= '0' && ev.Rune() <= '9':
			num += string(ev.Rune())
		}
	}
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

// renderPDFPage renders page of the PDF at path, at size, to a PNG named by
// base, with pdftoppm, and returns the PNG's path.
func renderPDFPage(path string, page int, size pdfSize, base string) (string, error) {
	pdftoppm, err := exec.LookPath("pdftoppm")
	if err != nil {
		return "", fmt.Errorf("the PDF view needs pdftoppm, from poppler, which is not on the PATH")
	}
	n := strconv.Itoa(page)
	args := []string{"-png", "-singlefile", "-f", n, "-l", n}
	if size.fitWidth > 0 {
		args = append(args, "-scale-to-x", strconv.Itoa(size.fitWidth), "-scale-to-y", "-1")
	} else {
		args = append(args, "-r", strconv.Itoa(size.dpi))
	}
	out, err := exec.Command(pdftoppm, append(args, path, base)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pdftoppm: %s", vcFirstLine(string(out), err))
	}
	return base + ".png", nil
}

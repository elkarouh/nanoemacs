package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A PDF chosen in yazi is shown as its first page, which pdftoppm renders as a
// PNG and the image view draws. It goes the way a picture does, and needs the
// same terminal; elsewhere it is refused as any binary file is.

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

// pdfPage renders the first page of the PDF at path as a PNG, with pdftoppm. It
// returns the PNG's path and a func that removes it.
func pdfPage(path string) (string, func(), error) {
	pdftoppm, err := exec.LookPath("pdftoppm")
	if err != nil {
		return "", nil, fmt.Errorf("the PDF preview needs pdftoppm, which is not on the PATH")
	}
	dir, err := os.MkdirTemp("", "nem-pdf-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	base := filepath.Join(dir, "page")
	out, err := exec.Command(pdftoppm, "-png", "-singlefile", "-f", "1", "-l", "1",
		"-r", "150", path, base).CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("pdftoppm: %s", vcFirstLine(string(out), err))
	}
	return base + ".png", cleanup, nil
}

// showPicture shows a picture, or the first page of a PDF, over the window.
func (e *Editor) showPicture(path string) error {
	if !isPDFFile(path) {
		return e.showImage(path)
	}
	png, cleanup, err := pdfPage(path)
	if err != nil {
		return err
	}
	defer cleanup()
	return e.showImage(png)
}

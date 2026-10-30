package editor

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Borderliner/nem/text"
)

// An EPUB is opened as its HTML: pandoc converts the book to HTML, and the
// HTML preview lays it out with lynx, as it does for an .html file. The HTML
// is kept in a read-only buffer with no file behind it, so nothing is written
// to disk and saving cannot overwrite the book.

// epubTimeout bounds one pandoc run. A whole book takes longer to convert than
// one Markdown file does to render.
const epubTimeout = 60 * time.Second

// isEPUBFile reports whether path names an EPUB.
func isEPUBFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".epub")
}

// epubHTML converts the EPUB at path to HTML with pandoc.
func epubHTML(path string) ([]byte, error) {
	pandoc, err := exec.LookPath("pandoc")
	if err != nil {
		return nil, fmt.Errorf("an EPUB needs pandoc, which is not on the PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), epubTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, pandoc, "--from=epub", "--to=html", path).Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			msg = vcFirstLine(string(ee.Stderr), err)
		}
		return nil, fmt.Errorf("pandoc: %s", msg)
	}
	return out, nil
}

// openEPUB opens the EPUB at path as the HTML preview of its converted text, and
// returns the buffer to show. Without lynx the HTML source is shown instead,
// with the reason on the echo line.
func (e *Editor) openEPUB(path string) (*text.Buffer, error) {
	html, err := epubHTML(path)
	if err != nil {
		return nil, err
	}
	src := text.FromBytes(html)
	src.SetReadOnly(true)
	e.adopt(src, e.uniqueName(filepath.Base(path)))
	st, err := e.renderWith(src, htmlRender)
	if err != nil {
		e.Echo("%v", err)
		return src, nil
	}
	e.showPreview(st)
	return st.buf, nil
}

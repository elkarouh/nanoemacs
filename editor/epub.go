package editor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
)

// An EPUB is opened as its HTML: pandoc converts the book to HTML, and the
// HTML preview lays it out with lynx, as it does for an .html file. The HTML
// is kept in a read-only buffer with no file behind it, so nothing is written
// to disk and saving cannot overwrite the book.
//
// Pictures are drawn inline, by the terminal, over rows the preview keeps blank
// for them; epubimage.go draws them. lynx prints no pictures, so each picture
// tag becomes a marker line here, which expandPictures turns into those rows.

// epubTimeout bounds one pandoc run. A whole book takes longer to convert than
// one Markdown file does to render.
const epubTimeout = 60 * time.Second

// A cell is taken to be twice as tall as it is wide, as terminal fonts are. A
// picture's rows are reserved before the terminal is asked anything, so this
// fixes the shape it is drawn in; the terminal then fits it to that box.
const (
	epubCellW   = 10 // pixels
	epubCellH   = 20
	epubMaxRows = 60
)

var (
	epubImgTag = regexp.MustCompile(`<img\b[^>]*>`)
	epubImgSrc = regexp.MustCompile(`\bsrc="([^"]*)"`)
	epubMarker = regexp.MustCompile(`NEMPICTURE(\d+)`)
)

// epubPicture is one picture of a book: where its rows are in the preview, its
// size in cells, and its data, as PNG, for the terminal. A picture nem cannot
// decode has no rows and is not drawn.
type epubPicture struct {
	Line, Cols, Rows int
	png              []byte
	id               int  // the terminal's name for the stored image
	sent             bool // the terminal has the data stored under id
}

// epubPictureIDs is the last terminal image id given out. nem is single
// threaded, so no lock is needed.
var epubPictureIDs int

// isEPUBFile reports whether path names an EPUB.
func isEPUBFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".epub")
}

// epubHTML converts the EPUB at path to HTML with pandoc, extracting its
// pictures into mediaDir. Each picture tag becomes a marker line, and the
// files of the pictures are returned in marker order.
func epubHTML(path, mediaDir string) ([]byte, []string, error) {
	pandoc, err := exec.LookPath("pandoc")
	if err != nil {
		return nil, nil, fmt.Errorf("an EPUB needs pandoc, which is not on the PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), epubTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, pandoc, "--from=epub", "--to=html",
		"--extract-media="+mediaDir, path).Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			msg = vcFirstLine(string(ee.Stderr), err)
		}
		return nil, nil, fmt.Errorf("pandoc: %s", msg)
	}
	var files []string
	html := epubImgTag.ReplaceAllFunc(out, func(tag []byte) []byte {
		m := epubImgSrc.FindSubmatch(tag)
		if m == nil {
			return nil
		}
		file := string(m[1])
		if !filepath.IsAbs(file) {
			file = filepath.Join(mediaDir, file)
		}
		n := len(files)
		files = append(files, file)
		return []byte(fmt.Sprintf("<p>NEMPICTURE%d</p>", n))
	})
	return html, files, nil
}

// epubPictures reads the pictures in files, in order. A file nem cannot show,
// such as an SVG, gives a picture with no rows.
func epubPictures(files []string) []*epubPicture {
	pictures := make([]*epubPicture, len(files))
	for i, file := range files {
		p := &epubPicture{}
		pictures[i] = p
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		png, w, h, err := ui.ImagePNG(data)
		if err != nil {
			continue
		}
		p.Cols, p.Rows = epubBox(w, h)
		p.png = png
		epubPictureIDs++
		p.id = epubPictureIDs
	}
	return pictures
}

// epubBox is the box, in cells, that a picture of w by h pixels is drawn in:
// as wide as a preview line, or as its picture is when that is narrower, and as
// tall as its shape needs.
func epubBox(w, h int) (cols, rows int) {
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	cols = min(previewWidth, max(1, w/epubCellW))
	px := h * cols * epubCellW / w
	rows = min(epubMaxRows, max(1, (px+epubCellH-1)/epubCellH))
	return cols, rows
}

// expandPictures replaces each marker line of dumped, lynx's text of the book,
// with the blank rows its picture takes, and sets each picture's first line.
// A marker for a picture that has no rows is dropped with its line.
func expandPictures(dumped []byte, pictures []*epubPicture) []byte {
	var lines []string
	for _, l := range strings.Split(string(dumped), "\n") {
		m := epubMarker.FindStringSubmatch(l)
		n := -1
		if m != nil {
			n, _ = strconv.Atoi(m[1])
		}
		if n < 0 || n >= len(pictures) {
			lines = append(lines, l)
			continue
		}
		p := pictures[n]
		p.Line = len(lines)
		for range p.Rows {
			lines = append(lines, "")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// epubRenderer lays a book's HTML out with lynx, and keeps the rows of its
// pictures blank.
func epubRenderer(pictures []*epubPicture) renderer {
	return func(src string) ([]byte, error) {
		dumped, err := htmlRender(src)
		if err != nil {
			return nil, err
		}
		return expandPictures(dumped, pictures), nil
	}
}

// openEPUB opens the EPUB at path as the HTML preview of its text, with its
// pictures, and returns the buffer to show. Without lynx the HTML source is
// shown instead, with the reason on the echo line.
func (e *Editor) openEPUB(path string) (*text.Buffer, error) {
	mediaDir, err := os.MkdirTemp("", "nem-epub-")
	if err != nil {
		return nil, err
	}
	// The pictures are read into memory below, so their files can go at once.
	defer os.RemoveAll(mediaDir)
	html, files, err := epubHTML(path, mediaDir)
	if err != nil {
		return nil, err
	}
	pictures := epubPictures(files)
	src := text.FromBytes(html)
	src.SetReadOnly(true)
	e.adopt(src, e.uniqueName(filepath.Base(path)))
	st, err := e.renderWith(src, epubRenderer(pictures))
	if err != nil {
		e.Echo("%v", err)
		return src, nil
	}
	st.pictures = pictures
	e.showPreview(st)
	return st.buf, nil
}

package ui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"  // registered so ImagePNG can read them
	_ "image/jpeg" // registered so ImagePNG can read them
	"image/png"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// The kitty graphics protocol draws raw pixels over the cell grid. nem uses it
// for one thing: showing an image file in the editor. See the image-view
// command.
//
// tcell knows nothing of it, so an image is written straight to the terminal
// between frames and removed again before the next one; tcell never learns it
// was there.

// KittySupported guesses whether the terminal draws kitty graphics. Terminals
// that do announce themselves in the environment. A multiplexer is ruled out
// even over one that does: tmux and screen do not forward the protocol
// without configuration nem cannot see.
//
// NEM_IMAGES=1 says it does whatever the environment shows, as over ssh where
// the terminal's variables are not passed on, and NEM_IMAGES=0 says it does not.
func KittySupported() bool {
	switch os.Getenv("NEM_IMAGES") {
	case "1":
		return true
	case "0":
		return false
	}
	if os.Getenv("TMUX") != "" || os.Getenv("STY") != "" {
		return false
	}
	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "":
		return true
	case strings.Contains(os.Getenv("TERM"), "kitty"), strings.Contains(os.Getenv("TERM"), "ghostty"):
		return true
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "ghostty", "WezTerm":
		return true
	}
	return false
}

// ImagePNG decodes a PNG, JPEG or GIF and returns it as PNG with its size in
// pixels. PNG is the one format the protocol takes without further word from
// the sender, so the others are converted.
func ImagePNG(data []byte) (out []byte, w, h int, err error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("not an image nem can show: %w", err)
	}
	b := img.Bounds()
	if format == "png" {
		return data, b.Dx(), b.Dy(), nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

// FitImage returns the cells an imgW by imgH pixel image occupies when scaled
// to fit within maxCols by maxRows without changing its shape. cellW and cellH
// are a cell's size in pixels; zero means the terminal did not say, and a cell
// is then taken as twice as tall as wide. A small image is not enlarged past
// its own size.
func FitImage(imgW, imgH, maxCols, maxRows, cellW, cellH int) (cols, rows int) {
	if imgW <= 0 || imgH <= 0 || maxCols <= 0 || maxRows <= 0 {
		return 0, 0
	}
	if cellW <= 0 || cellH <= 0 {
		cellW, cellH = 1, 2
	}
	// Work in pixels: the scale that fits both directions.
	scale := float64(maxCols*cellW) / float64(imgW)
	if s := float64(maxRows*cellH) / float64(imgH); s < scale {
		scale = s
	}
	if scale > 1 && cellW > 1 {
		scale = 1
	}
	cols = int(float64(imgW)*scale/float64(cellW) + 0.5)
	rows = int(float64(imgH)*scale/float64(cellH) + 0.5)
	cols = min(max(cols, 1), maxCols)
	rows = min(max(rows, 1), maxRows)
	return cols, rows
}

// KittyDraw returns the escape sequence that shows png in a cols by rows cell
// box whose top-left cell is x, y. The cursor is saved and restored around it.
// The image data is sent in the escape stream rather than by file name, so it
// works over ssh.
func KittyDraw(png []byte, x, y, cols, rows int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\x1b7\x1b[%d;%dH", y+1, x+1)
	enc := base64.StdEncoding.EncodeToString(png)
	const chunk = 4096
	for first := true; len(enc) > 0; first = false {
		n := min(chunk, len(enc))
		more := 0
		if n < len(enc) {
			more = 1
		}
		if first {
			fmt.Fprintf(&sb, "\x1b_Ga=T,f=100,c=%d,r=%d,q=2,C=1,m=%d;", cols, rows, more)
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;", more)
		}
		sb.WriteString(enc[:n])
		sb.WriteString("\x1b\\")
		enc = enc[n:]
	}
	sb.WriteString("\x1b8")
	return sb.String()
}

// KittyClear is the sequence that removes every image nem drew.
const KittyClear = "\x1b_Ga=d,d=A,q=2\x1b\\"

// KittyClearPlacements removes what is drawn on screen, and keeps the images
// stored, so KittyPlace can draw them again without sending their data.
const KittyClearPlacements = "\x1b_Ga=d,d=a,q=2\x1b\\"

// KittyTransmit stores the PNG under id in the terminal, without drawing it.
// KittyPlace draws it, any number of times.
func KittyTransmit(png []byte, id int) string {
	var sb strings.Builder
	enc := base64.StdEncoding.EncodeToString(png)
	const chunk = 4096
	for first := true; len(enc) > 0; first = false {
		n := min(chunk, len(enc))
		more := 0
		if n < len(enc) {
			more = 1
		}
		if first {
			fmt.Fprintf(&sb, "\x1b_Ga=t,f=100,i=%d,q=2,m=%d;", id, more)
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;", more)
		}
		sb.WriteString(enc[:n])
		sb.WriteString("\x1b\\")
		enc = enc[n:]
	}
	return sb.String()
}

// KittyPlace returns the escape sequence that draws the image stored under id
// in a cols by rows cell box whose top-left cell is x, y. The cursor is saved
// and restored around it.
func KittyPlace(id, x, y, cols, rows int) string {
	return fmt.Sprintf("\x1b7\x1b[%d;%dH\x1b_Ga=p,i=%d,c=%d,r=%d,q=2,C=1\x1b\\\x1b8", y+1, x+1, id, cols, rows)
}

// WriteTTY writes raw bytes to the terminal, past tcell. It reports whether
// the terminal could be reached.
func WriteTTY(scr tcell.Screen, seq string) bool {
	tty, ok := scr.Tty()
	if !ok || tty == nil {
		return false
	}
	_, err := tty.Write([]byte(seq))
	return err == nil
}

// CellPixels is the size of one cell in pixels, or 0, 0 when the terminal does
// not report it.
func CellPixels(scr tcell.Screen) (w, h int) {
	tty, ok := scr.Tty()
	if !ok || tty == nil {
		return 0, 0
	}
	ws, err := tty.WindowSize()
	if err != nil {
		return 0, 0
	}
	return ws.CellDimensions()
}

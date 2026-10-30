package ui

import (
	"bytes"
	"fmt"
	"image"
	stddraw "image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"
)

// Sixel is the older raster graphics protocol, which Windows Terminal draws
// where it has no kitty graphics. It is used for the same one thing as kitty
// graphics: showing a picture over the cell grid. A sixel image cannot be
// deleted; it goes when the cells under it are painted again.

// SixelSupported guesses whether the terminal draws sixel. Windows Terminal
// announces itself with WT_SESSION, which WSL passes on. A multiplexer is ruled
// out as for kitty.
//
// NEM_IMAGES=sixel says it does whatever the environment shows.
func SixelSupported() bool {
	switch os.Getenv("NEM_IMAGES") {
	case "sixel":
		return true
	case "0", "1":
		return false
	}
	if os.Getenv("TMUX") != "" || os.Getenv("STY") != "" {
		return false
	}
	return os.Getenv("WT_SESSION") != ""
}

// ImagesSupported reports whether the terminal draws pictures in either
// protocol.
func ImagesSupported() bool { return KittySupported() || SixelSupported() }

// UseSixel reports whether pictures go out as sixel: the terminal has no kitty
// graphics but has sixel.
func UseSixel() bool { return !KittySupported() && SixelSupported() }

// sixelCellW and sixelCellH are a cell's size in pixels when the terminal does not report it,
// as Windows Terminal does not.
const sixelCellW, sixelCellH = 10, 20

// SixelCell is the size of a cell in pixels for drawing sixel: the reported
// one, or a guess.
func SixelCell(cw, ch int) (int, int) {
	if cw <= 0 || ch <= 0 {
		return sixelCellW, sixelCellH
	}
	return cw, ch
}

// The palette is the 216 colors of a 6x6x6 cube, then 24 grays, so that the
// anti-aliased gray of text does not band.
const (
	sixelCube   = 216
	sixelGrays  = 24
	sixelColors = sixelCube + sixelGrays
)

func sixelLevel(v uint8) int { return (int(v)*5 + 127) / 255 }

// sixelIndex is the palette entry nearest to a color.
func sixelIndex(r, g, b uint8) int {
	mx, mn := max(r, g, b), min(r, g, b)
	if mx-mn < 12 {
		avg := (int(r) + int(g) + int(b)) / 3
		return sixelCube + (avg*(sixelGrays-1)+127)/255
	}
	return sixelLevel(r)*36 + sixelLevel(g)*6 + sixelLevel(b)
}

// sixelPalette is a palette entry's channels, in the percent sixel takes.
func sixelPalette(i int) (r, g, b int) {
	if i >= sixelCube {
		v := (i - sixelCube) * 100 / (sixelGrays - 1)
		return v, v, v
	}
	return (i / 36) * 20, (i / 6 % 6) * 20, (i % 6) * 20
}

// scaleBox scales img to w by h pixels, averaging the source pixels under each
// new one, over white where it is transparent.
func scaleBox(img image.Image, w, h int) *image.RGBA {
	b := img.Bounds()
	src := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	stddraw.Draw(src, src.Bounds(), image.White, image.Point{}, stddraw.Src)
	stddraw.Draw(src, src.Bounds(), img, b.Min, stddraw.Over)
	sw, sh := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := y*sh/h, max((y+1)*sh/h, y*sh/h+1)
		for x := 0; x < w; x++ {
			x0, x1 := x*sw/w, max((x+1)*sw/w, x*sw/w+1)
			var r, g, bl, n int
			for sy := y0; sy < y1; sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := x0; sx < x1; sx++ {
					p := row[sx*4:]
					r += int(p[0])
					g += int(p[1])
					bl += int(p[2])
					n++
				}
			}
			o := dst.Pix[y*dst.Stride+x*4:]
			o[0], o[1], o[2], o[3] = uint8(r/n), uint8(g/n), uint8(bl/n), 255
		}
	}
	return dst
}

// SixelDraw returns the escape sequence that shows the PNG in a cols by rows
// cell box whose top-left cell is x, y, with cells cw by ch pixels. The cursor
// is saved and restored around it.
func SixelDraw(data []byte, x, y, cols, rows, cw, ch int) (string, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	w, h := cols*cw, rows*ch
	px := scaleBox(img, w, h)

	var sb strings.Builder
	fmt.Fprintf(&sb, "\x1b7\x1b[%d;%dH\x1bPq\"1;1;%d;%d", y+1, x+1, w, h)
	for i := 0; i < sixelColors; i++ {
		r, g, b := sixelPalette(i)
		fmt.Fprintf(&sb, "#%d;2;%d;%d;%d", i, r, g, b)
	}
	masks := make([][]byte, sixelColors)
	for i := range masks {
		masks[i] = make([]byte, w)
	}
	for y0 := 0; y0 < h; y0 += 6 {
		var used [sixelColors]bool
		for i := range masks {
			clear(masks[i])
		}
		for dy := 0; dy < 6 && y0+dy < h; dy++ {
			row := px.Pix[(y0+dy)*px.Stride:]
			for xx := 0; xx < w; xx++ {
				c := sixelIndex(row[xx*4], row[xx*4+1], row[xx*4+2])
				masks[c][xx] |= 1 << dy
				used[c] = true
			}
		}
		first := true
		for c := 0; c < sixelColors; c++ {
			if !used[c] {
				continue
			}
			if !first {
				sb.WriteByte('$')
			}
			first = false
			sb.WriteByte('#')
			sb.WriteString(strconv.Itoa(c))
			sixelRuns(&sb, masks[c])
		}
		sb.WriteByte('-')
	}
	sb.WriteString("\x1b\\\x1b8")
	return sb.String(), nil
}

// sixelRuns writes a row of sixel masks, run-length encoded.
func sixelRuns(sb *strings.Builder, row []byte) {
	for i := 0; i < len(row); {
		j := i
		for j < len(row) && row[j] == row[i] {
			j++
		}
		ch := 63 + row[i]
		if n := j - i; n > 3 {
			sb.WriteByte('!')
			sb.WriteString(strconv.Itoa(n))
			sb.WriteByte(ch)
		} else {
			for k := 0; k < n; k++ {
				sb.WriteByte(ch)
			}
		}
		i = j
	}
}

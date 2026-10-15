// Package ansistyle reads the text a program prints with terminal styling, such
// as mdcat's output with --ansi, and returns the plain text together with the
// styles over it. The text is what a buffer holds; the styles are what the
// screen should show.
//
// It understands the sequences that matter for styled text: SGR (ESC [ ... m)
// for colours, bold, italic and underline; and OSC 8 hyperlinks, whose link
// target is dropped and whose text is kept. Any other escape sequence is
// removed without effect.
package ansistyle

import "strconv"

// Color is a terminal colour: the default, a palette index, or a truecolour.
type Color uint32

const (
	// Default is the colour the terminal would use anyway.
	Default Color = 0
	rgbFlag Color = 1 << 24
)

// Palette is the colour at index i of the 256-colour palette, 0 to 255.
func Palette(i int) Color { return Color(i) + 1 }

// RGB is a truecolour.
func RGB(r, g, b uint8) Color { return rgbFlag | Color(r)<<16 | Color(g)<<8 | Color(b) }

// IsRGB reports whether c is a truecolour, and its components if so.
func (c Color) IsRGB() (r, g, b uint8, ok bool) {
	if c&rgbFlag == 0 {
		return 0, 0, 0, false
	}
	return uint8(c >> 16), uint8(c >> 8), uint8(c), true
}

// PaletteIndex reports c's index in the 256-colour palette, if it is one.
func (c Color) PaletteIndex() (int, bool) {
	if c == Default || c&rgbFlag != 0 {
		return 0, false
	}
	return int(c) - 1, true
}

// Style is the styling over a run of text. The zero Style is the default.
type Style struct {
	FG, BG    Color
	Bold      bool
	Italic    bool
	Underline bool
}

// Span is a run of styled text, by rune offset into the plain text: the runes
// from Start up to End, not including End.
type Span struct {
	Start, End int
	Style      Style
}

// Parse returns the plain text of s and the non-default styles over it. Spans
// are in order, do not overlap, and are merged where the style is the same.
func Parse(s string) (string, []Span) {
	var (
		text  []rune
		spans []Span
		cur   Style
		start int
	)
	emit := func() {
		if cur == (Style{}) || start == len(text) {
			return
		}
		if n := len(spans); n > 0 && spans[n-1].End == start && spans[n-1].Style == cur {
			spans[n-1].End = len(text)
			return
		}
		spans = append(spans, Span{Start: start, End: len(text), Style: cur})
	}
	setStyle := func(next Style) {
		if next == cur {
			return
		}
		emit()
		cur = next
		start = len(text)
	}

	rs := []rune(s)
	for i := 0; i < len(rs); {
		r := rs[i]
		if r != 0x1b {
			if r != '\r' {
				text = append(text, r)
			}
			i++
			continue
		}
		if i+1 >= len(rs) {
			break // a lone ESC at the very end is dropped
		}
		switch rs[i+1] {
		case '[': // CSI: parameters, then a final byte from @ to ~
			j := i + 2
			for j < len(rs) && (rs[j] < 0x40 || rs[j] > 0x7e) {
				j++
			}
			if j >= len(rs) {
				return finish(text, spans, cur, start, emit)
			}
			if rs[j] == 'm' {
				setStyle(applySGR(cur, string(rs[i+2:j])))
			}
			i = j + 1
		case ']': // OSC: ends at BEL, or at ESC \
			j := i + 2
			for j < len(rs) && rs[j] != 0x07 && !(rs[j] == 0x1b && j+1 < len(rs) && rs[j+1] == '\\') {
				j++
			}
			if j >= len(rs) {
				return finish(text, spans, cur, start, emit)
			}
			if rs[j] == 0x07 {
				i = j + 1
			} else {
				i = j + 2
			}
		default:
			// Any other escape: intermediate bytes (0x20 to 0x2f), then one
			// final byte. ESC ( B is three bytes; ESC 7 is two.
			j := i + 1
			for j < len(rs) && rs[j] >= 0x20 && rs[j] <= 0x2f {
				j++
			}
			i = j + 1
		}
	}
	emit()
	return string(text), spans
}

// finish ends parsing early on a truncated sequence, keeping what came before.
func finish(text []rune, spans []Span, cur Style, start int, emit func()) (string, []Span) {
	emit()
	return string(text), spans
}

// applySGR returns the style after the SGR parameters p, which are the digits
// and semicolons between ESC [ and m.
func applySGR(cur Style, p string) Style {
	params := splitParams(p)
	if len(params) == 0 {
		return Style{} // ESC[m is a reset
	}
	for i := 0; i < len(params); i++ {
		n := params[i]
		switch {
		case n == 0:
			cur = Style{}
		case n == 1:
			cur.Bold = true
		case n == 3:
			cur.Italic = true
		case n == 4:
			cur.Underline = true
		case n == 22:
			cur.Bold = false
		case n == 23:
			cur.Italic = false
		case n == 24:
			cur.Underline = false
		case n >= 30 && n <= 37:
			cur.FG = Palette(n - 30)
		case n == 39:
			cur.FG = Default
		case n >= 40 && n <= 47:
			cur.BG = Palette(n - 40)
		case n == 49:
			cur.BG = Default
		case n >= 90 && n <= 97:
			cur.FG = Palette(n - 90 + 8)
		case n >= 100 && n <= 107:
			cur.BG = Palette(n - 100 + 8)
		case n == 38 || n == 48:
			c, used := extendedColor(params[i+1:])
			i += used
			if n == 38 {
				cur.FG = c
			} else {
				cur.BG = c
			}
		}
	}
	return cur
}

// extendedColor reads the colour after a 38 or 48: 5;n for a palette index, or
// 2;r;g;b for a truecolour. It returns the colour and how many parameters it
// used; an incomplete one is Default.
func extendedColor(p []int) (Color, int) {
	if len(p) >= 2 && p[0] == 5 {
		return Palette(clamp8(p[1])), 2
	}
	if len(p) >= 4 && p[0] == 2 {
		return RGB(uint8(clamp8(p[1])), uint8(clamp8(p[2])), uint8(clamp8(p[3]))), 4
	}
	return Default, len(p)
}

func clamp8(n int) int {
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return n
}

// splitParams reads "1;94;104" as numbers. An empty field is a zero, as the
// standard has it.
func splitParams(p string) []int {
	var out []int
	field := ""
	for _, r := range p + ";" {
		if r == ';' {
			n, err := strconv.Atoi(field)
			if err != nil {
				n = 0
			}
			out = append(out, n)
			field = ""
			continue
		}
		field += string(r)
	}
	return out
}

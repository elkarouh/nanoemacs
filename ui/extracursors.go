package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/view"
)

// Extra cursors are drawn by reversing their cells: the hardware cursor can
// only be in one place, so the others are marked on the screen. A selection
// made by an extra cursor is reversed too, when it sits on one line.
//
// The cell arithmetic is the same as for point's cursor, including line
// wrapping, which is why the rows come from screenRows. Lines that run
// right-to-left are skipped: their columns do not map to cells this simply,
// and a mark in the wrong place is worse than none.
func paintExtraCursors(scr tcell.Screen, rect view.Rect, gw, textH, textW int, f Frame, th Theme) {
	w := f.Active
	if w == nil || w.Buf == nil || len(w.Extra) == 0 {
		return
	}
	rows := screenRows(w, textH, textW, th.Wrap)
	wrap := th.Wrap
	for _, c := range w.Extra {
		lo, hi := text.OrderPos(c.Pt, c.Mark)
		if lo.Line == hi.Line {
			for col := lo.Col; col < hi.Col; col++ {
				reverseCell(scr, rect, gw, textW, textH, w, rows, wrap, th, text.Pos{Line: lo.Line, Col: col})
			}
		}
		reverseCell(scr, rect, gw, textW, textH, w, rows, wrap, th, c.Pt)
	}
}

// reverseCell reverses the screen cell that holds the character at p, when
// that cell is on screen.
func reverseCell(scr tcell.Screen, rect view.Rect, gw, textW, textH int, w *view.Window, rows []screenRow, wrap bool, th Theme, p text.Pos) {
	b := w.Buf
	if p.Line >= b.NumLines() {
		return
	}
	l := b.Line(p.Line)
	if th.lineUsesBidi(l) {
		return
	}
	sx, sy, ok := cellOf(w, rows, wrap, l, b.ClampPos(p))
	if !ok || sx < 0 || sx >= textW || sy < 0 || sy >= textH {
		return
	}
	x, y := rect.X+gw+sx, rect.Y+sy
	main, comb, style, _ := scr.GetContent(x, y)
	scr.SetContent(x, y, main, comb, style.Reverse(true))
}

// cellOf returns the text-area column and row that draw the character at p.
// Unwrapped, that is p's line and column less the scroll; wrapped, it is the
// row whose span holds p, found the way placeCursor finds point's row.
func cellOf(w *view.Window, rows []screenRow, wrap bool, l *text.Line, p text.Pos) (sx, sy int, ok bool) {
	if !wrap {
		return int(l.DisplayCol(p.Col) - w.LeftCol), p.Line - w.Top, true
	}
	for i, r := range rows {
		if r.line != p.Line || p.Col < r.from || (p.Col >= r.to && !r.last) {
			continue
		}
		return int(l.DisplayCol(p.Col) - r.col), i, true
	}
	return 0, 0, false
}

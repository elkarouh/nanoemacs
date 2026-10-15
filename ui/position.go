package ui

import (
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/view"
)

// Cell is what a screen cell lies on, for the mouse: the window whose pane it
// is in, and the text position drawn under it.
//
// InText is false on the window's modeline, where there is no text. TextH and
// TextW are the window's text area, in rows and columns, so a caller scrolling
// the window knows how much of it is shown.
type Cell struct {
	Win          *view.Window
	Pos          text.Pos
	InText       bool
	TextH, TextW int
}

// CellAt maps a screen cell to the window it falls in and the text position
// drawn there, so a mouse click, drag or scroll can act on what the eye sees
// under it. It uses the renderer's own layout, gutter and rows.
//
// The gutter is not text. A cell in it maps to the start of its line, so
// dragging from the numbers selects from the line's beginning and never
// includes the numbers themselves. The position is clamped into the window's
// text, so a drag past the window's edge or below the last line still gives a
// usable position.
//
// ok is false when the cell is in no window at all: the echo area, or a gap in
// the layout.
//
// A line drawn right to left is mapped by display column, which is not what the
// screen shows for it, so a click on such a line is approximate.
func CellAt(f Frame, th Theme, w, h, x, y int) (Cell, bool) {
	if f.Tree == nil || w <= 0 || h <= 0 {
		return Cell{}, false
	}
	echoY, _ := echoRow(f, h)
	if echoY <= 0 || y < 0 || y >= echoY {
		return Cell{}, false
	}
	for win, rect := range treeRects(f, w, echoY) {
		if x < rect.X || x >= rect.X+rect.W || y < rect.Y || y >= rect.Y+rect.H {
			continue
		}
		if win.Buf == nil {
			return Cell{}, false
		}
		textH := view.TextHeight(rect)
		inText := y-rect.Y < textH
		if textH <= 0 {
			return Cell{Win: win, Pos: win.Buf.ClampPos(win.Pt), InText: false}, true
		}
		gw := gutterFor(rect, win, f.isListing(win.Buf), th)
		textW := rect.W - gw
		c := Cell{Win: win, InText: inText, TextH: textH, TextW: textW}

		// A cell below the text is the last row, and one in the gutter or left
		// of the text is column zero.
		row := min(max(y-rect.Y, 0), textH-1)
		cx := max(x-rect.X-gw, 0)

		rows := screenRows(win, textH, textW, th.Wrap)
		if len(rows) == 0 {
			c.Pos = win.Buf.End()
			return c, true
		}
		if row >= len(rows) {
			row = len(rows) - 1
		}
		r := rows[row]
		l := win.Buf.Line(r.line)
		var pos text.Pos
		if th.Wrap {
			// A wrapped row starts at its own display column, and the rune
			// chosen must lie within the row's runes.
			at := l.RuneAt(r.col + text.ColIdx(cx))
			pos = text.Pos{Line: r.line, Col: min(max(at, r.from), r.to)}
		} else {
			at := l.RuneAt(win.LeftCol + text.ColIdx(cx))
			pos = text.Pos{Line: r.line, Col: at}
		}
		c.Pos = win.Buf.ClampPos(pos)
		return c, true
	}
	return Cell{}, false
}

// echoRow is the row the echo area is pinned to, and the candidate rows a
// prompt lists beneath it, as Render lays them out. Render and CellAt both use
// it, so a click is mapped against the layout that was drawn.
func echoRow(f Frame, h int) (echoY int, miniRows []PanelLine) {
	miniRows = f.MiniRows
	if room := h - 1 - minTreeRows; len(miniRows) > room {
		miniRows = miniRows[:max(room, 0)]
	}
	return h - 1 - len(miniRows), miniRows
}

package editor

// Mouse input.
//
// A terminal's own selection copies whatever cells are on screen, and the line
// numbers are cells, so a drag through them took the numbers with the text.
// nem therefore takes the mouse itself. A press sets point and a drag sets the
// region, and the gutter is never part of either: ui.CellAt maps a cell in it
// to the start of its line. The wheel scrolls the window under the pointer. Most
// terminals still give their own selection back while Shift is held.
//
// A prompt keeps the keyboard, and the mouse is ignored while one is open.

import (
	"slices"

	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
	"github.com/Borderliner/nem/view"
	"github.com/gdamore/tcell/v2"
)

// wheelRows is how many rows one notch of the wheel scrolls.
const wheelRows = 3

// mouseDrag is a press in progress. win is the window a selection drag began
// in, nil when none. tab is the buffer whose tab the button went down on, nil
// when none; tabMoved records that the drag reordered the tabs, so the release
// that follows is not also a click on one.
type mouseDrag struct {
	win      *view.Window
	tab      *text.Buffer
	tabMoved bool
}

// handleMouse reacts to a mouse event: the wheel scrolls; a press on a tab
// starts a possible reorder and a press in a window sets point and the mark; a
// drag moves a tab along the strip or moves point and so the region; a release
// ends the press, and shows the buffer of a tab that was clicked rather than
// dragged.
func (e *Editor) handleMouse(ev *tcell.EventMouse) {
	if e.mini != nil || e.scr == nil {
		return
	}
	w, h := e.scr.Size()
	x, y := ev.Position()
	btns := ev.Buttons()
	f := ui.Frame{Tree: e.tree, ListingOf: e.isListing, Tabs: e.tabItems()}

	if btns&(tcell.WheelUp|tcell.WheelDown) != 0 {
		cell, ok := ui.CellAt(f, e.th, w, h, x, y)
		if !ok {
			return
		}
		rows := wheelRows
		if btns&tcell.WheelUp != 0 {
			rows = -rows
		}
		e.scrollUnder(cell, rows)
		return
	}

	if btns&tcell.Button1 == 0 {
		// The button is up. A tab that was pressed and not dragged is shown.
		if e.mouse.tab != nil && !e.mouse.tabMoved {
			e.active.Visit(e.mouse.tab)
		}
		e.mouse = mouseDrag{}
		return
	}

	// A press on a tab, or a drag that began on one, reorders the strip: the
	// tab follows the pointer along the row, and the buffers are reordered to
	// match, as ttt's tabs are.
	if e.mouse.tab != nil {
		if i, onTab := ui.TabAt(f, w, x, y); onTab {
			e.moveTab(e.mouse.tab, i)
		}
		return
	}
	if i, onTab := ui.TabAt(f, w, x, y); onTab {
		if e.mouse.win == nil && i < len(e.buffers) {
			e.mouse.tab = e.buffers[i]
			e.mouse.tabMoved = false
		}
		return
	}

	cell, ok := ui.CellAt(f, e.th, w, h, x, y)
	if !ok {
		return
	}

	if e.mouse.win == nil {
		// A press on a modeline or in a gap is not a place to put point, so it
		// does nothing. One in a text area selects that window and starts a drag.
		if !cell.InText {
			return
		}
		win := cell.Win
		e.mouse.win = win
		// A click in the side panel picks what is under it, and starts no drag.
		if win == e.side && e.sideOpen() {
			e.sidebarClick(cell.Pos)
			return
		}
		e.active = win
		win.Pt = cell.Pos
		// The mark goes where the press was, but the region is not shown until
		// the pointer moves: a click alone selects nothing, as in emacs.
		win.Buf.SetMark(cell.Pos)
		win.Buf.DeactivateMark()
		return
	}

	// A drag that has left the window it began in is ignored. One within it
	// moves point and shows the region from the press to here.
	if cell.Win != e.mouse.win {
		return
	}
	win := cell.Win
	win.Pt = cell.Pos
	if cell.Pos != win.Buf.Mark() {
		win.Buf.ActivateMark()
	}
}

// moveTab puts buffer b at index to of the buffer order, shifting the others,
// and so of the tab strip. Buffers are kept in that order everywhere they are
// listed, so next-buffer and previous-buffer follow the strip as it is shown.
func (e *Editor) moveTab(b *text.Buffer, to int) {
	from := slices.Index(e.tabs, b)
	if from < 0 || to < 0 || to >= len(e.tabs) || from == to {
		return
	}
	e.tabs = slices.Insert(slices.Delete(e.tabs, from, from+1), to, b)
	e.mouse.tabMoved = true
}

// scrollUnder scrolls the window under the wheel by rows lines, negative for
// up. As in emacs, point stays where it is while it can, and only moves when
// scrolling leaves it outside the comfortable part of the window, so the render
// path's own scrolling to keep point in view does not undo the wheel.
func (e *Editor) scrollUnder(c ui.Cell, rows int) {
	if !c.InText || c.Win == nil || c.Win.Buf == nil || c.TextH <= 0 {
		return
	}
	w := c.Win
	b := w.Buf
	margin := min(max(e.th.ScrollMargin, 0), (c.TextH-1)/2)

	if e.th.Wrap {
		width := c.TextW
		old := view.RowPos{Line: w.Top, Row: w.TopRow}
		top, _ := view.StepRows(b, old, rows, width)
		if top == old {
			return
		}
		w.Top, w.TopRow = top.Line, top.Row

		lo, _ := view.StepRows(b, top, margin, width)
		hi, _ := view.StepRows(b, top, c.TextH-1-margin, width)
		cur := view.RowOf(b, w.Pt, width)
		var target view.RowPos
		switch {
		case cur.Before(lo):
			target = lo
		case hi.Before(cur):
			target = hi
		default:
			return
		}
		_, _, rowCol := view.RowSpan(b, cur, width)
		col := b.Line(w.Pt.Line).DisplayCol(w.Pt.Col) - rowCol
		w.Pt = view.PosInRow(b, target, width, col)
		return
	}

	// Unwrapped, a row is a line and the window's top is a line.
	n := b.NumLines()
	top := min(max(w.Top+rows, 0), max(n-c.TextH, 0))
	if top == w.Top {
		return
	}
	w.Top, w.TopRow = top, 0

	lo := min(top+margin, n-1)
	hi := min(top+c.TextH-1-margin, n-1)
	ln := w.Pt.Line
	switch {
	case ln < lo:
		ln = lo
	case ln > hi:
		ln = hi
	default:
		return
	}
	w.Pt = b.ClampPos(text.Pos{Line: ln, Col: w.Pt.Col})
}

// tabItems lists the buffers the tab strip shows, in the order they were
// opened, the one the current window shows marked active. With fewer than two
// buffers there is no strip, and nil is returned.
func (e *Editor) tabItems() []ui.TabItem {
	if len(e.tabs) < 2 {
		return nil
	}
	cur := e.active.Buf
	out := make([]ui.TabItem, 0, len(e.tabs))
	for _, b := range e.tabs {
		out = append(out, ui.TabItem{Name: e.BufferName(b), Active: b == cur, Modified: b.Modified()})
	}
	return out
}

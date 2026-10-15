package ui

import (
	"github.com/Borderliner/nem/view"
	"github.com/gdamore/tcell/v2"
)

// The tab strip is one row across the top of the frame, one tab per buffer, as
// ttt's is. It shows only when there is more than one buffer to choose between;
// with a single buffer the frame is exactly as it was. A tab's active one is
// the buffer the current window shows, and a tab with unsaved changes is
// marked with a star.

// TabItem is one buffer in the tab strip.
type TabItem struct {
	Name     string
	Active   bool
	Modified bool
}

// tabRows is how many rows the tab strip takes at the top of the frame.
func tabRows(f Frame) int {
	if len(f.Tabs) < 2 {
		return 0
	}
	return 1
}

// tabLabel is what one tab shows: its name, starred if it has unsaved changes.
func tabLabel(t TabItem) string {
	mark := ""
	if t.Modified {
		mark = " *"
	}
	return " " + t.Name + mark + " "
}

// tabSpans lays the tabs out across a row width cells wide. It returns the
// cells each tab shown takes, and the index of the first one shown: when the
// tabs do not all fit, the strip starts at the first tab that lets the active
// one show, so the tab the user is on is never off the row.
func tabSpans(tabs []TabItem, width int) (spans [][2]int, first int) {
	widths := make([]int, len(tabs))
	active := -1
	for i, t := range tabs {
		widths[i] = len([]rune(tabLabel(t)))
		if t.Active {
			active = i
		}
	}
	for first = 0; first < active; first++ {
		if sum(widths[first:active+1]) <= width {
			break
		}
	}
	x := 0
	for i := first; i < len(tabs); i++ {
		if x+widths[i] > width {
			break
		}
		spans = append(spans, [2]int{x, x + widths[i]})
		x += widths[i]
	}
	return spans, first
}

func sum(xs []int) int {
	n := 0
	for _, x := range xs {
		n += x
	}
	return n
}

// drawTabBar writes the tab strip into row y of the frame, the active tab
// reversed and the others quiet.
func drawTabBar(scr tcell.Screen, y, width int, tabs []TabItem, th Theme) {
	spans, first := tabSpans(tabs, width)
	for k, sp := range spans {
		t := tabs[first+k]
		style := th.LineNumber
		if t.Active {
			style = th.PanelSelected
		}
		runes := []rune(tabLabel(t))
		for x := sp[0]; x < sp[1]; x++ {
			scr.SetContent(x, y, runes[x-sp[0]], nil, style)
		}
	}
}

// TabAt reports which tab, by its index in f.Tabs, the cell at x, y is on. ok
// is false when there is no tab strip or the cell is not on it.
func TabAt(f Frame, w, x, y int) (index int, ok bool) {
	if tabRows(f) == 0 || y != 0 {
		return 0, false
	}
	spans, first := tabSpans(f.Tabs, w)
	for k, sp := range spans {
		if x >= sp[0] && x < sp[1] {
			return first + k, true
		}
	}
	return 0, false
}

// treeRects lays the windows out in the rows below the tab strip, within
// treeH rows of the frame. Render draws them and CellAt maps clicks against
// them, so both see the same layout.
func treeRects(f Frame, w, treeH int) map[*view.Window]view.Rect {
	top := tabRows(f)
	if treeH <= top {
		return map[*view.Window]view.Rect{}
	}
	rects := f.Tree.Layout(w, treeH-top)
	for win, r := range rects {
		r.Y += top
		rects[win] = r
	}
	return rects
}

// treeDividers is treeRects' counterpart for the dividers between windows.
func treeDividers(f Frame, w, treeH int) []view.Rect {
	top := tabRows(f)
	if treeH <= top {
		return nil
	}
	ds := f.Tree.Dividers(w, treeH-top)
	for i := range ds {
		ds[i].Y += top
	}
	return ds
}

package ui

import (
	"testing"

	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/view"
)

// An extra cursor's selected word is reversed on screen, and so is the cell
// where that cursor sits. Ordinary text next to it is left alone, and point
// keeps the hardware cursor.
func TestExtraCursorSelectionIsReversed(t *testing.T) {
	f, w := singleFrame(t, "foo bar foo")
	w.Pt = text.Pos{Line: 0, Col: 3}
	w.Extra = []view.Cursor{{Pt: text.Pos{Line: 0, Col: 11}, Mark: text.Pos{Line: 0, Col: 8}}}
	scr := drawPlain(t, 20, 6, f)

	for col := 8; col <= 10; col++ {
		if !reversed(cellAt(t, scr, col, 0)) {
			t.Errorf("cell %d of the extra selection is not reversed", col)
		}
	}
	if reversed(cellAt(t, scr, 4, 0)) {
		t.Error("ordinary text between cursors was reversed")
	}
	if x, y, vis := scr.GetCursor(); !vis || x != 3 || y != 0 {
		t.Errorf("hardware cursor at (%d,%d) visible=%v, want point at (3,0)", x, y, vis)
	}
}

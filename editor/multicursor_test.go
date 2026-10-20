package editor

import (
	"testing"

	"github.com/Borderliner/nem/text"
)

// C-d selects the word at point. The next C-d adds a cursor on the next
// occurrence; typing then changes every occurrence at once.
func TestCtrlDAddsCursorsAndTypingEditsThemAll(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo baz foo")
	press(t, e, "C-d", "C-d")
	if got := len(e.active.Extra); got != 1 {
		t.Fatalf("extra cursors = %d, want 1", got)
	}

	press(t, e, "X")

	wantText(t, e, "X bar X baz foo")
	if len(e.active.Extra) != 1 {
		t.Errorf("typing dropped the extra cursor: %d", len(e.active.Extra))
	}
}

// Every press adds the next occurrence, and occurrences already selected are
// not added twice.
func TestRepeatedCtrlDWalksThroughOccurrences(t *testing.T) {
	e, _ := newTestEditor(t, "a a a")
	press(t, e, "C-d", "C-d", "C-d")
	if got := len(e.active.Extra); got != 2 {
		t.Fatalf("extra cursors = %d, want 2", got)
	}
	press(t, e, "b")
	wantText(t, e, "b b b")
}

// A whole-word search does not match inside a longer word.
func TestCtrlDMatchesWholeWordsOnly(t *testing.T) {
	e, _ := newTestEditor(t, "cat concat cat")
	press(t, e, "C-d", "C-d")
	press(t, e, "Z")
	wantText(t, e, "Z concat Z")
}

// Backspace with a selection at each cursor removes the selections.
func TestBackspaceRemovesEverySelection(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo")
	press(t, e, "C-d", "C-d", "<backspace>")
	wantText(t, e, " bar ")
}

// Every cursor has a selection, so Backspace at one removes the whole of each
// selected word and leaves the space between them.
func TestBackspaceOnTwoWordsKeepsTheSpace(t *testing.T) {
	e, _ := newTestEditor(t, "abc abc")
	press(t, e, "C-d", "C-d", "<backspace>")
	wantText(t, e, " ")
}

// An edit made with several cursors undoes in one step.
func TestMultiCursorEditUndoesAsOneStep(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo")
	press(t, e, "C-d", "C-d", "Q")
	wantText(t, e, "Q bar Q")
	press(t, e, "C-/")
	wantText(t, e, "foo bar foo")
}

// Motion drops the extra cursors and leaves point where it was moved.
func TestOtherCommandsDropExtraCursors(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo")
	press(t, e, "C-d", "C-d", "C-g")
	if len(e.active.Extra) != 0 {
		t.Errorf("extra cursors survived motion: %d", len(e.active.Extra))
	}
}

// A line break typed at several cursors lands at each of them.
func TestTypingNewlineAtEveryCursor(t *testing.T) {
	e, _ := newTestEditor(t, "x", "x")
	press(t, e, "C-d", "C-d", "Y")
	wantText(t, e, "Y\nY")
}

// Typing after a cursor on a later line does not disturb earlier ones.
func TestEditsOnLaterLinesKeepEarlierCursorsRight(t *testing.T) {
	e, _ := newTestEditor(t, "one x two", "x three x")
	// Point to the first x, four columns in: past "one" and the space.
	press(t, e, "C-f", "C-f", "C-f", "C-f", "C-d", "C-d", "C-d", "W")
	wantText(t, e, "one W two\nW three W")
}

// Right with a selection collapses it to the selection's end, and does not
// move on: the cursors sit just after each selected word, and typing inserts
// there instead of replacing the word.
func TestRightArrowCollapsesEverySelection(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo baz foo")
	press(t, e, "C-d", "C-d", "<right>")
	if got := len(e.active.Extra); got != 1 {
		t.Fatalf("extra cursors after <right> = %d, want 1", got)
	}
	press(t, e, "X")
	wantText(t, e, "fooX bar fooX baz foo")
}

// Motion keeps each cursor's own goal column: next-line takes point and the
// extra cursor down a line each, and each keeps the column it started from.
func TestNextLineKeepsEachCursorsColumn(t *testing.T) {
	e, _ := newTestEditor(t, "ab ab", "xxxxxxxx", "ab ab")
	press(t, e, "C-d", "C-d", "C-n")
	if len(e.active.Extra) != 1 {
		t.Fatalf("extra cursors = %d, want 1", len(e.active.Extra))
	}
	wantPt(t, e, 1, 2) // point: its selection ended at column 2
	if got := e.active.Extra[0].Pt; got != (text.Pos{Line: 1, Col: 5}) {
		t.Errorf("extra cursor at %v, want line 1 column 5", got)
	}
}

// Two cursors that a motion brings to the same spot collapse to one. C-a takes
// each cursor to the start of its line, so both end up at column 0.
func TestMotionMergesCursorsThatMeet(t *testing.T) {
	e, _ := newTestEditor(t, "ab ab")
	press(t, e, "C-d", "C-d", "C-a")
	if got := len(e.active.Extra); got != 0 {
		t.Errorf("extra cursors after they met = %d, want 0", got)
	}
}

// After one Right the selections have collapsed. A second motion must leave
// bare cursors, not turn the distance it moved into a selection, or the next
// typed character would replace text instead of being inserted.
func TestTypingAfterTwoRightPressesInserts(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo baz foo")
	press(t, e, "C-d", "C-d", "<right>", "<right>", "X")
	wantText(t, e, "foo Xbar foo Xbaz foo")
}

// C-M-d drops the cursor added last. Its word goes back to plain text, and the
// earlier selections are kept.
func TestRemoveLastCursorGivesItsWordBack(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo baz foo")
	press(t, e, "C-d", "C-d", "C-d", "C-M-d")
	if got := len(e.active.Extra); got != 1 {
		t.Fatalf("extra cursors = %d, want 1", got)
	}
	press(t, e, "X")
	wantText(t, e, "X bar X baz foo")
}

// C-M-d with only point's selection left does nothing harmful.
func TestRemoveLastCursorWithNoneLeftIsSafe(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo")
	press(t, e, "C-d", "C-M-d")
	wantEcho(t, e, "Only one cursor")
	if got := e.Buf().String(); got != "foo bar foo" {
		t.Errorf("buffer changed: %q", got)
	}
}

// C-M-s drops the last cursor and moves on: the word it skipped is not picked
// again, so the third occurrence is selected instead of the second.
func TestSkipNextOccurrenceSkipsTheSecondMatch(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo baz foo")
	press(t, e, "C-d", "C-d", "C-M-s", "X")
	wantText(t, e, "X bar foo baz X")
}

// Skipping with no further occurrence restores the cursor it dropped.
func TestSkipWithNothingBeyondKeepsTheCursor(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo")
	press(t, e, "C-d", "C-d", "C-M-s")
	if got := len(e.active.Extra); got != 1 {
		t.Errorf("extra cursors = %d, want 1 (the skipped one restored)", got)
	}
}

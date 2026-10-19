package editor

import (
	"errors"
	"sort"
	"unicode"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/view"
)

// Multiple cursors, as Sublime Text and VS Code have them.
//
// C-d selects the word at point. Each further C-d adds a cursor, selecting the
// next occurrence of the selected text. Typing, Backspace and Delete then act
// at every cursor at once, and the whole change undoes as one step. Any other
// command - a motion, a quit - drops the extra cursors, so they never outlive
// the edit they were made for.
//
// Deliberately out of scope for this version: column selection, select-all-
// occurrences, mouse-placed cursors, and occurrences spanning more than one
// line.

// addNextOccurrence is C-d's command.
const addNextOccurrence = "add-next-occurrence"

// registerMultiCursorCommands adds add-next-occurrence.
func registerMultiCursorCommands(e *Editor, reg *command.Registry) error {
	return reg.Register(command.Command{
		Name:        addNextOccurrence,
		Doc:         "Select the word at point, then each further press adds a cursor at the next occurrence of the selection. Edits then apply at every cursor.",
		Interactive: true,
		Fn:          func(command.Env) error { return e.addNextOccurrence() },
	})
}

// hasExtraCursors reports whether the active window has cursors beyond point.
func (e *Editor) hasExtraCursors() bool {
	return e.active != nil && len(e.active.Extra) > 0
}

// clearExtraCursors drops the extra cursors, leaving point and its selection.
func (e *Editor) clearExtraCursors() {
	if e.active != nil {
		e.active.Extra = nil
	}
}

// primaryCursor is point with the buffer's mark as its selection's other end.
func primaryCursor(w *view.Window) view.Cursor {
	mark := w.Pt
	if w.Buf.MarkActive() {
		mark = w.Buf.Mark()
	}
	return view.Cursor{Pt: w.Pt, Mark: mark, Goal: w.GoalCol}
}

// allCursors lists every cursor, point's first.
func allCursors(w *view.Window) []view.Cursor {
	cs := make([]view.Cursor, 0, 1+len(w.Extra))
	cs = append(cs, primaryCursor(w))
	return append(cs, w.Extra...)
}

// isWordRune is what a word is made of, for picking the word at point.
func isWordRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// wordAt returns the bounds of the word point is in or at the end of, or an
// empty range when there is no word there.
func wordAt(b *text.Buffer, p text.Pos) (lo, hi text.Pos) {
	p = b.ClampPos(p)
	runes := b.Line(p.Line).Runes()
	start, end := int(p.Col), int(p.Col)
	for start > 0 && isWordRune(runes[start-1]) {
		start--
	}
	for end < len(runes) && isWordRune(runes[end]) {
		end++
	}
	return text.Pos{Line: p.Line, Col: text.RuneIdx(start)}, text.Pos{Line: p.Line, Col: text.RuneIdx(end)}
}

// addNextOccurrence is C-d: the first press selects the word at point, and
// each later one adds a cursor on the next occurrence of the selection.
func (e *Editor) addNextOccurrence() error {
	w := e.active
	b := w.Buf
	if !b.MarkActive() {
		lo, hi := wordAt(b, w.Pt)
		if lo == hi {
			e.Echo("No word at point to select")
			return nil
		}
		b.SetMark(lo)
		b.ActivateMark()
		w.Pt = hi
		w.GoalCol = view.GoalColUnset
		return nil
	}

	lo, hi := text.OrderPos(b.Mark(), w.Pt)
	if lo.Line != hi.Line {
		e.Echo("The selection spans lines; multiple cursors need a selection within one line")
		return nil
	}
	needle := b.Text(lo, hi)
	cs := allCursors(w)
	// Search from the furthest selection end, so the next press moves on
	// rather than finding the same occurrence again.
	from := cs[0].Pt
	for _, c := range cs {
		from = maxOf(from, c.Pt, c.Mark)
	}
	mlo, mhi, ok := findOccurrence(b, needle, from, cs)
	if !ok {
		e.Echo("No other occurrence of %q", string(needle))
		return nil
	}
	w.Extra = append(w.Extra, view.Cursor{Pt: mhi, Mark: mlo, Goal: view.GoalColUnset})
	e.Echo("%d cursors", len(w.Extra)+1)
	return nil
}

// findOccurrence finds the next place needle occurs as a whole word - or as
// plain text, when the needle is not a word - that does not overlap a cursor's
// selection. It searches forward from from and wraps to the start of the
// buffer when nothing lies beyond.
func findOccurrence(b *text.Buffer, needle []rune, from text.Pos, cs []view.Cursor) (lo, hi text.Pos, ok bool) {
	if len(needle) == 0 {
		return text.Pos{}, text.Pos{}, false
	}
	wholeWord := isWordRune(needle[0])
	type match struct{ lo, hi text.Pos }
	var all []match
	for ln := 0; ln < b.NumLines(); ln++ {
		runes := b.Line(ln).Runes()
		for col := 0; col+len(needle) <= len(runes); col++ {
			if !runesAt(runes, col, needle) {
				continue
			}
			end := col + len(needle)
			if wholeWord && ((col > 0 && isWordRune(runes[col-1])) || (end < len(runes) && isWordRune(runes[end]))) {
				continue
			}
			all = append(all, match{
				lo: text.Pos{Line: ln, Col: text.RuneIdx(col)},
				hi: text.Pos{Line: ln, Col: text.RuneIdx(end)},
			})
		}
	}
	free := func(m match) bool {
		for _, c := range cs {
			clo, chi := text.OrderPos(c.Pt, c.Mark)
			if posLess(m.lo, chi) && posLess(clo, m.hi) {
				return false
			}
		}
		return true
	}
	// The matches from `from` onwards first; then, wrapping round, the rest.
	var after, before []match
	for _, m := range all {
		if posLess(m.lo, from) {
			before = append(before, m)
		} else {
			after = append(after, m)
		}
	}
	for _, m := range append(after, before...) {
		if free(m) {
			return m.lo, m.hi, true
		}
	}
	return text.Pos{}, text.Pos{}, false
}

// runesAt reports whether needle occurs in runes at col.
func runesAt(runes []rune, col int, needle []rune) bool {
	for i, r := range needle {
		if runes[col+i] != r {
			return false
		}
	}
	return true
}

// maxOf is the furthest of a point and a selection's other end.
func maxOf(p, a, b text.Pos) text.Pos {
	m := p
	for _, q := range []text.Pos{a, b} {
		if posLess(m, q) {
			m = q
		}
	}
	return m
}

// posLess orders positions in the buffer.
func posLess(p, q text.Pos) bool {
	return p.Line < q.Line || (p.Line == q.Line && p.Col < q.Col)
}

// multiMotionNames are the motion commands, which move every cursor. Each runs
// once for point, as usual, and then once for each extra cursor in turn.
var multiMotionNames = map[string]bool{
	"forward-char":           true,
	"backward-char":          true,
	"right-char":             true,
	"left-char":              true,
	"next-line":              true,
	"previous-line":          true,
	"forward-word":           true,
	"backward-word":          true,
	"move-beginning-of-line": true,
	"move-end-of-line":       true,
}

// leftMotion and rightMotion name the motions that move a selection's cursor
// to its start or its end. With a selection, these only collapse it there:
// the cursor does not then move on, which is the behaviour of every other
// multi-cursor editor.
var leftMotion = map[string]bool{"backward-char": true, "left-char": true}
var rightMotion = map[string]bool{"forward-char": true, "right-char": true}

// collapseForMotion prepares a motion with extra cursors. Each selection is
// dropped first: a left or right motion collapses it to the edge it points
// at, and does nothing else; any other motion moves from the cursor's own
// position, and so loses the selection. It reports whether point's motion
// should be skipped, and for each extra cursor whether its motion should be.
func (e *Editor) collapseForMotion(name string) (skipPoint bool, skipExtra []bool) {
	w := e.active
	b := w.Buf
	edge := leftMotion[name] || rightMotion[name]
	collapse := func(pt, mark text.Pos) (newPt text.Pos, only bool) {
		if pt == mark {
			return pt, false
		}
		lo, hi := text.OrderPos(pt, mark)
		switch {
		case leftMotion[name]:
			return lo, true
		case rightMotion[name]:
			return hi, true
		}
		return pt, false
	}
	if b.MarkActive() {
		newPt, only := collapse(w.Pt, b.Mark())
		w.Pt = newPt
		b.DeactivateMark()
		skipPoint = only && edge
	}
	skipExtra = make([]bool, len(w.Extra))
	for i := range w.Extra {
		c := w.Extra[i]
		newPt, only := collapse(c.Pt, c.Mark)
		w.Extra[i] = view.Cursor{Pt: newPt, Mark: newPt, Goal: c.Goal}
		skipExtra[i] = only && edge
	}
	return skipPoint, skipExtra
}

// moveExtraCursors runs a motion command at each extra cursor, except those
// whose selection the motion only collapsed. Point has already moved, so its
// own position and goal column are saved first and put back afterwards. Each
// cursor's point is what the motion moves; a motion never changes the mark.
func (e *Editor) moveExtraCursors(name string, skipExtra []bool) {
	w := e.active
	primPt, primGoal := w.Pt, w.GoalCol
	for i := range w.Extra {
		if i < len(skipExtra) && skipExtra[i] {
			continue
		}
		c := w.Extra[i]
		w.Pt = w.Buf.ClampPos(c.Pt)
		w.GoalCol = c.Goal
		_ = e.reg.Run(name, e)
		// collapseForMotion has already removed every selection, so the mark
		// follows the point. Keeping the old mark would turn the motion into a
		// one-character selection, which the next typed character replaces.
		w.Extra[i] = view.Cursor{Pt: w.Pt, Mark: w.Pt, Goal: w.GoalCol}
	}
	w.Pt, w.GoalCol = primPt, primGoal
	// Two cursors that land on the same spot would edit it twice, so the
	// later one is dropped.
	kept := w.Extra[:0]
	seen := map[text.Pos]bool{w.Pt: true}
	for _, c := range w.Extra {
		if seen[c.Pt] {
			continue
		}
		seen[c.Pt] = true
		kept = append(kept, c)
	}
	w.Extra = kept
}

// multiEditNames are the commands that act at every cursor. add-next-occurrence
// is also kept, because it adds to the cursors rather than dropping them.
var multiEditNames = map[string]bool{
	"self-insert-command":  true,
	"delete-backward-char": true,
	"delete-char":          true,
}

// multiEdit runs an edit command at every cursor, if one is active. It reports
// whether it handled the command, so the caller runs nothing else for it. Any
// other command drops the extra cursors and is run as usual.
func (e *Editor) multiEdit(name string) (bool, error) {
	if name == addNextOccurrence || multiMotionNames[name] {
		return false, nil
	}
	if !multiEditNames[name] {
		e.clearExtraCursors()
		return false, nil
	}

	var ins []rune
	if name == "self-insert-command" {
		if e.seq.LastRune == 0 {
			return true, errors.New("no self-inserting key recorded")
		}
		ins = []rune{e.seq.LastRune}
	}

	w := e.active
	b := w.Buf
	cs := allCursors(w)

	// Work from the last cursor to the first, so each edit lands after every
	// edit still to come: earlier positions in the buffer are untouched.
	order := make([]int, len(cs))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, c int) bool {
		lo1, _ := text.OrderPos(cs[order[a]].Pt, cs[order[a]].Mark)
		lo2, _ := text.OrderPos(cs[order[c]].Pt, cs[order[c]].Mark)
		return posLess(lo2, lo1)
	})

	b.BeginUndoGroup()
	defer b.EndUndoGroup()
	var firstErr error
	for _, i := range order {
		lo, hi := spanFor(b, name, cs[i], ins != nil)
		end, err := replaceSpan(b, lo, hi, ins)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		for j := range cs {
			if j == i {
				continue
			}
			cs[j].Pt = shiftAfterEdit(cs[j].Pt, lo, hi, end)
			cs[j].Mark = shiftAfterEdit(cs[j].Mark, lo, hi, end)
		}
		cs[i] = view.Cursor{Pt: end, Mark: end, Goal: view.GoalColUnset}
	}

	// Point and the buffer's mark take the first cursor's place. Extras are
	// left as bare cursors: an edit has consumed their selections.
	w.Pt = b.ClampPos(cs[0].Pt)
	w.GoalCol = view.GoalColUnset
	if cs[0].Pt != cs[0].Mark {
		b.SetMark(cs[0].Mark)
		b.ActivateMark()
	} else {
		b.ClearMark()
	}
	w.Extra = append([]view.Cursor(nil), cs[1:]...)
	return true, firstErr
}

// spanFor is the text a command replaces at one cursor. A selection is always
// replaced by typing; without one, typing replaces nothing and the deletions
// take the character before or after point.
func spanFor(b *text.Buffer, name string, c view.Cursor, typing bool) (lo, hi text.Pos) {
	if c.Pt != c.Mark {
		lo, hi = text.OrderPos(c.Pt, c.Mark)
		return lo, hi
	}
	p := b.ClampPos(c.Pt)
	switch {
	case typing:
		return p, p
	case name == "delete-backward-char":
		if p.Col > 0 {
			return text.Pos{Line: p.Line, Col: p.Col - 1}, p
		}
		if p.Line > 0 {
			return text.Pos{Line: p.Line - 1, Col: b.Line(p.Line - 1).Len()}, p
		}
		return p, p
	default: // delete-char
		if p.Col < b.Line(p.Line).Len() {
			return p, text.Pos{Line: p.Line, Col: p.Col + 1}
		}
		if p.Line < b.NumLines()-1 {
			return p, text.Pos{Line: p.Line + 1, Col: 0}
		}
		return p, p
	}
}

// replaceSpan replaces [lo, hi) with ins and returns the position just after
// the new text.
func replaceSpan(b *text.Buffer, lo, hi text.Pos, ins []rune) (text.Pos, error) {
	if lo != hi {
		if err := b.Delete(lo, hi); err != nil {
			return lo, err
		}
	}
	if len(ins) > 0 {
		if err := b.Insert(lo, ins); err != nil {
			return lo, err
		}
	}
	return mcAdvance(lo, ins), nil
}

// mcAdvance is the position reached by writing rs starting at p.
func mcAdvance(p text.Pos, rs []rune) text.Pos {
	line, col := p.Line, p.Col
	for _, r := range rs {
		if r == '\n' {
			line, col = line+1, 0
		} else {
			col++
		}
	}
	return text.Pos{Line: line, Col: col}
}

// shiftAfterEdit moves a position past an edit that replaced [lo, hi) with text
// ending at end. Positions at or before lo stay put; positions after hi move
// with the text around them.
func shiftAfterEdit(q, lo, hi, end text.Pos) text.Pos {
	if !posLess(lo, q) {
		return q
	}
	if posLess(q, hi) {
		return lo // inside the replaced text: cannot happen for disjoint cursors
	}
	if q.Line == hi.Line {
		return text.Pos{Line: end.Line, Col: end.Col + (q.Col - hi.Col)}
	}
	return text.Pos{Line: q.Line + end.Line - hi.Line, Col: q.Col}
}

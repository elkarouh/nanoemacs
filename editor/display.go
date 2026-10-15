package editor

import (
	"fmt"
	"regexp"
	"unicode"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
	"github.com/Borderliner/nem/view"
)

// registerDisplayCommands adds the commands that change how the editor looks
// rather than what a buffer contains.
//
// They close over the Editor instead of going through Env for the same reason
// recover-file does: display state lives in the theme, which Env cannot reach by
// design, and adding an interface method to serve one command would widen a
// surface sixty other commands share.
func registerDisplayCommands(e *Editor, reg *command.Registry) error {
	if err := reg.Register(command.Command{
		Name:        "symbol-overlay",
		Doc:         "Highlight the symbol at point throughout the buffer.",
		Interactive: true,
		Fn: func(env command.Env) error {
			b := env.Buf()
			symbol, ok := symbolAt(b, env.Win().Pt)
			if !ok {
				e.Echo("No symbol at point")
				return nil
			}
			active := e.symbols[b]
			if active == nil {
				active = make(map[string]int)
				e.symbols[b] = active
			}
			if _, found := active[symbol]; found {
				delete(active, symbol)
				if len(active) == 0 {
					delete(e.symbols, b)
				}
				e.Echo("Removed symbol overlay: %s", symbol)
			} else {
				used := make(map[int]bool, len(active))
				for _, c := range active {
					used[c] = true
				}
				color := 0
				for used[color] {
					color++
				}
				active[symbol] = color
				e.Echo("Highlighted symbol: %s", symbol)
			}
			return nil
		},
	}); err != nil {
		return err
	}
	for _, c := range []struct {
		name, doc string
		fn        func()
	}{
		{"symbol-overlay-next", "Move to the next occurrence of the highlighted symbol at point.", func() { e.symbolJump(1) }},
		{"symbol-overlay-prev", "Move to the previous occurrence of the highlighted symbol at point.", func() { e.symbolJump(-1) }},
		{"symbol-overlay-last", "Move to the last occurrence of the highlighted symbol at point.", func() { e.symbolEnd(true) }},
		{"symbol-overlay-first", "Move to the first occurrence of the highlighted symbol at point.", func() { e.symbolEnd(false) }},
	} {
		fn := c.fn
		if err := reg.Register(command.Command{
			Name:        c.name,
			Doc:         c.doc,
			Interactive: true,
			Fn:          func(command.Env) error { fn(); return nil },
		}); err != nil {
			return err
		}
	}
	if err := reg.Register(command.Command{
		Name:        "symbol-overlay-rename",
		Doc:         "Rename every occurrence of the highlighted symbol at point.",
		Interactive: true,
		Fn:          func(command.Env) error { return e.symbolRename() },
	}); err != nil {
		return err
	}
	if err := reg.Register(command.Command{
		Name:        "symbol-overlay-occur",
		Doc:         "List the lines containing the highlighted symbol at point.",
		Interactive: true,
		Fn:          func(command.Env) error { return e.symbolOccur() },
	}); err != nil {
		return err
	}
	if err := reg.Register(command.Command{
		Name:        "toggle-line-wrap",
		Doc:         "Wrap lines wider than the window, or cut them off at its edge.",
		Interactive: true,
		Fn: func(command.Env) error {
			e.SetLineWrap(!e.th.Wrap)
			if e.th.Wrap {
				e.Echo("Line wrap on")
			} else {
				e.Echo("Line wrap off")
			}
			return nil
		},
	}); err != nil {
		return err
	}
	return reg.Register(command.Command{
		Name:        "toggle-line-numbers",
		Doc:         "Show or hide the line-number gutter.",
		Interactive: true,
		Fn: func(command.Env) error {
			e.th.LineNumbers = !e.th.LineNumbers
			if e.th.LineNumbers {
				e.Echo("Line numbers on")
			} else {
				e.Echo("Line numbers off")
			}
			return nil
		},
	})
}

// SetLineWrap turns line wrapping on or off. The goal column each window
// keeps means a column of the line unwrapped and of the row wrapped, so it
// is let go of; and a window scrolled sideways or part way down a line starts
// its view afresh.
func (e *Editor) SetLineWrap(on bool) {
	e.th.Wrap = on
	for _, w := range e.tree.Windows() {
		w.GoalCol, w.LeftCol, w.TopRow = view.GoalColUnset, 0, 0
	}
}

func symbolRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) ||
		unicode.In(r, unicode.Mn, unicode.Mc)
}

func symbolAt(b *text.Buffer, p text.Pos) (string, bool) {
	if b == nil || b.NumLines() == 0 {
		return "", false
	}
	p = b.ClampPos(p)
	rs := b.Line(p.Line).View()
	col := int(p.Col)
	if col >= len(rs) && col > 0 {
		col--
	}
	if col < 0 || col >= len(rs) || !symbolRune(rs[col]) {
		return "", false
	}
	start, end := col, col+1
	for start > 0 && symbolRune(rs[start-1]) {
		start--
	}
	for end < len(rs) && symbolRune(rs[end]) {
		end++
	}
	return string(rs[start:end]), true
}

// symbolSpans reports what to emphasise on one line: the hits of an
// incremental search in progress, then active whole-symbol matches.
//
// The search's hits come first. A span covering a position is the first one
// found there, so the current hit wins over the others it sits among, and a
// symbol that is also a hit shows as the hit.
func (e *Editor) symbolSpans(b *text.Buffer, line int) []ui.SymbolSpan {
	out := e.searchSpans(b, line)
	if st := e.previewOf(b); st != nil {
		out = append(out, st.lines[line]...)
	}
	active := e.symbols[b]
	if len(active) == 0 || b == nil || line < 0 || line >= b.NumLines() {
		return out
	}
	rs := b.Line(line).View()
	for i := 0; i < len(rs); {
		if !symbolRune(rs[i]) {
			i++
			continue
		}
		start := i
		for i < len(rs) && symbolRune(rs[i]) {
			i++
		}
		sym := string(rs[start:i])
		if color, ok := active[sym]; ok {
			out = append(out, ui.SymbolSpan{
				Start: text.RuneIdx(start),
				End:   text.RuneIdx(i),
				Color: color,
			})
		}
	}
	return out
}

// searchSpans reports the hits of the incremental search in progress, if it is
// searching b. A prompt that is not a search, or a search of another buffer,
// has none.
func (e *Editor) searchSpans(b *text.Buffer, line int) []ui.SymbolSpan {
	if e.mini == nil || e.mini.opts.Session == nil || b == nil {
		return nil
	}
	var out []ui.SymbolSpan
	for _, h := range e.mini.opts.Session.Hits(b, line) {
		out = append(out, ui.SymbolSpan{Start: h.Start, End: h.End, Hit: true, Current: h.Current})
	}
	return out
}

// symbolOverlayAtPoint reports whether point is on a symbol that has an
// overlay, which is when n and p stop being ordinary letters.
func (e *Editor) symbolOverlayAtPoint() bool {
	if e.mini != nil || e.active == nil {
		return false
	}
	b := e.active.Buf
	if len(e.symbols[b]) == 0 {
		return false
	}
	sym, ok := symbolAt(b, e.active.Pt)
	if !ok {
		return false
	}
	_, on := e.symbols[b][sym]
	return on
}

// symbolOccurrences lists where the whole symbol sym starts in b, in buffer
// order, and which of them point is on (-1 if none).
func symbolOccurrences(b *text.Buffer, sym string, pt text.Pos) (occ []text.Pos, cur int) {
	cur = -1
	pt = b.ClampPos(pt)
	for ln := 0; ln < b.NumLines(); ln++ {
		rs := b.Line(ln).View()
		for i := 0; i < len(rs); {
			if !symbolRune(rs[i]) {
				i++
				continue
			}
			start := i
			for i < len(rs) && symbolRune(rs[i]) {
				i++
			}
			if string(rs[start:i]) != sym {
				continue
			}
			occ = append(occ, text.Pos{Line: ln, Col: text.RuneIdx(start)})
			// Point may sit just past the end of the symbol (symbolAt
			// allows that), so the end is inclusive.
			if ln == pt.Line && start <= int(pt.Col) && int(pt.Col) <= i {
				cur = len(occ) - 1
			}
		}
	}
	return occ, cur
}

// symbolGo moves point to occurrence idx of sym, leaving the mark behind.
func (e *Editor) symbolGo(sym string, occ []text.Pos, idx int) {
	w := e.active
	b := w.Buf
	b.SetMark(w.Pt)
	b.DeactivateMark()
	w.Pt = occ[idx]
	w.GoalCol = view.GoalColUnset
	e.Echo("%s: %d/%d", sym, idx+1, len(occ))
}

// symbolJump moves point to the start of the next (step > 0) or previous
// occurrence of the highlighted symbol at point, wrapping at the buffer's ends
// as symbol-overlay does.
func (e *Editor) symbolJump(step int) {
	w := e.active
	sym, ok := symbolAt(w.Buf, w.Pt)
	if !ok {
		e.Echo("No symbol at point")
		return
	}
	occ, cur := symbolOccurrences(w.Buf, sym, w.Pt)
	if len(occ) == 0 {
		return
	}
	cur = max(cur, 0)
	n := len(occ)
	e.symbolGo(sym, occ, ((cur+step)%n+n)%n)
}

// symbolEnd moves point to the last (or first) occurrence of the symbol at point.
func (e *Editor) symbolEnd(last bool) {
	w := e.active
	sym, ok := symbolAt(w.Buf, w.Pt)
	if !ok {
		e.Echo("No symbol at point")
		return
	}
	occ, _ := symbolOccurrences(w.Buf, sym, w.Pt)
	if len(occ) == 0 {
		return
	}
	idx := 0
	if last {
		idx = len(occ) - 1
	}
	e.symbolGo(sym, occ, idx)
}

// symbolRename replaces every whole-symbol occurrence of the symbol at point
// with a name read from the minibuffer, as one undoable change. The overlay
// follows the new name, keeping its colour.
func (e *Editor) symbolRename() error {
	w := e.active
	b := w.Buf
	if b.ReadOnly() {
		return text.ErrReadOnly
	}
	sym, ok := symbolAt(b, w.Pt)
	if !ok {
		e.Echo("No symbol at point")
		return nil
	}
	name, err := e.ReadString(command.ReadOpts{
		Prompt:  fmt.Sprintf("Rename %s to: ", sym),
		Initial: sym,
		History: "replace",
	})
	if err != nil {
		return err
	}
	if name == "" || name == sym {
		return nil
	}
	occ, cur := symbolOccurrences(b, sym, w.Pt)
	if len(occ) == 0 {
		return nil
	}
	old, repl := []rune(sym), []rune(name)
	// Edit from the end so the positions still to be edited stay valid.
	b.BeginUndoGroup()
	defer b.EndUndoGroup()
	for i := len(occ) - 1; i >= 0; i-- {
		from := occ[i]
		to := text.Pos{Line: from.Line, Col: from.Col + text.RuneIdx(len(old))}
		if err := b.Delete(from, to); err != nil {
			return err
		}
		if err := b.Insert(from, repl); err != nil {
			return err
		}
	}
	// Point stays on the occurrence it was on, shifted by the renames before
	// it on its line.
	if cur >= 0 {
		at := occ[cur]
		shift := 0
		for _, o := range occ[:cur] {
			if o.Line == at.Line {
				shift += len(repl) - len(old)
			}
		}
		w.Pt = b.ClampPos(text.Pos{Line: at.Line, Col: at.Col + text.RuneIdx(shift)})
		w.GoalCol = view.GoalColUnset
	}
	if active := e.symbols[b]; active != nil {
		if color, on := active[sym]; on {
			delete(active, sym)
			if _, taken := active[name]; !taken {
				active[name] = color
			}
		}
	}
	e.Echo("Renamed %d occurrence%s of %s to %s", len(occ), plural(len(occ)), sym, name)
	return nil
}

// symbolOccur lists the lines holding the symbol at point, in an *Occur*
// buffer, as occur does for a regexp.
func (e *Editor) symbolOccur() error {
	w := e.active
	sym, ok := symbolAt(w.Buf, w.Pt)
	if !ok {
		e.Echo("No symbol at point")
		return nil
	}
	return e.occur(w.Buf, sym, symbolRegexp(sym))
}

// symbolRegexp matches sym as a whole symbol. Go's \b is ASCII-only, so it
// serves an ASCII symbol; any other is fenced by what is not a symbol rune.
func symbolRegexp(sym string) *regexp.Regexp {
	q := regexp.QuoteMeta(sym)
	for _, r := range sym {
		if r > unicode.MaxASCII {
			const not = `[^\p{L}\p{N}\p{M}_]`
			return regexp.MustCompile(`(?:^|` + not + `)` + q + `(?:$|` + not + `)`)
		}
	}
	return regexp.MustCompile(`\b` + q + `\b`)
}

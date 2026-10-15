package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Borderliner/nem/text"
)

// target is one file named on the command line, with where to put point in
// it. Line and Col are one-based, as the user writes them; zero means not
// given, so the file opens where it was left last time.
type target struct {
	Path string
	Line int
	Col  int
}

// parseTargets reads the file arguments the way emacs does: a +LINE or
// +LINE:COLUMN argument applies to the file that follows it, so
//
//	nem +44 file.go
//
// opens file.go at line 44. An argument that starts with + but is not a
// number, such as +notes.txt, is a file name like any other. A +LINE with no
// file after it is an error rather than silently dropped.
func parseTargets(args []string) ([]target, error) {
	var out []target
	var line, col int
	pending := ""
	for _, a := range args {
		if l, c, ok := parsePosition(a); ok {
			line, col, pending = l, c, a
			continue
		}
		out = append(out, target{Path: a, Line: line, Col: col})
		line, col, pending = 0, 0, ""
	}
	if pending != "" {
		return nil, fmt.Errorf("%s is not followed by a file", pending)
	}
	return out, nil
}

// parsePosition parses +LINE or +LINE:COLUMN. Values below one are taken as
// one, as emacs does with +0.
func parsePosition(a string) (line, col int, ok bool) {
	rest, found := strings.CutPrefix(a, "+")
	if !found || rest == "" {
		return 0, 0, false
	}
	ls, cs, hasCol := strings.Cut(rest, ":")
	line, err := strconv.Atoi(ls)
	if err != nil {
		return 0, 0, false
	}
	if hasCol {
		if col, err = strconv.Atoi(cs); err != nil {
			return 0, 0, false
		}
		col = max(col, 1)
	}
	return max(line, 1), col, true
}

// place sets where a window visiting b puts point, when t asked for a line. A
// +LINE asked for wins over where the file was left last time. It is set as
// the place to come back to, so it holds for every file named, not only the
// one shown first. A line or column past the end clamps to it.
func (t target) place(b *text.Buffer) {
	if t.Line < 1 {
		return
	}
	p := b.ClampPos(text.Pos{Line: t.Line - 1, Col: text.RuneIdx(max(t.Col-1, 0))})
	b.SetSavePoint(p)
	b.SetSaveTop(p.Line)
}

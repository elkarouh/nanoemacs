package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Borderliner/nem/text"
)

// TestRgRunResultsOpenAtTheMatch checks that a match in dired R's results
// visits its file at the matched line, as RET does on an fd result.
func TestRgRunResultsOpenAtTheMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\nneedle: here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e, _ := newTestEditor(t)
	if err := e.rgRun(dir, "needle"); err != nil {
		t.Fatalf("rgRun: %v", err)
	}
	res := e.Active().Buf
	if e.rgOf(res) == nil {
		t.Fatal("the results buffer holds no rg results")
	}
	// The match's text is coloured, as rg colours it at a terminal, and the
	// colour sits on the matched line of the results, after the heading and
	// the blank line.
	if got := e.rgOf(res).lines[rgHeaderLines]; len(got) == 0 {
		t.Fatal("the matched line has no colour over it")
	}
	// The match is on the third line of the results, after the heading and
	// the blank line.
	e.Active().Pt = text.Pos{Line: rgHeaderLines}
	if err := e.rgFindMatch(); err != nil {
		t.Fatalf("rgFindMatch: %v", err)
	}
	if got := e.Active().Buf.Path(); got != path {
		t.Fatalf("visited %q, want %q", got, path)
	}
	if got := e.Active().Pt.Line; got != 2 {
		t.Fatalf("point on line %d, want the matched line 2", got)
	}
}

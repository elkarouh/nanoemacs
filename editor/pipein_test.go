package editor

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Borderliner/nem/ui"
	"github.com/gdamore/tcell/v2"
)

// pending reports what the reader has taken in and not yet shown, and whether
// it has seen the end of its input.
func pending(pi *pipeIn) (n int, done bool) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	return len(pi.in), pi.done
}

// waitFor polls cond until it holds, failing the test if it never does. The
// reader runs in its own goroutine, so the test waits for it as the loop would
// be woken by it.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// Piped input is shown in a buffer of its own, named for where it came from,
// with no file behind it, so saving asks for one rather than writing over
// anything.
func TestFollowTextMakesFilelessBuffer(t *testing.T) {
	e, _ := newTestEditor(t)
	b := e.FollowText("*stdin*", strings.NewReader("one\ntwo\n"))
	pi := e.pipes[b]
	waitFor(t, "the end of the input", func() bool { _, done := pending(pi); return done })
	pi.show()
	if got := b.String(); got != "one\ntwo" {
		t.Errorf("text = %q, want %q", got, "one\ntwo")
	}
	if b.Path() != "" {
		t.Errorf("path = %q, want none", b.Path())
	}
	if got, ok := e.BufferByName("*stdin*"); !ok || got != b {
		t.Error("buffer is not found under *stdin*")
	}
	if b.Modified() {
		t.Error("text that only came in is marked modified")
	}
	if !b.ReadOnly() {
		t.Error("piped text is editable; its colours would be left on the wrong words")
	}
}

// A second piped input while the first buffer is open must not replace it.
func TestFollowTextUniqueName(t *testing.T) {
	e, _ := newTestEditor(t)
	first := e.FollowText("*stdin*", strings.NewReader("a\n"))
	second := e.FollowText("*stdin*", strings.NewReader("b\n"))
	if first == second {
		t.Fatal("two inputs share one buffer")
	}
	if got := e.BufferName(second); got == "*stdin*" {
		t.Errorf("second buffer kept the taken name %q", got)
	}
}

// Text shows as it arrives: a line is in the buffer once its newline has come,
// while the command writing it is still running.
func TestFollowTextShowsLinesAsTheyArrive(t *testing.T) {
	e, _ := newTestEditor(t)
	r, w := io.Pipe()
	b := e.FollowText("*stdin*", r)
	pi := e.pipes[b]

	go func() {
		_, _ = io.WriteString(w, "first\nsec")
	}()
	waitFor(t, "the first read", func() bool { n, _ := pending(pi); return n > 0 })
	pi.show()
	if got := b.String(); got != "first\n" {
		t.Fatalf("while running: text = %q, want the complete line only", got)
	}

	go func() {
		_, _ = io.WriteString(w, "ond\nthird\n")
		_ = w.Close()
	}()
	waitFor(t, "the end of the input", func() bool { _, done := pending(pi); return done })
	pi.show()
	if got := b.String(); got != "first\nsecond\nthird" {
		t.Errorf("after the end: text = %q, want all three lines", got)
	}
}

// A colour code split across two reads is still read as one code, and its
// colour lands on the line it styled.
func TestFollowTextColoursAcrossReads(t *testing.T) {
	e, _ := newTestEditor(t)
	r, w := io.Pipe()
	b := e.FollowText("*stdin*", r)
	pi := e.pipes[b]

	go func() {
		_, _ = io.WriteString(w, "plain\n\x1b[1;37;4")
		_, _ = io.WriteString(w, "1mCHANGES\x1b[0m BY\n")
		_ = w.Close()
	}()
	waitFor(t, "the end of the input", func() bool { _, done := pending(pi); return done })
	pi.show()
	if got := b.String(); got != "plain\nCHANGES BY" {
		t.Fatalf("text = %q, want %q", got, "plain\nCHANGES BY")
	}
	var found *ui.SymbolSpan
	for _, sp := range e.symbolSpans(b, 1) {
		if sp.Styled && sp.Start == 0 && sp.End == 7 {
			found = &sp
		}
	}
	if found == nil {
		t.Fatalf("no colour over CHANGES on line 1, got %+v", e.symbolSpans(b, 1))
	}
	if _, bg, _ := found.Style.Decompose(); bg != tcell.PaletteColor(1) {
		t.Errorf("background = %v, want palette 1 (the 41 code's red)", bg)
	}
	if got := e.symbolSpans(b, 0); len(got) != 0 {
		t.Errorf("line 0 has colour %+v, want none", got)
	}
}

// Input that ends without a newline keeps its last line, and input that ends
// with one does not show a stray empty line at the foot.
func TestFollowTextEndsOnItsLastLine(t *testing.T) {
	e, _ := newTestEditor(t)
	for _, in := range []string{"a\nb\n", "a\nb"} {
		b := e.FollowText("*stdin*", strings.NewReader(in))
		pi := e.pipes[b]
		waitFor(t, "the end of the input", func() bool { _, done := pending(pi); return done })
		pi.show()
		if got := b.NumLines(); got != 2 {
			t.Errorf("input %q: %d lines, want 2", in, got)
		}
		if got := b.Line(1).Runes(); string(got) != "b" {
			t.Errorf("input %q: last line = %q, want b", in, string(got))
		}
	}
}

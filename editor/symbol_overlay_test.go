package editor

import (
	"testing"

	"github.com/Borderliner/nem/text"
)

func TestSymbolOverlayMultipleSymbolsDifferentColors(t *testing.T) {
	e, _ := newTestEditor(t, "apple banana apple banana cherry")
	b := e.Buf()

	// Point at origin: "apple"
	press(t, e, "C-.")
	if len(e.symbols[b]) != 1 {
		t.Fatalf("expected 1 symbol overlay, got %d", len(e.symbols[b]))
	}
	colorApple, ok := e.symbols[b]["apple"]
	if !ok {
		t.Fatalf("expected 'apple' to be highlighted")
	}

	// Move point to "banana"
	e.Active().Pt = text.Pos{Line: 0, Col: 6}
	press(t, e, "C-.")
	if len(e.symbols[b]) != 2 {
		t.Fatalf("expected 2 symbol overlays, got %d", len(e.symbols[b]))
	}
	colorBanana, ok := e.symbols[b]["banana"]
	if !ok {
		t.Fatalf("expected 'banana' to be highlighted")
	}
	if colorApple == colorBanana {
		t.Errorf("expected different colors for apple and banana, got %d and %d", colorApple, colorBanana)
	}

	// Check symbolSpans
	spans := e.symbolSpans(b, 0)
	if len(spans) != 4 {
		t.Fatalf("expected 4 symbol spans on line 0, got %d", len(spans))
	}
	// "apple" at 0..5
	if spans[0].Start != 0 || spans[0].End != 5 || spans[0].Color != colorApple {
		t.Errorf("span 0 = %+v, want apple with color %d", spans[0], colorApple)
	}
	// "banana" at 6..12
	if spans[1].Start != 6 || spans[1].End != 12 || spans[1].Color != colorBanana {
		t.Errorf("span 1 = %+v, want banana with color %d", spans[1], colorBanana)
	}
	// "apple" at 13..18
	if spans[2].Start != 13 || spans[2].End != 18 || spans[2].Color != colorApple {
		t.Errorf("span 2 = %+v, want apple with color %d", spans[2], colorApple)
	}
	// "banana" at 19..25
	if spans[3].Start != 19 || spans[3].End != 25 || spans[3].Color != colorBanana {
		t.Errorf("span 3 = %+v, want banana with color %d", spans[3], colorBanana)
	}

	// Toggle off "apple"
	e.Active().Pt = text.Pos{Line: 0, Col: 0}
	press(t, e, "C-.")
	if _, ok := e.symbols[b]["apple"]; ok {
		t.Errorf("apple should have been removed")
	}

	// Highlight "cherry"
	e.Active().Pt = text.Pos{Line: 0, Col: 26}
	press(t, e, "C-.")
	colorCherry, ok := e.symbols[b]["cherry"]
	if !ok {
		t.Fatalf("expected 'cherry' to be highlighted")
	}
	// It should have reused the lowest freed color (colorApple)
	if colorCherry != colorApple {
		t.Errorf("expected cherry to reuse freed color %d, got %d", colorApple, colorCherry)
	}
}

func TestSymbolOverlayNextPrev(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo", "baz foo")
	b := e.Buf()
	press(t, e, "C-.") // highlight foo
	if _, ok := e.symbols[b]["foo"]; !ok {
		t.Fatal("foo not highlighted")
	}
	at := func(l, c int) {
		t.Helper()
		if p := e.Active().Pt; p != (text.Pos{Line: l, Col: text.RuneIdx(c)}) {
			t.Fatalf("point = %+v, want %d:%d", p, l, c)
		}
	}
	press(t, e, "n")
	at(0, 8)
	press(t, e, "n")
	at(1, 4)
	press(t, e, "n") // wraps
	at(0, 0)
	press(t, e, "p") // wraps back
	at(1, 4)
	press(t, e, "p")
	at(0, 8)
	if got := e.Buf().Line(0).String(); got != "foo bar foo" {
		t.Fatalf("buffer changed: %q", got)
	}
	// Off a highlighted symbol, n still types.
	e.Active().Pt = text.Pos{Line: 0, Col: 4}
	press(t, e, "n")
	if got := e.Buf().Line(0).String(); got != "foo nbar foo" {
		t.Fatalf("n off a symbol should insert, got %q", got)
	}
}

func TestSymbolOverlayFirstLast(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar foo", "baz foo")
	press(t, e, "C-.")
	press(t, e, ">")
	if p := e.Active().Pt; p != (text.Pos{Line: 1, Col: 4}) {
		t.Fatalf("> went to %+v, want 1:4", p)
	}
	press(t, e, "<")
	if p := e.Active().Pt; p != (text.Pos{Line: 0, Col: 0}) {
		t.Fatalf("< went to %+v, want 0:0", p)
	}
}

func TestSymbolOverlayRename(t *testing.T) {
	e, scr := newTestEditor(t, "foo foobar foo", "foo")
	b := e.Buf()
	press(t, e, "C-.")
	color := e.symbols[b]["foo"]
	feed(t, scr, key(t, "C-a", "C-k"), txt("quux"), key(t, "RET"))
	press(t, e, "r")
	wantText(t, e, "quux foobar quux\nquux")
	if c, ok := e.symbols[b]["quux"]; !ok || c != color {
		t.Errorf("overlay should follow the rename with colour %d, got %v %d", color, ok, c)
	}
	if _, ok := e.symbols[b]["foo"]; ok {
		t.Errorf("old name should no longer be highlighted")
	}
	press(t, e, "C-/")
	wantText(t, e, "foo foobar foo\nfoo")
}

func TestSymbolOverlayOccur(t *testing.T) {
	e, _ := newTestEditor(t, "foo bar", "foobar", "x foo")
	press(t, e, "C-.")
	press(t, e, "o")
	if got := e.BufferName(e.Buf()); got != occurName {
		t.Fatalf("buffer = %q, want %q", got, occurName)
	}
	if st := e.grepOf(e.Buf()); st == nil || st.count() != 2 {
		t.Fatalf("want 2 matching lines (foobar is a different symbol)")
	}
}

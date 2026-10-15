package ansistyle

import "testing"

// The mdcat --ansi sample, as pasted from a real run.
const mdcatSample = "\x1b[94m\x1b[104m \x1b[0m\x1b[1m\x1b[97m\x1b[104mTitle\x1b[0m\x1b[94m\x1b[104m \x1b[0m\n\n" +
	"Some\x1b[3m emphasis\x1b[0m,\x1b[33m code\x1b[0m, and a \x1b]8;;https://example.com/\x1b\\\x1b[34mlink\x1b[0m\x1b]8;;\x1b\\.\n\n" +
	"• one\n• two\n"

func TestSampleFromMdcat(t *testing.T) {
	text, spans := Parse(mdcatSample)
	want := " Title \n\nSome emphasis, code, and a link.\n\n• one\n• two\n"
	if text != want {
		t.Fatalf("text = %q\nwant %q", text, want)
	}
	// Each span's text, checked against the plain text by rune offset.
	rs := []rune(text)
	got := map[string]Style{}
	for _, sp := range spans {
		got[string(rs[sp.Start:sp.End])] = sp.Style
	}
	if s, ok := got["Title"]; !ok || !s.Bold || s.FG != Palette(15) || s.BG != Palette(12) {
		t.Errorf("Title style = %+v (present %v), want bold, bright white on bright blue", s, ok)
	}
	if s := got[" emphasis"]; !s.Italic {
		t.Errorf(" emphasis style = %+v, want italic", s)
	}
	if s := got[" code"]; s.FG != Palette(3) {
		t.Errorf(" code style = %+v, want yellow", s)
	}
	if s := got["link"]; s.FG != Palette(4) {
		t.Errorf("link style = %+v, want blue", s)
	}
}

func TestOSC8LinkTargetIsDropped(t *testing.T) {
	text, _ := Parse("\x1b]8;;https://example.com\x07click\x1b]8;;\x07!")
	if text != "click!" {
		t.Errorf("text = %q, want %q", text, "click!")
	}
}

func TestSpansAreOffsetsInRunes(t *testing.T) {
	// "é" and "•" are one rune each but several bytes; spans count runes.
	text, spans := Parse("é•\x1b[1mx\x1b[0m")
	if text != "é•x" {
		t.Fatalf("text = %q", text)
	}
	if len(spans) != 1 || spans[0].Start != 2 || spans[0].End != 3 || !spans[0].Style.Bold {
		t.Errorf("spans = %+v, want one bold span over rune 2", spans)
	}
}

func TestSpansMergeAcrossReset(t *testing.T) {
	// Adjacent runs of the same style are one span, even across a reset that
	// leaves no text between them.
	_, spans := Parse("\x1b[1mab\x1b[0m\x1b[1mcd\x1b[0m")
	if len(spans) != 1 || spans[0].End != 4 {
		t.Errorf("spans = %+v, want one span over all four runes", spans)
	}
	// Unstyled text between them keeps them apart.
	_, spans = Parse("\x1b[1mab\x1b[0m-\x1b[1mcd\x1b[0m")
	if len(spans) != 2 {
		t.Errorf("spans = %+v, want two", spans)
	}
}

func TestExtendedColours(t *testing.T) {
	_, spans := Parse("\x1b[38;5;196mA\x1b[0m\x1b[38;2;10;20;30mB\x1b[0m")
	if len(spans) != 2 {
		t.Fatalf("spans = %+v", spans)
	}
	if spans[0].Style.FG != Palette(196) {
		t.Errorf("palette colour = %v, want index 196", spans[0].Style.FG)
	}
	r, g, b, ok := spans[1].Style.FG.IsRGB()
	if !ok || r != 10 || g != 20 || b != 30 {
		t.Errorf("truecolour = %d,%d,%d (%v), want 10,20,30", r, g, b, ok)
	}
}

func TestNonSGRSequencesAreRemoved(t *testing.T) {
	text, spans := Parse("a\x1b[2Kb\x1b(Bc")
	if text != "abc" || len(spans) != 0 {
		t.Errorf("text = %q, spans = %v; want abc and none", text, spans)
	}
}

func TestTruncatedSequenceKeepsWhatCameBefore(t *testing.T) {
	text, _ := Parse("kept\x1b[1")
	if text != "kept" {
		t.Errorf("text = %q, want %q", text, "kept")
	}
	text, _ = Parse("kept\x1b]8;;http")
	if text != "kept" {
		t.Errorf("text = %q, want %q", text, "kept")
	}
}

func TestCarriageReturnsDropped(t *testing.T) {
	if text, _ := Parse("a\r\nb"); text != "a\nb" {
		t.Errorf("text = %q", text)
	}
}

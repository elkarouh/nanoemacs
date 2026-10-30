package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestSixelDraw(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	got, err := SixelDraw(buf.Bytes(), 1, 2, 2, 1, 5, 6)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"\x1b7\x1b[3;2H", "\x1bPq\"1;1;10;6", "#180;2;100;0;0", "#180!10~", "\x1b\\\x1b8"} {
		if !strings.Contains(got, want) {
			t.Errorf("SixelDraw lacks %q in %q", want, got[:min(len(got), 120)])
		}
	}
	if _, err := SixelDraw([]byte("nope"), 0, 0, 1, 1, 1, 1); err == nil {
		t.Error("SixelDraw accepted a non-PNG")
	}
}

func TestSixelSupported(t *testing.T) {
	t.Setenv("NEM_IMAGES", "")
	t.Setenv("TMUX", "")
	t.Setenv("STY", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("WT_SESSION", "abc")
	if !SixelSupported() || !UseSixel() {
		t.Error("Windows Terminal should use sixel")
	}
	t.Setenv("KITTY_WINDOW_ID", "1")
	if UseSixel() {
		t.Error("kitty graphics should win over sixel")
	}
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TMUX", "x")
	if SixelSupported() {
		t.Error("tmux should rule sixel out")
	}
}

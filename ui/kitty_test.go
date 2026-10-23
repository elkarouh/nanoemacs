package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestFitImage(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		iw, ih, mc, mr, cw, ch, c, r int
	}{
		{"wide image limited by columns", 1600, 400, 80, 40, 10, 20, 80, 10},
		{"tall image limited by rows", 400, 1600, 80, 20, 10, 20, 10, 20},
		{"small image is not enlarged", 100, 100, 80, 40, 10, 20, 10, 5},
		{"unknown cell size assumes 1:2", 1000, 1000, 80, 20, 0, 0, 40, 20},
		{"nothing to draw", 0, 10, 80, 20, 10, 20, 0, 0},
	} {
		c, r := FitImage(tc.iw, tc.ih, tc.mc, tc.mr, tc.cw, tc.ch)
		if c != tc.c || r != tc.r {
			t.Errorf("%s: FitImage = %d x %d, want %d x %d", tc.name, c, r, tc.c, tc.r)
		}
	}
}

func TestKittyDrawChunks(t *testing.T) {
	big := bytes.Repeat([]byte{1, 2, 3}, 6000) // > 4096 base64 bytes
	got := KittyDraw(big, 2, 3, 10, 5)
	if !strings.HasPrefix(got, "\x1b7\x1b[4;3H\x1b_Ga=T,f=100,c=10,r=5,q=2,C=1,m=1;") {
		t.Errorf("unexpected start: %q", got[:60])
	}
	if !strings.HasSuffix(got, "\x1b\\\x1b8") || !strings.Contains(got, "\x1b_Gm=0;") {
		t.Error("last chunk must say m=0 and the cursor must be restored")
	}
	if n := strings.Count(got, "\x1b_G"); n != 6 {
		t.Errorf("got %d chunks, want 6", n)
	}
}

func TestImagePNGPassesPNGThrough(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, w, h, err := ImagePNG(buf.Bytes())
	if err != nil || w != 3 || h != 2 || !bytes.Equal(out, buf.Bytes()) {
		t.Errorf("ImagePNG = %d bytes, %d x %d, %v", len(out), w, h, err)
	}
	if _, _, _, err := ImagePNG([]byte("not an image")); err == nil {
		t.Error("garbage decoded without error")
	}
}

func TestKittySupportedRefusesMultiplexers(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TMUX", "")
	if !KittySupported() {
		t.Error("kitty not recognised")
	}
	t.Setenv("TMUX", "/tmp/tmux-1/default,1,0")
	if KittySupported() {
		t.Error("tmux should disable images")
	}
}

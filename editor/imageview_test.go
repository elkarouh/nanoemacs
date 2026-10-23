package editor

import "testing"

func TestIsImageFile(t *testing.T) {
	for path, want := range map[string]bool{
		"a.png": true, "dir/B.JPG": true, "c.jpeg": true, "d.gif": true,
		"e.txt": false, "png": false, "f.png.bak": false,
	} {
		if got := isImageFile(path); got != want {
			t.Errorf("isImageFile(%q) = %v, want %v", path, got, want)
		}
	}
	if !allImages([]string{"a.png", "b.gif"}) || allImages([]string{"a.png", "b.go"}) {
		t.Error("allImages misjudged a mix")
	}
}

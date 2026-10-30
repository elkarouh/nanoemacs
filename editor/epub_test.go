package editor

import "testing"

func TestEPUBFiles(t *testing.T) {
	if !isEPUBFile("Book.EPUB") || isEPUBFile("book.pdf") {
		t.Error("isEPUBFile misjudged an extension")
	}
	if !onlyEPUBs([]string{"a.epub", "b.epub"}) || onlyEPUBs([]string{"a.epub", "b.pdf"}) || onlyEPUBs(nil) {
		t.Error("onlyEPUBs misjudged a list")
	}
}

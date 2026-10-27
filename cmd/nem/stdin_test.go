package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPipedForAPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	piped, err := isPiped(r)
	if err != nil {
		t.Fatalf("isPiped: %v", err)
	}
	if !piped {
		t.Fatal("a pipe was not reported as input")
	}
}

func TestIsPipedForARedirectedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.txt")
	if err := os.WriteFile(path, []byte("from a file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	piped, err := isPiped(f)
	if err != nil {
		t.Fatalf("isPiped: %v", err)
	}
	if !piped {
		t.Fatal("a redirected file was not reported as input")
	}
}

// A terminal, or /dev/null, has no text to offer, so nem starts as it does
// with nothing piped in. /dev/null stands in for both: each is a character
// device.
func TestIsPipedForACharacterDevice(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Skip("no null device:", err)
	}
	defer f.Close()

	piped, err := isPiped(f)
	if err != nil {
		t.Fatalf("isPiped: %v", err)
	}
	if piped {
		t.Error("a character device was reported as input")
	}
}

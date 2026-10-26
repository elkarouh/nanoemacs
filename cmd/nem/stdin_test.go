package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadStdinPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	want := "total 8\n-rw-r--r-- 1 me me 0 notes.txt\n"
	if _, err := w.WriteString(want); err != nil {
		t.Fatal(err)
	}
	w.Close()

	got, piped, err := readStdin(r)
	if err != nil {
		t.Fatalf("readStdin: %v", err)
	}
	if !piped {
		t.Fatal("a pipe was not reported as input")
	}
	if string(got) != want {
		t.Errorf("readStdin = %q, want %q", got, want)
	}
}

func TestReadStdinEmptyPipeIsStillInput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	w.Close()

	_, piped, err := readStdin(r)
	if err != nil {
		t.Fatalf("readStdin: %v", err)
	}
	if !piped {
		t.Fatal("an empty pipe was not reported as input")
	}
}

func TestReadStdinRedirectedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.txt")
	if err := os.WriteFile(path, []byte("from a file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, piped, err := readStdin(f)
	if err != nil {
		t.Fatalf("readStdin: %v", err)
	}
	if !piped || string(got) != "from a file\n" {
		t.Errorf("readStdin = %q, %v; want the file's text, true", got, piped)
	}
}

// A terminal, or /dev/null, has no text to offer, so nem starts as it does
// with nothing piped in. /dev/null stands in for both: each is a character
// device.
func TestReadStdinCharacterDeviceIsNotInput(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Skip("no null device:", err)
	}
	defer f.Close()

	got, piped, err := readStdin(f)
	if err != nil {
		t.Fatalf("readStdin: %v", err)
	}
	if piped || got != nil {
		t.Errorf("readStdin = %q, %v; want nothing, false", got, piped)
	}
}

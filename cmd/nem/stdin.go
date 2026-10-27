package main

import "os"

// stdinName is the buffer name for text piped into nem, as emacs names the
// buffers of text that came from outside the editor.
const stdinName = "*stdin*"

// isPiped reports whether f is a pipe or a file rather than a terminal or
// /dev/null. Those last two are character devices, with no text to offer, so
// nem starts as it does with no input at all. A pipe or a file counts even when
// it is empty for now: its text may still be on the way.
func isPiped(f *os.File) (bool, error) {
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	return fi.Mode()&os.ModeCharDevice == 0, nil
}

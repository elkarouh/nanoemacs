package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// An EPUB is read in a terminal reader, when one is installed: nem steps aside
// and the reader has the terminal, as yazi does, until it quits. Nothing is
// converted or drawn by nem itself.

// epubReaders are the terminal EPUB readers nem looks for, in order of
// preference. Each takes the book as its one argument.
var epubReaders = []string{"epy", "baca", "bk"}

// isEPUBFile reports whether path names an EPUB.
func isEPUBFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".epub")
}

// epubReader returns the path of the first terminal EPUB reader on the PATH, or
// "" when there is none.
func epubReader() string {
	for _, name := range epubReaders {
		if bin, err := exec.LookPath(name); err == nil {
			return bin
		}
	}
	return ""
}

// onlyEPUBs reports whether paths is not empty and every one is an EPUB.
func onlyEPUBs(paths []string) bool {
	for _, p := range paths {
		if !isEPUBFile(p) {
			return false
		}
	}
	return len(paths) > 0
}

// epubRun runs the reader at bin on path, on nem's terminal. It is a variable
// so the test binary can replace it: a test run must never start a real reader.
var epubRun = func(bin, path string) error {
	cmd := exec.Command(bin, path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(bin), err)
	}
	return nil
}

// readEPUB reads the EPUB at path in the terminal reader at bin, with nem
// stepped aside.
func (e *Editor) readEPUB(bin, path string) error {
	if e.scr != nil {
		if err := e.scr.Suspend(); err != nil {
			return err
		}
	}
	runErr := epubRun(bin, path)
	if e.scr != nil {
		if err := e.scr.Resume(); err != nil {
			return err
		}
		e.scr.Sync()
	}
	return runErr
}

package main

import (
	"io"
	"os"

	"github.com/Borderliner/nem/ansistyle"
)

// stdinName is the buffer name for text piped into nem, as emacs names the
// buffers of text that came from outside the editor.
const stdinName = "*stdin*"

// readStdin reads what is piped or redirected into f, and reports whether there
// was any input to read. A terminal, or /dev/null, is a character device with
// nothing to offer, and gives false: nem then starts as it does with no
// input at all. A pipe or a file gives true, even when it is empty.
func readStdin(f *os.File) ([]byte, bool, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	if fi.Mode()&os.ModeCharDevice != 0 {
		return nil, false, nil
	}
	data, err := io.ReadAll(f)
	return data, true, err
}

// plainStdin returns the text in data without its styling. A program run with
// colour, such as ls --color=always, puts escape sequences in its output; they
// are the styling, not text, and a buffer that held them would show them as
// garbage. The words the styling was over are kept.
func plainStdin(data []byte) []byte {
	text, _ := ansistyle.Parse(string(data))
	return []byte(text)
}

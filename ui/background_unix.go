//go:build unix

package ui

import (
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// DetectBackground asks the terminal for its background colour with OSC 11 and
// remembers the answer for TerminalIsLight. It is a better source than
// COLORFGBG, which most terminals never set.
//
// Call it before the screen is opened: once tcell owns the terminal it reads
// every byte, and the reply would arrive as keystrokes. A terminal that does
// not know the query says nothing, so the wait is short and a silent terminal
// leaves the COLORFGBG guess in place.
func DetectBackground() {
	if os.Getenv("TERM") == "dumb" {
		return
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer tty.Close()
	fd := int(tty.Fd())
	if !term.IsTerminal(fd) {
		return
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return
	}
	defer term.Restore(fd, old)

	if _, err := tty.WriteString("\x1b]11;?\x1b\\"); err != nil {
		return
	}
	reply := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 64)
		for {
			n, err := tty.Read(buf)
			sb.Write(buf[:n])
			s := sb.String()
			if err != nil || strings.HasSuffix(s, "\a") || strings.HasSuffix(s, "\x1b\\") {
				reply <- s
				return
			}
		}
	}()
	select {
	case s := <-reply:
		if light, ok := parseOSC11(s); ok {
			detectedLight = &light
		}
	case <-time.After(200 * time.Millisecond):
		// No answer. The reader goroutine is left blocked on the tty and ends
		// with the deferred Close, which makes its Read fail.
	}
}

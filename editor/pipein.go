package editor

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Borderliner/nem/ansistyle"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
	"github.com/gdamore/tcell/v2"
)

// Text piped into nem can still be arriving from a command that is running, so
// it is read in the background and shown as it comes, as a command's output is
// in process.go. The reader only collects bytes and wakes the loop; the loop
// appends them to the buffer, so the buffer is still touched by one goroutine.

// pipeIn is one piped input being shown in a buffer.
type pipeIn struct {
	e *Editor
	b *text.Buffer

	// mu guards in and done, which the reader writes and the loop takes.
	mu   sync.Mutex
	in   []byte // read, not yet shown
	done bool   // the reader has seen the end of the input

	woken atomic.Bool

	// Loop only. tail is the last line, held back until its newline comes, so
	// that a line split across reads is shown whole and its colour codes are
	// never cut in two. ended is set once the input has finished.
	tail  []byte
	ended bool
}

// pipeWake is the event that tells the loop piped input has arrived.
type pipeWake struct{ in *pipeIn }

// FollowText shows what r delivers in a new buffer named name, as it arrives.
// It returns at once; the text comes in through the loop, so FollowText must be
// called before Loop runs or from within it.
//
// The buffer has no file, and is read-only: the colours are over the text as it
// came, so an edit would leave them on the wrong words. A name already taken
// gets a suffix, so the text is never lost to another buffer.
func (e *Editor) FollowText(name string, r io.Reader) *text.Buffer {
	b := text.NewBuffer()
	b.SetReadOnly(true)
	e.adopt(b, e.uniqueName(name))
	pi := &pipeIn{e: e, b: b}
	e.pipes[b] = pi
	go pi.read(r)
	return b
}

// read collects what r delivers until its end, waking the loop for each read.
func (pi *pipeIn) read(r io.Reader) {
	buf := make([]byte, 32<<10)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			pi.mu.Lock()
			pi.in = append(pi.in, buf[:n]...)
			pi.mu.Unlock()
			pi.wake()
		}
		if err != nil {
			pi.mu.Lock()
			pi.done = true
			pi.mu.Unlock()
			pi.wake()
			return
		}
	}
}

// wake asks the loop to show what has been read. One wake is pending at a time:
// the loop clears the flag before it takes the bytes, so a read that lands
// after the take posts a wake of its own.
func (pi *pipeIn) wake() {
	if pi.woken.Swap(true) {
		return
	}
	for range 1000 {
		if pi.e.scr != nil && pi.e.scr.PostEvent(tcell.NewEventInterrupt(pipeWake{pi})) == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	pi.woken.Store(false)
}

// show appends what has been read to the buffer, on the loop. Whole lines are
// shown as they complete; the last line waits for its newline, or for the end
// of the input.
func (pi *pipeIn) show() {
	pi.woken.Store(false)
	pi.mu.Lock()
	data, done := pi.in, pi.done
	pi.in = nil
	pi.mu.Unlock()
	if pi.ended {
		return
	}

	data = append(pi.tail, data...)
	pi.tail = nil
	if !done {
		cut := bytes.LastIndexByte(data, '\n') + 1
		pi.tail = append([]byte(nil), data[cut:]...)
		data = data[:cut]
	}
	if len(data) > 0 {
		pi.insert(data)
	}
	if done {
		pi.ended = true
		pi.dropFinalNewline()
	}
}

// insert appends data, whole lines, to the buffer, and the colours it carries
// over the lines they are on.
func (pi *pipeIn) insert(data []byte) {
	plain, spans := ansistyle.Parse(string(data))
	b := pi.b
	base := b.NumLines() - 1 // the empty last line, where this text starts
	b.SetReadOnly(false)
	err := b.Insert(b.End(), []rune(plain))
	b.SetReadOnly(true)
	if err != nil {
		pi.e.Echo("piped input: %v", err)
		return
	}
	// Not an edit to keep: quitting must not ask about text that only came in.
	b.SetModified(false)
	if len(spans) == 0 {
		return
	}
	st := pi.e.styles[b]
	if st == nil {
		st = map[int][]ui.SymbolSpan{}
		pi.e.styles[b] = st
	}
	for line, ss := range previewLines(plain, spans) {
		st[base+line] = append(st[base+line], ss...)
	}
}

// dropFinalNewline takes the empty line that the input's last newline leaves
// at the foot of the buffer, so a finished input ends on its last line as a
// file read from disk does.
func (pi *pipeIn) dropFinalNewline() {
	b := pi.b
	n := b.NumLines()
	if n < 2 || b.Line(n-1).Len() != 0 {
		return
	}
	b.SetReadOnly(false)
	err := b.Delete(text.Pos{Line: n - 2, Col: b.Line(n - 2).Len()}, b.End())
	b.SetReadOnly(true)
	if err == nil {
		b.SetModified(false)
	}
}

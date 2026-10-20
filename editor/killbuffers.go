package editor

import (
	"fmt"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
)

// kill-some-buffers asks about each buffer in turn, and kill-all-buffers kills
// every one after asking once. Neither kills *scratch*: the last buffer cannot
// go, so it stays, and one is made if there is none.

// killable lists the buffers the two commands may kill: all but *scratch*.
func (e *Editor) killable() []*text.Buffer {
	scratch := e.byName[ui.ScratchName]
	var out []*text.Buffer
	for _, b := range e.buffers {
		if b != scratch {
			out = append(out, b)
		}
	}
	return out
}

// ensureScratch makes sure a *scratch* buffer exists to be left behind.
func (e *Editor) ensureScratch() {
	if _, ok := e.byName[ui.ScratchName]; !ok {
		e.NewBuffer(ui.ScratchName)
	}
}

// confirmModified asks whether to kill b anyway when it holds unsaved changes
// to a file. It reports whether b may go.
func (e *Editor) confirmModified(b *text.Buffer) (bool, error) {
	if !b.Modified() || b.Path() == "" {
		return true, nil
	}
	c, err := e.ReadChar(fmt.Sprintf("Buffer %s modified; kill anyway? (y/n) ", e.BufferName(b)), []rune{'y', 'n'})
	return c == 'y', err
}

// killAllBuffers is kill-all-buffers.
func (e *Editor) killAllBuffers() error {
	bufs := e.killable()
	if len(bufs) == 0 {
		e.Echo("No buffers to kill")
		return nil
	}
	c, err := e.ReadChar(fmt.Sprintf("Kill %d buffer%s? (y/n) ", len(bufs), plural(len(bufs))), []rune{'y', 'n'})
	if err != nil || c != 'y' {
		return err
	}
	e.ensureScratch()
	killed := 0
	for _, b := range bufs {
		ok, err := e.confirmModified(b)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := e.KillBuffer(b); err != nil {
			return err
		}
		killed++
	}
	e.Echo("Killed %d buffer%s", killed, plural(killed))
	return nil
}

// killSomeBuffers is kill-some-buffers: y kills the buffer shown, n keeps it,
// and q stops asking.
func (e *Editor) killSomeBuffers() error {
	bufs := e.killable()
	if len(bufs) == 0 {
		e.Echo("No buffers to kill")
		return nil
	}
	e.ensureScratch()
	killed := 0
ask:
	for _, b := range bufs {
		c, err := e.ReadChar(fmt.Sprintf("Kill buffer %s? (y/n/q) ", e.BufferName(b)), []rune{'y', 'n', 'q'})
		if err != nil {
			return err
		}
		switch c {
		case 'q':
			break ask
		case 'n':
			continue
		}
		ok, err := e.confirmModified(b)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := e.KillBuffer(b); err != nil {
			return err
		}
		killed++
	}
	e.Echo("Killed %d buffer%s", killed, plural(killed))
	return nil
}

// registerKillBufferCommands adds kill-some-buffers and kill-all-buffers.
func registerKillBufferCommands(e *Editor, reg *command.Registry) error {
	for _, c := range []command.Command{
		{Name: "kill-some-buffers", Doc: "Ask about each buffer in turn and kill the ones you answer y to; q stops asking. *scratch* is kept.",
			Interactive: true, Fn: func(command.Env) error { return e.killSomeBuffers() }},
		{Name: "kill-all-buffers", Doc: "Kill every buffer after asking once, and again for each with unsaved changes. *scratch* is kept.",
			Interactive: true, Fn: func(command.Env) error { return e.killAllBuffers() }},
	} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

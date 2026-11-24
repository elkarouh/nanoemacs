package editor

import (
	"path/filepath"
	"slices"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/keymap"
	"github.com/Borderliner/nem/text"
)

// A file that yazi opens is read-only at first, so a key pressed by mistake
// cannot change it. e makes it editable, and it stays so. Until then the
// file's keys are the viewer's. The keypad's keys read as their navigation
// keys whatever NumLock says: with NumLock on they type digits, so they are
// bound here by their digits as well as by the keys they send with NumLock off.

// keypadBindings are the keypad's navigation keys, in a read-only view or a
// preview. Keypad 7 and 1 go to the start and end of the buffer, as Ctrl-Home
// and Ctrl-End do, since with NumLock on Ghostty sends them as plain digits.
// Keypad 5, 0 and . do nothing: they have no navigation key.
var keypadBindings = []struct{ Spec, Command string }{
	{"8", "previous-line"},
	{"2", "next-line"},
	{"4", "left-char"},
	{"6", "right-char"},
	{"9", "scroll-down-command"},
	{"3", "scroll-up-command"},
	{"7", "beginning-of-buffer"},
	{"1", "end-of-buffer"},
}

// viewBindings are a file's keys while it is read-only from yazi, on top of the
// keypad's. Every other key falls through to the global map, as in a preview.
var viewBindings = []struct{ Spec, Command string }{
	{"e", "view-edit"},
	{"C-<home>", "beginning-of-buffer"},
	{"C-<end>", "end-of-buffer"},
}

// newViewKeymap builds the view keymap from keypadBindings and viewBindings.
func newViewKeymap() (*keymap.Map, error) {
	m := keymap.New()
	for _, b := range append(slices.Clone(keypadBindings), viewBindings...) {
		if err := bindSpec(m, b.Spec, b.Command); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// registerViewCommands adds view-edit.
func registerViewCommands(e *Editor, reg *command.Registry) error {
	return reg.Register(command.Command{
		Name:        "view-edit",
		Doc:         "Make this read-only file, opened from yazi, editable. Until then e does this, and the keypad reads as its navigation keys.",
		Interactive: true,
		Fn:          func(command.Env) error { return e.viewEdit() },
	})
}

// viewOpened makes b, a file just opened from yazi, read-only for viewing.
func (e *Editor) viewOpened(b *text.Buffer) {
	b.SetReadOnly(true)
	e.viewing[b] = true
}

// viewEdit makes the active buffer editable, if it is a file yazi opened
// read-only.
func (e *Editor) viewEdit() error {
	b := e.active.Buf
	if !e.viewing[b] {
		return nil
	}
	b.SetReadOnly(false)
	delete(e.viewing, b)
	e.Echo("%s is editable", e.BufferName(b))
	return nil
}

// hasFileBuffer reports whether a buffer of the file at path is already open.
func (e *Editor) hasFileBuffer(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	for _, b := range e.buffers {
		if b.Path() == abs {
			return true
		}
	}
	return false
}

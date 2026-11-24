package editor

import (
	"path/filepath"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/keymap"
	"github.com/Borderliner/nem/text"
)

// A file that yazi opens is read-only at first, so a key pressed by mistake
// cannot change it. e makes it editable, and it stays so. Until then the
// file's keys are the viewer's: the keypad's 9 and 7 read as PgUp and Home,
// whatever NumLock says, since with NumLock on they type digits.

// viewBindings are a file's keys while it is read-only from yazi. Every other
// key falls through to the global map, as in a preview.
var viewBindings = []struct{ Spec, Command string }{
	{"e", "view-edit"},
	{"9", "scroll-down-command"},
	{"7", "move-beginning-of-line"},
}

// newViewKeymap builds the view keymap from viewBindings.
func newViewKeymap() (*keymap.Map, error) {
	m := keymap.New()
	for _, b := range viewBindings {
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
		Doc:         "Make this read-only file, opened from yazi, editable. Until then e does this, and the keypad's 9 and 7 read as PgUp and Home.",
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

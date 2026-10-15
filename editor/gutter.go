package editor

import (
	"github.com/Borderliner/nem/gitdiff"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
)

// The gutter's change signs: which lines of a file differ from git's HEAD.
//
// They are worked out when a file is opened and when it is saved, and stored
// per buffer, so drawing only looks them up. Edits made since the last save do
// not move the signs until the next save: the diff is against what git holds,
// and that is what changes on save.

// refreshGutter recomputes b's signs from git. A buffer with no file, a file
// git does not know, or one that is not text has none.
func (e *Editor) refreshGutter(b *text.Buffer) {
	if b == nil {
		return
	}
	p := b.Path()
	if p == "" || e.raw[b] {
		delete(e.gutter, b)
		return
	}
	repo, _ := git1Repo(p) // "" when the file is not git1-tracked
	marks, err := gitdiff.Run(p, repo)
	if err != nil {
		delete(e.gutter, b)
		return
	}
	// A removal at the very end of the file is marked on the line before it,
	// since there is no line after to carry the mark.
	if n := b.NumLines(); n > 0 {
		if c, ok := marks[n]; ok {
			delete(marks, n)
			if _, taken := marks[n-1]; !taken {
				marks[n-1] = c
			}
		}
	}
	e.gutter[b] = marks
}

// changeAt reports how line of b differs from HEAD, for the gutter. Drawing asks
// for every visible line of every window, so this is a map lookup and nothing
// more.
func (e *Editor) changeAt(b *text.Buffer, line int) ui.Change {
	switch e.gutter[b][line] {
	case gitdiff.Added:
		return ui.ChangeAdded
	case gitdiff.Changed:
		return ui.ChangeChanged
	case gitdiff.Deleted:
		return ui.ChangeDeleted
	}
	return ui.ChangeNone
}

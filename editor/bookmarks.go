package editor

import (
	"errors"
	"path/filepath"
	"sort"

	"github.com/Borderliner/nem/command"
)

// Bookmarks are dired's, as emacs has them: C-x r m names the directory of the
// dired buffer, and C-x r b goes to a named one. A name holds one directory,
// and setting a name again moves it. They last as long as the editor runs.

// registerBookmarkCommands adds dired-bookmark-set and dired-bookmark-jump.
func registerBookmarkCommands(e *Editor, reg *command.Registry) error {
	cmds := []command.Command{
		{Name: "dired-bookmark-set", Doc: "Bookmark the directory of this dired buffer under a name, which C-x r b goes to.",
			Fn: func(command.Env) error {
				st := e.diredOf(e.active.Buf)
				if st == nil {
					return errors.New("not a dired buffer")
				}
				name, err := e.ReadString(command.ReadOpts{
					Prompt:  "Set bookmark named: ",
					Initial: filepath.Base(st.dir),
					History: "bookmark",
				})
				if err != nil {
					return err
				}
				if name == "" {
					return errors.New("a bookmark needs a name")
				}
				e.bookmarks[name] = st.dir
				e.Echo("Bookmark %s set at %s", name, st.dir)
				return nil
			}},
		{Name: "dired-bookmark-jump", Doc: "Go to the directory of a bookmark, read by its name with completion.",
			Fn: func(command.Env) error {
				name, err := e.ReadString(command.ReadOpts{
					Prompt:       "Jump to bookmark: ",
					History:      "bookmark",
					Complete:     command.CompleteFrom(e.bookmarkNames()),
					RequireMatch: true,
				})
				if err != nil {
					return err
				}
				dir, ok := e.bookmarks[name]
				if !ok {
					return errors.New("no bookmark named " + name)
				}
				db, err := e.Dired(dir)
				if err != nil {
					return err
				}
				e.active.Visit(db)
				return nil
			}},
	}
	for _, c := range cmds {
		c.Interactive = true
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// bookmarkNames lists the bookmark names, sorted.
func (e *Editor) bookmarkNames() []string {
	names := make([]string, 0, len(e.bookmarks))
	for n := range e.bookmarks {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

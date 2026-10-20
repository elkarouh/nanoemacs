package editor

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Borderliner/nem/command"
)

// Bookmarks are dired's, as emacs has them: C-x r m names the directory of the
// dired buffer, and C-x r b goes to a named one. A name holds one directory,
// and setting a name again moves it. They are kept in nem's memory between sessions.

// bookmarkListName is the name of the buffer the bookmark list shows in.
const bookmarkListName = "*Bookmarks*"

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
				e.setDirBookmark(name, st.dir)
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
		{Name: "dired-bookmark-list", Doc: "List the bookmarks, each name with its directory, in a read-only buffer.",
			Fn: func(command.Env) error {
				var sb strings.Builder
				sb.WriteString("Bookmarks\n\n")
				for _, n := range e.bookmarkNames() {
					sb.WriteString(n + "  " + e.bookmarks[n] + "\n")
				}
				b, ok := e.byName[bookmarkListName]
				if !ok {
					b = e.NewBuffer(e.uniqueName(bookmarkListName))
				}
				if err := fillReadOnly(b, sb.String()); err != nil {
					return err
				}
				e.active.Visit(b)
				return nil
			}},
		{Name: "dired-bookmark-delete", Doc: "Delete a bookmark, read by its name with completion.",
			Fn: func(command.Env) error {
				name, err := e.ReadString(command.ReadOpts{
					Prompt:       "Delete bookmark: ",
					History:      "bookmark",
					Complete:     command.CompleteFrom(e.bookmarkNames()),
					RequireMatch: true,
				})
				if err != nil {
					return err
				}
				if _, ok := e.bookmarks[name]; !ok {
					return errors.New("no bookmark named " + name)
				}
				e.deleteDirBookmark(name)
				e.Echo("Bookmark %s deleted", name)
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

// setDirBookmark bookmarks dir under name and remembers it for later sessions.
func (e *Editor) setDirBookmark(name, dir string) {
	e.bookmarks[name] = dir
	e.mem.SetDirBookmark(name, dir)
	e.saveMemory()
}

// deleteDirBookmark removes the bookmark called name, here and in the memory.
func (e *Editor) deleteDirBookmark(name string) {
	delete(e.bookmarks, name)
	e.mem.RemoveDirBookmark(name)
	e.saveMemory()
}

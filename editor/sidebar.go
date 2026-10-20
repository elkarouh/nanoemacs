package editor

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/keymap"
	"github.com/Borderliner/nem/project"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
	"github.com/Borderliner/nem/view"
)

// C-b toggles the side panel: a window along the left edge with two tabs, the
// directory tree of the project and the bookmarks. It is an ordinary window
// showing a read-only buffer, so it scrolls, takes point and shares the frame
// like any other; what is chosen in it opens in the window you came from.
//
// C-b opens the panel and moves into it; C-b in the panel closes it. With the
// panel open and another window selected, C-b moves into the panel.

// sidebarName is the panel buffer's name.
const sidebarName = "*Sidebar*"

// The panel's tabs.
const (
	sideTree = iota
	sideBookmarks
)

// sideHeaderLines is how many lines come before the tab's own: the tab line
// and a rule under it.
const sideHeaderLines = 2

// sideEntry is what a panel line stands for. Path is empty for a line that is
// only a label.
type sideEntry struct {
	Path string
	Dir  bool
}

// sidebarState is what the panel shows.
type sidebarState struct {
	tab        int
	root       string          // the tree's top directory
	expanded   map[string]bool // directories opened in the tree
	showHidden bool
	lines      []sideEntry // the entry of each line of the buffer
	spans      map[int][]ui.SymbolSpan
	width      int
}

// sidebarBindings is the panel's keymap. The buffer is read-only, so printable
// keys are free to take.
var sidebarBindings = []struct{ Spec, Command string }{
	{"TAB", "sidebar-switch-tab"},
	{"RET", "sidebar-open"},
	{"o", "sidebar-open"},
	{"b", "sidebar-bookmark"},
	{"d", "sidebar-unbookmark"},
	{"r", "sidebar-set-root"},
	{"^", "sidebar-parent"},
	{".", "sidebar-toggle-hidden"},
	{"g", "sidebar-refresh"},
	{"n", "next-line"},
	{"p", "previous-line"},
	{"q", "sidebar-toggle"},
}

// newSidebarKeymap builds the panel's keymap from sidebarBindings.
func newSidebarKeymap() (*keymap.Map, error) {
	m := keymap.New()
	for _, b := range sidebarBindings {
		if err := bindSpec(m, b.Spec, b.Command); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// sideOpen reports whether the panel is in the frame.
func (e *Editor) sideOpen() bool { return e.side != nil && e.tree.Contains(e.side) }

// sidebarToggle is C-b.
func (e *Editor) sidebarToggle(env command.Env) error {
	// A prompt keeps its emacs C-b: back one character.
	if e.mini != nil {
		return env.Run("backward-char")
	}
	switch {
	case e.sideOpen() && e.active == e.side:
		return e.sidebarClose()
	case e.sideOpen():
		e.sidePrev = e.active
		e.active = e.side
		return nil
	}
	return e.sidebarShow()
}

// sidebarClose takes the panel out of the frame and goes back to the window
// that was selected before it.
func (e *Editor) sidebarClose() error {
	if !e.sideOpen() {
		return nil
	}
	if err := e.tree.Delete(e.side); err != nil {
		return err
	}
	e.side = nil
	if e.sidePrev != nil && e.tree.Contains(e.sidePrev) {
		e.active = e.sidePrev
	} else {
		e.active = e.tree.Windows()[0]
	}
	return nil
}

// sidebarShow opens the panel and moves into it.
func (e *Editor) sidebarShow() error {
	if e.sideBuf == nil || e.byName[sidebarName] != e.sideBuf {
		e.sideBuf = e.NewBuffer(sidebarName)
		e.sideBuf.SetReadOnly(true)
		// The panel is not one of the files, so it is no tab in the strip.
		e.tabs = slices.DeleteFunc(e.tabs, func(b *text.Buffer) bool { return b == e.sideBuf })
		e.sidebar = &sidebarState{expanded: map[string]bool{}}
	}
	st := e.sidebar
	if st.root == "" {
		st.root = e.sidebarRoot()
	}
	sw, _ := e.scr.Size()
	st.width = min(34, max(22, sw/4))
	if err := e.sidebarRender(); err != nil {
		return err
	}
	win := &view.Window{Buf: e.sideBuf, GoalCol: view.GoalColUnset}
	e.tree.AttachLeft(win, st.width)
	e.side = win
	e.sidePrev = e.active
	e.active = win
	return nil
}

// sidebarRoot is the directory the tree starts from: the project of the active
// buffer, or else its directory, or else the working directory.
func (e *Editor) sidebarRoot() string {
	dir := e.bufferDir(e.active.Buf)
	if dir == "" {
		dir, _ = os.Getwd()
	}
	if root, ok := project.Root(dir); ok {
		return root
	}
	return dir
}

// sidebarWindow is the window the panel's choices open in.
func (e *Editor) sidebarWindow() *view.Window {
	if e.sidePrev != nil && e.sidePrev != e.side && e.tree.Contains(e.sidePrev) {
		return e.sidePrev
	}
	for _, w := range e.tree.Windows() {
		if w != e.side {
			return w
		}
	}
	return e.active
}

// sidebarRender writes the panel's text for its tab.
func (e *Editor) sidebarRender() error {
	st := e.sidebar
	var sb strings.Builder
	st.lines = st.lines[:0]
	st.spans = map[int][]ui.SymbolSpan{}
	add := func(line string, ent sideEntry) {
		sb.WriteString(line + "\n")
		st.lines = append(st.lines, ent)
	}

	tabs := []string{" Tree ", " Bookmarks "}
	head := ""
	for i, t := range tabs {
		start := utf8.RuneCountInString(head)
		head += t
		if i == st.tab {
			st.spans[0] = append(st.spans[0], ui.SymbolSpan{
				Start: text.RuneIdx(start), End: text.RuneIdx(start + utf8.RuneCountInString(t)),
				Styled: true, Style: tcell.StyleDefault.Reverse(true).Bold(true),
			})
		}
	}
	add(head, sideEntry{})
	add(strings.Repeat("─", max(st.width, 8)), sideEntry{})

	if st.tab == sideTree {
		add(filepath.Base(st.root)+"/", sideEntry{Path: st.root, Dir: true})
		e.sidebarTree(st.root, 0, add)
	} else {
		e.sidebarBookmarkLines(add)
	}
	return fillReadOnly(e.sideBuf, sb.String())
}

// sidebarTree lists dir's entries under it, the opened directories expanded.
func (e *Editor) sidebarTree(dir string, depth int, add func(string, sideEntry)) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	sort.SliceStable(ents, func(i, j int) bool {
		if ents[i].IsDir() != ents[j].IsDir() {
			return ents[i].IsDir()
		}
		return strings.ToLower(ents[i].Name()) < strings.ToLower(ents[j].Name())
	})
	st := e.sidebar
	for _, en := range ents {
		name := en.Name()
		if !st.showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		isDir := en.IsDir()
		if !isDir && en.Type()&os.ModeSymlink != 0 {
			if fi, err := os.Stat(path); err == nil && fi.IsDir() {
				isDir = true
			}
		}
		pad := strings.Repeat("  ", depth+1)
		switch {
		case isDir && st.expanded[path]:
			add(pad+"▾ "+name+"/", sideEntry{Path: path, Dir: true})
			e.sidebarTree(path, depth+1, add)
		case isDir:
			add(pad+"▸ "+name+"/", sideEntry{Path: path, Dir: true})
		default:
			add(pad+"  "+name, sideEntry{Path: path})
		}
	}
}

// sidebarBookmarkLines lists the bookmarked files and then the bookmarked
// directories.
func (e *Editor) sidebarBookmarkLines(add func(string, sideEntry)) {
	add("Files", sideEntry{})
	files := e.mem.Bookmarks
	if len(files) == 0 {
		add("  (b in Tree, C-x r f)", sideEntry{})
	}
	for _, p := range files {
		add("  "+filepath.Base(p)+"  "+e.diredShort(filepath.Dir(p)), sideEntry{Path: p})
	}
	add("", sideEntry{})
	add("Directories", sideEntry{})
	names := e.bookmarkNames()
	if len(names) == 0 {
		add("  (C-x r m in dired)", sideEntry{})
	}
	for _, n := range names {
		add("  "+n+"/  "+e.diredShort(e.bookmarks[n]), sideEntry{Path: e.bookmarks[n], Dir: true})
	}
}

// sidebarAtPoint is the entry on the panel's line at point.
func (e *Editor) sidebarAtPoint() (sideEntry, error) {
	if e.sideBuf == nil || e.active.Buf != e.sideBuf {
		return sideEntry{}, errors.New("not in the side panel")
	}
	line := e.active.Pt.Line
	if line < 0 || line >= len(e.sidebar.lines) {
		return sideEntry{}, nil
	}
	return e.sidebar.lines[line], nil
}

// sidebarRefresh redraws the panel and keeps point on the same entry when it
// is still there.
func (e *Editor) sidebarRefresh() error {
	cur, _ := e.sidebarAtPoint()
	line := e.active.Pt.Line
	if err := e.sidebarRender(); err != nil {
		return err
	}
	for i, ent := range e.sidebar.lines {
		if cur.Path != "" && ent == cur {
			line = i
			break
		}
	}
	e.active.Pt = e.sideBuf.ClampPos(text.Pos{Line: line})
	return nil
}

// sidebarOpen is RET: a directory in the tree opens or closes; a file, or a
// bookmark, is visited in the window the panel was entered from.
func (e *Editor) sidebarOpen() error {
	ent, err := e.sidebarAtPoint()
	if err != nil || ent.Path == "" {
		return err
	}
	st := e.sidebar
	if ent.Dir && st.tab == sideTree && ent.Path != st.root {
		st.expanded[ent.Path] = !st.expanded[ent.Path]
		return e.sidebarRefresh()
	}
	if ent.Dir && st.tab == sideTree {
		return nil // the root line is only a label
	}
	target := e.sidebarWindow()
	var b *text.Buffer
	if ent.Dir {
		b, err = e.Dired(ent.Path)
	} else {
		b, err = e.OpenFile(ent.Path)
	}
	if err != nil {
		return err
	}
	e.active = target
	if ent.Dir {
		e.active.Visit(b)
	} else {
		e.VisitOpened(b)
	}
	return nil
}

// toggleFileBookmark bookmarks path, or takes it off the bookmarks if it is
// there, and reports which.
func (e *Editor) toggleFileBookmark(path string) {
	if slices.Contains(e.mem.Bookmarks, path) {
		e.mem.RemoveBookmark(path)
		e.Echo("Bookmark removed: %s", filepath.Base(path))
	} else {
		e.mem.AddBookmark(path)
		e.Echo("Bookmarked %s", filepath.Base(path))
	}
	e.saveMemory()
}

// registerSidebarCommands adds the panel's commands and bookmark-file.
func registerSidebarCommands(e *Editor, reg *command.Registry) error {
	panel := func(fn func() error) func(command.Env) error {
		return func(command.Env) error { return fn() }
	}
	cmds := []command.Command{
		{Name: "sidebar-toggle", Doc: "Open the side panel, with the project's directory tree and the bookmarks, and move into it; in the panel, close it.",
			Fn: e.sidebarToggle},
		{Name: "sidebar-switch-tab", Doc: "Switch the side panel between its Tree and Bookmarks tabs.",
			Fn: panel(func() error {
				if _, err := e.sidebarAtPoint(); err != nil {
					return err
				}
				e.sidebar.tab = 1 - e.sidebar.tab
				e.active.Pt = text.Pos{Line: sideHeaderLines}
				return e.sidebarRefreshTab()
			})},
		{Name: "sidebar-open", Doc: "Open the entry at point: expand or close a directory in the tree, visit a file or a bookmark in the other window.",
			Fn: panel(e.sidebarOpen)},
		{Name: "sidebar-bookmark", Doc: "Bookmark the file at point in the side panel (a directory is bookmarked under its name); again to remove it.",
			Fn: panel(func() error {
				ent, err := e.sidebarAtPoint()
				if err != nil || ent.Path == "" {
					return err
				}
				if ent.Dir {
					name := filepath.Base(ent.Path)
					if _, ok := e.bookmarks[name]; ok {
						e.deleteDirBookmark(name)
						e.Echo("Bookmark removed: %s", name)
					} else {
						e.setDirBookmark(name, ent.Path)
						e.Echo("Bookmarked %s", name)
					}
				} else {
					e.toggleFileBookmark(ent.Path)
				}
				return e.sidebarRefresh()
			})},
		{Name: "sidebar-unbookmark", Doc: "Remove the bookmark at point in the side panel's Bookmarks tab.",
			Fn: panel(func() error {
				ent, err := e.sidebarAtPoint()
				if err != nil || ent.Path == "" || e.sidebar.tab != sideBookmarks {
					return err
				}
				if ent.Dir {
					for n, d := range e.bookmarks {
						if d == ent.Path {
							e.deleteDirBookmark(n)
						}
					}
				} else {
					e.mem.RemoveBookmark(ent.Path)
					e.saveMemory()
				}
				return e.sidebarRefresh()
			})},
		{Name: "sidebar-set-root", Doc: "Make the directory at point the root of the side panel's tree.",
			Fn: panel(func() error {
				ent, err := e.sidebarAtPoint()
				if err != nil || !ent.Dir || e.sidebar.tab != sideTree {
					return err
				}
				e.sidebar.root = ent.Path
				e.active.Pt = text.Pos{Line: sideHeaderLines}
				return e.sidebarRefreshTab()
			})},
		{Name: "sidebar-parent", Doc: "Make the parent of the tree's root the root.",
			Fn: panel(func() error {
				if _, err := e.sidebarAtPoint(); err != nil {
					return err
				}
				e.sidebar.root = filepath.Dir(e.sidebar.root)
				e.sidebar.tab = sideTree
				return e.sidebarRefreshTab()
			})},
		{Name: "sidebar-toggle-hidden", Doc: "Show or hide dotfiles in the side panel's tree.",
			Fn: panel(func() error {
				if _, err := e.sidebarAtPoint(); err != nil {
					return err
				}
				e.sidebar.showHidden = !e.sidebar.showHidden
				return e.sidebarRefresh()
			})},
		{Name: "sidebar-refresh", Doc: "Read the side panel's directories and bookmarks again.",
			Fn: panel(func() error {
				if _, err := e.sidebarAtPoint(); err != nil {
					return err
				}
				return e.sidebarRefresh()
			})},
		{Name: "bookmark-file", Doc: "Bookmark this buffer's file, to find it in the side panel's Bookmarks tab; again to remove it.",
			Fn: panel(func() error {
				p := e.active.Buf.Path()
				if p == "" || e.diredOf(e.active.Buf) != nil {
					return errors.New("this buffer has no file to bookmark")
				}
				e.toggleFileBookmark(p)
				if e.sideOpen() && e.sidebar.tab == sideBookmarks {
					return e.sidebarRender()
				}
				return nil
			})},
	}
	for _, c := range cmds {
		c.Interactive = true
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// sidebarRefreshTab redraws the panel after the tab or root changed, with point
// at the top of the tab's own lines.
func (e *Editor) sidebarRefreshTab() error {
	if err := e.sidebarRender(); err != nil {
		return err
	}
	e.active.Pt = e.sideBuf.ClampPos(text.Pos{Line: sideHeaderLines})
	return nil
}

package editor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/icons"
	"github.com/Borderliner/nem/imenu"
	"github.com/Borderliner/nem/keymap"
	"github.com/Borderliner/nem/memory"
	"github.com/Borderliner/nem/project"
	"github.com/Borderliner/nem/syntax"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
	"github.com/Borderliner/nem/view"
)

// C-b toggles the side panel: a window along the left edge with four tabs - the
// directory tree of the project, the favourites, the open buffers and the
// outline of the file being edited. It is an ordinary window showing a
// read-only buffer, so it scrolls, takes point and shares the frame like any
// other; what is chosen in it opens in the window you came from.
//
// C-b opens the panel and moves into it; C-b in the panel closes it. With the
// panel open and another window selected, C-b moves into the panel.

// sidebarName is the panel buffer's name.
const sidebarName = "*Sidebar*"

// The panel's tabs.
const (
	sideTree = iota
	sideFavorites
	sideBuffers
	sideOutline
	sideTabs
)

// sideTabNames are the tabs' labels, and sideTabIcons the glyphs shown with
// them where icons are on (Font Awesome, as the file icons are).
var (
	sideTabNames = [sideTabs]string{"Tree", "Favs", "Bufs", "Outline"}
	sideTabIcons = [sideTabs]rune{'', '', '', ''}
)

// sideHeaderLines is how many lines come before the tab's own: the tab line
// and a rule under it.
const sideHeaderLines = 2

// The favourites' groups, in the order they are listed.
var sideGroups = []struct {
	Name, Kind string
	Icon       rune
}{
	{"Files", "", ''},
	{"Directories", memory.KindDir, ''},
	{"WWW", memory.KindLink, ''},
	{"Tools", memory.KindTool, ''},
}

// sideKind is what a panel line stands for.
type sideKind int

const (
	sideLabel  sideKind = iota // a line with nothing behind it
	sideNode                   // a directory or file of the tree
	sideGroup                  // a favourites group, which folds
	sideFile                   // a bookmarked file
	sideDirFav                 // a bookmarked directory
	sideLink                   // a web address
	sideTool                   // a command
	sideBuffer                 // an open buffer
	sideSymbol                 // a definition in the outline
)

// sideEntry is what a panel line stands for: Path the file, directory or
// address, Name the favourite's name or the group's, Buf the buffer, Line the
// definition's line, Cmd the tool's command.
type sideEntry struct {
	Kind sideKind
	Path string
	Dir  bool
	Name string
	Buf  *text.Buffer
	Line int
}

// sidebarState is what the panel shows.
type sidebarState struct {
	tab        int
	root       string          // the tree's top directory
	expanded   map[string]bool // directories opened in the tree
	folded     map[string]bool // favourites groups folded shut
	showHidden bool
	lines      []sideEntry // the entry of each line of the buffer
	classes    map[int][]syntax.Span
	marks      map[int][]ui.SymbolSpan
	tabCols    [sideTabs][2]int // where each tab's label sits on the first line
	width      int
	key        string // what the live tabs were last drawn from
}

// sidebarBindings is the panel's keymap. The buffer is read-only, so printable
// keys are free to take.
var sidebarBindings = []struct{ Spec, Command string }{
	{"TAB", "sidebar-next-tab"},
	{"<backtab>", "sidebar-previous-tab"},
	{"1", "sidebar-tree"},
	{"2", "sidebar-favorites"},
	{"3", "sidebar-buffers"},
	{"4", "sidebar-outline"},
	{"RET", "sidebar-open"},
	{"o", "sidebar-open"},
	{"a", "sidebar-add"},
	{"b", "sidebar-bookmark"},
	{"d", "sidebar-remove"},
	{"k", "sidebar-remove"},
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
		e.sidebar = &sidebarState{expanded: map[string]bool{}, folded: map[string]bool{}}
	}
	st := e.sidebar
	if st.root == "" {
		st.root = e.sidebarRoot()
	}
	sw, _ := e.scr.Size()
	st.width = min(34, max(24, sw/4))
	// The tab to show is read from the window about to lose focus, so the
	// outline is that of the file in front.
	e.sidePrev = e.active
	if err := e.sidebarRender(); err != nil {
		return err
	}
	win := &view.Window{Buf: e.sideBuf, GoalCol: view.GoalColUnset}
	e.tree.AttachLeft(win, st.width)
	e.side = win
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

// lineBuilder writes one panel line and the colours over it.
type lineBuilder struct {
	sb    strings.Builder
	n     int
	spans []syntax.Span
}

// add appends s, coloured as class (Plain for none).
func (l *lineBuilder) add(s string, class syntax.Class) {
	rc := utf8.RuneCountInString(s)
	if class != syntax.Plain && rc > 0 {
		l.spans = append(l.spans, syntax.Span{Start: l.n, End: l.n + rc, Class: class})
	}
	l.n += rc
	l.sb.WriteString(s)
}

// icon appends the glyph of ic and a space, when icons are on.
func (e *Editor) icon(l *lineBuilder, ic icons.Icon) {
	if e.icons {
		l.add(string(ic.Glyph)+" ", ic.Class)
	}
}

// sidebarRender writes the panel's text for its tab.
func (e *Editor) sidebarRender() error {
	st := e.sidebar
	var sb strings.Builder
	st.lines = st.lines[:0]
	st.classes = map[int][]syntax.Span{}
	st.marks = map[int][]ui.SymbolSpan{}
	add := func(l *lineBuilder, ent sideEntry) {
		line := len(st.lines)
		sb.WriteString(l.sb.String() + "\n")
		if len(l.spans) > 0 {
			st.classes[line] = l.spans
		}
		st.lines = append(st.lines, ent)
	}
	plain := func(s string, ent sideEntry) { l := &lineBuilder{}; l.add(s, syntax.Plain); add(l, ent) }
	dim := func(s string) { l := &lineBuilder{}; l.add(s, syntax.Comment); add(l, sideEntry{}) }

	// The tab line: with icons, a glyph for each tab and the active one named;
	// without, every tab by name.
	var head lineBuilder
	head.add(" ", syntax.Plain)
	for i := range sideTabs {
		label := sideTabNames[i]
		if e.icons {
			label = string(sideTabIcons[i])
			if i == st.tab {
				label += " " + sideTabNames[i]
			}
		}
		label = " " + label + " "
		start := head.n
		head.add(label, syntax.Plain)
		st.tabCols[i] = [2]int{start, head.n}
		if i == st.tab {
			st.marks[0] = append(st.marks[0], ui.SymbolSpan{
				Start: text.RuneIdx(start), End: text.RuneIdx(head.n),
				Styled: true, Style: tcell.StyleDefault.Reverse(true).Bold(true),
			})
		}
	}
	add(&head, sideEntry{})
	dim(strings.Repeat("─", max(st.width, 8)))

	switch st.tab {
	case sideTree:
		l := &lineBuilder{}
		e.icon(l, icons.For(st.root, icons.Dir))
		l.add(filepath.Base(st.root)+"/", syntax.Function)
		add(l, sideEntry{Kind: sideNode, Path: st.root, Dir: true})
		e.sidebarTree(st.root, 0, add)
	case sideFavorites:
		e.sidebarFavorites(add, plain, dim)
	case sideBuffers:
		e.sidebarBuffers(add, dim)
	case sideOutline:
		e.sidebarOutline(add, dim)
	}
	return fillReadOnly(e.sideBuf, sb.String())
}

// sidebarTree lists dir's entries under it, the opened directories expanded.
func (e *Editor) sidebarTree(dir string, depth int, add func(*lineBuilder, sideEntry)) {
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
		l := &lineBuilder{}
		l.add(strings.Repeat("  ", depth+1), syntax.Plain)
		if isDir {
			mark := "▸ "
			if st.expanded[path] {
				mark = "▾ "
			}
			l.add(mark, syntax.Comment)
			e.icon(l, icons.For(name, icons.Dir))
			l.add(name+"/", syntax.Function)
			add(l, sideEntry{Kind: sideNode, Path: path, Dir: true})
			if st.expanded[path] {
				e.sidebarTree(path, depth+1, add)
			}
			continue
		}
		l.add("  ", syntax.Plain)
		e.icon(l, icons.For(name, icons.File))
		l.add(name, syntax.Plain)
		add(l, sideEntry{Kind: sideNode, Path: path})
	}
}

// sidebarFavorites lists the favourites as a tree: Files, Directories, WWW and
// Tools, each folding, with what was added under it.
func (e *Editor) sidebarFavorites(add func(*lineBuilder, sideEntry), plain func(string, sideEntry), dim func(string)) {
	st := e.sidebar
	for _, g := range sideGroups {
		var names []string
		switch g.Kind {
		case "":
			names = slices.Clone(e.mem.Bookmarks)
		case memory.KindDir:
			names = e.bookmarkNames()
		case memory.KindLink:
			names = sortedNames(e.mem.Links)
		case memory.KindTool:
			names = sortedNames(e.mem.Tools)
		}
		l := &lineBuilder{}
		mark := "▾ "
		if st.folded[g.Name] {
			mark = "▸ "
		}
		l.add(mark, syntax.Comment)
		if e.icons {
			l.add(string(g.Icon)+" ", syntax.Keyword)
		}
		l.add(g.Name, syntax.Keyword)
		l.add(fmt.Sprintf(" (%d)", len(names)), syntax.Comment)
		add(l, sideEntry{Kind: sideGroup, Name: g.Name})
		if st.folded[g.Name] {
			continue
		}
		if len(names) == 0 {
			dim("      a to add")
		}
		for _, n := range names {
			l := &lineBuilder{}
			l.add("    ", syntax.Plain)
			switch g.Kind {
			case "":
				e.icon(l, icons.For(n, icons.File))
				l.add(filepath.Base(n), syntax.Plain)
				l.add("  "+e.diredShort(filepath.Dir(n)), syntax.Comment)
				add(l, sideEntry{Kind: sideFile, Path: n})
			case memory.KindDir:
				e.icon(l, icons.For(n, icons.Dir))
				l.add(n+"/", syntax.Function)
				l.add("  "+e.diredShort(e.bookmarks[n]), syntax.Comment)
				add(l, sideEntry{Kind: sideDirFav, Path: e.bookmarks[n], Name: n, Dir: true})
			case memory.KindLink:
				if e.icons {
					l.add(" ", syntax.String)
				}
				l.add(n, syntax.Plain)
				add(l, sideEntry{Kind: sideLink, Path: e.mem.Links[n], Name: n})
			case memory.KindTool:
				if e.icons {
					l.add(" ", syntax.Number)
				}
				l.add(n, syntax.Plain)
				add(l, sideEntry{Kind: sideTool, Path: e.mem.Tools[n], Name: n})
			}
		}
	}
}

// sortedKeys lists a map's keys in order.
func sortedNames(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sidebarBuffers lists the open buffers, a dot before each that has unsaved
// changes and the one in front picked out.
func (e *Editor) sidebarBuffers(add func(*lineBuilder, sideEntry), dim func(string)) {
	front := e.sidebarWindow().Buf
	for _, b := range e.tabs {
		if b == e.sideBuf {
			continue
		}
		name := e.BufferName(b)
		l := &lineBuilder{}
		if b.Modified() {
			l.add("● ", syntax.Number)
		} else {
			l.add("  ", syntax.Plain)
		}
		ic := icons.ForBuffer(name)
		e.icon(l, ic)
		class := syntax.Plain
		if b == front {
			class = syntax.Keyword
		}
		l.add(name, class)
		add(l, sideEntry{Kind: sideBuffer, Buf: b})
	}
}

// sidebarOutline lists the definitions of the file in front, from imenu.
func (e *Editor) sidebarOutline(add func(*lineBuilder, sideEntry), dim func(string)) {
	b := e.sidebarWindow().Buf
	if b == nil || b == e.sideBuf {
		dim("  (no file)")
		return
	}
	l := &lineBuilder{}
	e.icon(l, icons.ForBuffer(e.BufferName(b)))
	l.add(e.BufferName(b), syntax.Keyword)
	add(l, sideEntry{Kind: sideLabel})
	lines := make([]string, b.NumLines())
	for i := range lines {
		lines[i] = b.Line(i).String()
	}
	entries := imenu.For(b.Path(), lines)
	if len(entries) == 0 {
		dim("  (no definitions)")
		return
	}
	for _, en := range entries {
		l := &lineBuilder{}
		l.add("  "+en.Name, syntax.Plain)
		l.add(fmt.Sprintf(" :%d", en.Line+1), syntax.Comment)
		add(l, sideEntry{Kind: sideSymbol, Buf: b, Line: en.Line})
	}
}

// sidebarSync redraws the tabs that follow the editor - the buffers and the
// outline - when what they show has changed. It runs before each frame is
// built, and is quick when nothing has.
func (e *Editor) sidebarSync() {
	if !e.sideOpen() || e.sidebar == nil {
		return
	}
	st := e.sidebar
	var key string
	switch st.tab {
	case sideBuffers:
		var sb strings.Builder
		fmt.Fprintf(&sb, "%p", e.sidebarWindow().Buf)
		for _, b := range e.tabs {
			fmt.Fprintf(&sb, "|%p%s%v", b, e.BufferName(b), b.Modified())
		}
		key = sb.String()
	case sideOutline:
		b := e.sidebarWindow().Buf
		key = fmt.Sprintf("%p:%d", b, b.Revision())
	default:
		return
	}
	if key == st.key {
		return
	}
	st.key = key
	line := e.side.Pt.Line
	if err := e.sidebarRender(); err == nil {
		e.side.Pt = e.sideBuf.ClampPos(text.Pos{Line: line})
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
		if cur.Kind != sideLabel && ent == cur {
			line = i
			break
		}
	}
	e.active.Pt = e.sideBuf.ClampPos(text.Pos{Line: line})
	return nil
}

// sidebarShowTab switches to tab and puts point at the top of its lines.
func (e *Editor) sidebarShowTab(tab int) error {
	if _, err := e.sidebarAtPoint(); err != nil {
		return err
	}
	e.sidebar.tab = (tab%sideTabs + sideTabs) % sideTabs
	e.sidebar.key = ""
	if err := e.sidebarRender(); err != nil {
		return err
	}
	e.sideBuf.SetModified(false)
	e.active.Pt = e.sideBuf.ClampPos(text.Pos{Line: sideHeaderLines})
	return nil
}

// sidebarOpen is RET: a directory in the tree, or a favourites group, opens or
// closes; a file, a buffer or a definition is shown in the window the panel was
// entered from; a web address goes to the browser and a tool runs.
func (e *Editor) sidebarOpen() error {
	ent, err := e.sidebarAtPoint()
	if err != nil {
		return err
	}
	st := e.sidebar
	switch ent.Kind {
	case sideNode:
		if ent.Dir && ent.Path != st.root {
			st.expanded[ent.Path] = !st.expanded[ent.Path]
			return e.sidebarRefresh()
		}
		if ent.Dir {
			return nil // the root line is only a label
		}
		return e.sidebarVisit(ent.Path, false)
	case sideGroup:
		st.folded[ent.Name] = !st.folded[ent.Name]
		return e.sidebarRefresh()
	case sideFile:
		return e.sidebarVisit(ent.Path, false)
	case sideDirFav:
		return e.sidebarVisit(ent.Path, true)
	case sideLink:
		return e.launch(ent.Path)
	case sideTool:
		dir := e.sidebarRoot()
		return e.runCompile(ent.Path, dir, false)
	case sideBuffer:
		e.active = e.sidebarWindow()
		e.active.Visit(ent.Buf)
	case sideSymbol:
		w := e.sidebarWindow()
		e.active = w
		if w.Buf != ent.Buf {
			w.Visit(ent.Buf)
		}
		b := w.Buf
		b.SetMark(w.Pt)
		b.DeactivateMark()
		rs := b.Line(ent.Line).View()
		col := 0
		for col < len(rs) && (rs[col] == ' ' || rs[col] == '\t') {
			col++
		}
		w.Pt = text.Pos{Line: ent.Line, Col: text.RuneIdx(col)}
		w.GoalCol = view.GoalColUnset
		w.SetTop(max(0, ent.Line-e.TextHeight()/3))
	}
	return nil
}

// sidebarVisit shows a file or a directory in the window the panel was
// entered from, and moves there.
func (e *Editor) sidebarVisit(path string, dir bool) error {
	target := e.sidebarWindow()
	var b *text.Buffer
	var err error
	if dir {
		b, err = e.Dired(path)
	} else {
		b, err = e.OpenFile(path)
	}
	if err != nil {
		return err
	}
	e.active = target
	if dir {
		e.active.Visit(b)
	} else {
		e.VisitOpened(b)
	}
	return nil
}

// sidebarClick is a click on the panel at pos: a tab on the first line is
// shown, and a line below acts as RET does.
func (e *Editor) sidebarClick(pos text.Pos) {
	e.active = e.side
	e.side.Pt = e.sideBuf.ClampPos(pos)
	if pos.Line == 0 {
		for i, c := range e.sidebar.tabCols {
			if int(pos.Col) >= c[0] && int(pos.Col) < c[1] {
				if err := e.sidebarShowTab(i); err != nil {
					e.Echo("%v", err)
				}
				return
			}
		}
		return
	}
	if err := e.sidebarOpen(); err != nil {
		e.Echo("%v", err)
	}
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

// sidebarAdd is a on the Favorites tab: it asks what to add - a file, a
// directory, a web address or a tool - and then for the rest.
func (e *Editor) sidebarAdd() error {
	if _, err := e.sidebarAtPoint(); err != nil {
		return err
	}
	c, err := e.ReadChar("Add favourite: f)ile d)irectory w)eb link t)ool ", []rune{'f', 'd', 'w', 't'})
	if err != nil {
		return err
	}
	switch c {
	case 'f', 'd':
		prompt := "File: "
		if c == 'd' {
			prompt = "Directory: "
		}
		ans, err := e.ReadString(command.ReadOpts{
			Prompt: prompt, History: "directory", Initial: promptDir(e.sidebarRoot()),
			Complete: command.DirectoryCompleter(), Descend: command.IsDirCandidate,
			Icon: icons.ForCandidate, Rewrite: command.RestartPath,
		})
		if err != nil || ans == "" {
			return err
		}
		path, err := filepath.Abs(command.ExpandPath(ans))
		if err != nil {
			return err
		}
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if c == 'd' {
			if !fi.IsDir() {
				return fmt.Errorf("%s is not a directory", ans)
			}
			e.setDirBookmark(filepath.Base(path), path)
		} else {
			if fi.IsDir() {
				return fmt.Errorf("%s is a directory", ans)
			}
			e.mem.AddBookmark(path)
			e.saveMemory()
		}
	case 'w', 't':
		kind, what, valueLabel := memory.KindLink, "link", "Address: "
		if c == 't' {
			kind, what, valueLabel = memory.KindTool, "tool", "Command: "
		}
		name, err := e.ReadString(command.ReadOpts{Prompt: "Name of the " + what + ": ", History: "favourite"})
		if err != nil || name == "" {
			return err
		}
		value, err := e.ReadString(command.ReadOpts{Prompt: valueLabel, History: "favourite"})
		if err != nil || value == "" {
			return err
		}
		e.mem.SetNamed(kind, name, value)
		e.saveMemory()
	}
	e.sidebar.tab = sideFavorites
	e.sidebar.key = ""
	return e.sidebarRefresh()
}

// sidebarRemove is d on the Favorites and Buffers tabs: the favourite at
// point is dropped, or the buffer is killed.
func (e *Editor) sidebarRemove() error {
	ent, err := e.sidebarAtPoint()
	if err != nil {
		return err
	}
	switch ent.Kind {
	case sideFile:
		e.mem.RemoveBookmark(ent.Path)
		e.saveMemory()
	case sideDirFav:
		e.deleteDirBookmark(ent.Name)
	case sideLink:
		e.mem.RemoveNamed(memory.KindLink, ent.Name)
		e.saveMemory()
	case sideTool:
		e.mem.RemoveNamed(memory.KindTool, ent.Name)
		e.saveMemory()
	case sideBuffer:
		if ent.Buf.Modified() && ent.Buf.Path() != "" {
			e.Echo("%s has unsaved changes; kill it with C-x k", e.BufferName(ent.Buf))
			return nil
		}
		if len(e.buffers) <= 1 {
			e.Echo("cannot kill the last buffer")
			return nil
		}
		if err := e.KillBuffer(ent.Buf); err != nil {
			return err
		}
	default:
		return nil
	}
	return e.sidebarRefresh()
}

// registerSidebarCommands adds the panel's commands and bookmark-file.
func registerSidebarCommands(e *Editor, reg *command.Registry) error {
	panel := func(fn func() error) func(command.Env) error {
		return func(command.Env) error { return fn() }
	}
	tab := func(i int) func(command.Env) error {
		return func(command.Env) error { return e.sidebarShowTab(i) }
	}
	cmds := []command.Command{
		{Name: "sidebar-toggle", Doc: "Open the side panel, with the directory tree, favourites, open buffers and outline, and move into it; in the panel, close it.",
			Fn: e.sidebarToggle},
		{Name: "sidebar-next-tab", Doc: "Show the side panel's next tab.",
			Fn: panel(func() error { return e.sidebarShowTab(e.sidebar.tab + 1) })},
		{Name: "sidebar-previous-tab", Doc: "Show the side panel's previous tab.",
			Fn: panel(func() error { return e.sidebarShowTab(e.sidebar.tab - 1) })},
		{Name: "sidebar-tree", Doc: "Show the side panel's Tree tab.", Fn: tab(sideTree)},
		{Name: "sidebar-favorites", Doc: "Show the side panel's Favorites tab.", Fn: tab(sideFavorites)},
		{Name: "sidebar-buffers", Doc: "Show the side panel's Buffers tab.", Fn: tab(sideBuffers)},
		{Name: "sidebar-outline", Doc: "Show the side panel's Outline tab.", Fn: tab(sideOutline)},
		{Name: "sidebar-open", Doc: "Open the entry at point: fold or unfold a directory or group, show a file, buffer or definition in the other window, open a web link, run a tool.",
			Fn: panel(e.sidebarOpen)},
		{Name: "sidebar-add", Doc: "Add a favourite: a file, a directory, a web link or a tool.",
			Fn: panel(e.sidebarAdd)},
		{Name: "sidebar-bookmark", Doc: "Add the file or directory at point in the side panel's tree to the favourites; again to remove it.",
			Fn: panel(func() error {
				ent, err := e.sidebarAtPoint()
				if err != nil || ent.Kind != sideNode {
					return err
				}
				if ent.Dir {
					name := filepath.Base(ent.Path)
					if _, ok := e.bookmarks[name]; ok {
						e.deleteDirBookmark(name)
						e.Echo("Favourite removed: %s", name)
					} else {
						e.setDirBookmark(name, ent.Path)
						e.Echo("Added %s to the favourites", name)
					}
				} else {
					e.toggleFileBookmark(ent.Path)
				}
				return nil
			})},
		{Name: "sidebar-remove", Doc: "Remove the favourite at point in the side panel, or kill the buffer there.",
			Fn: panel(e.sidebarRemove)},
		{Name: "sidebar-set-root", Doc: "Make the directory at point the root of the side panel's tree.",
			Fn: panel(func() error {
				ent, err := e.sidebarAtPoint()
				if err != nil || ent.Kind != sideNode || !ent.Dir || e.sidebar.tab != sideTree {
					return err
				}
				e.sidebar.root = ent.Path
				return e.sidebarShowTab(sideTree)
			})},
		{Name: "sidebar-parent", Doc: "Make the parent of the tree's root the root.",
			Fn: panel(func() error {
				if _, err := e.sidebarAtPoint(); err != nil {
					return err
				}
				e.sidebar.root = filepath.Dir(e.sidebar.root)
				return e.sidebarShowTab(sideTree)
			})},
		{Name: "sidebar-toggle-hidden", Doc: "Show or hide dotfiles in the side panel's tree.",
			Fn: panel(func() error {
				if _, err := e.sidebarAtPoint(); err != nil {
					return err
				}
				e.sidebar.showHidden = !e.sidebar.showHidden
				return e.sidebarRefresh()
			})},
		{Name: "sidebar-refresh", Doc: "Read the side panel's directories and favourites again.",
			Fn: panel(func() error {
				if _, err := e.sidebarAtPoint(); err != nil {
					return err
				}
				return e.sidebarRefresh()
			})},
		{Name: "bookmark-file", Doc: "Add this buffer's file to the favourites, shown in the side panel's Favorites tab; again to remove it.",
			Fn: panel(func() error {
				p := e.active.Buf.Path()
				if p == "" || e.diredOf(e.active.Buf) != nil {
					return errors.New("this buffer has no file to bookmark")
				}
				e.toggleFileBookmark(p)
				if e.sideOpen() && e.sidebar.tab == sideFavorites {
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

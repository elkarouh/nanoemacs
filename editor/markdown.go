package editor

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Borderliner/nem/ansistyle"
	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/keymap"
	"github.com/Borderliner/nem/text"
	"github.com/Borderliner/nem/ui"
	"github.com/gdamore/tcell/v2"
)

// The Markdown preview takes the place of the buffer it renders: in that
// buffer's tab, and in every window that showed it. q puts the buffer back.
// The rendering is held apart from the buffer list, so it never gets a tab of
// its own. mdcat is the external tool: it does the Markdown, and its colour codes
// are read by the ansistyle package and drawn as styled spans. The rendering is
// refreshed on each save of its source.

// markdownTimeout bounds one mdcat run.
const markdownTimeout = 30 * time.Second

// previewWidth is the column width mdcat wraps the preview to.
const previewWidth = 80

// previewState is one source buffer's preview: the rendering, its styled spans
// by line, and whether it is showing in the source's place.
type previewState struct {
	source *text.Buffer
	buf    *text.Buffer
	lines  map[int][]ui.SymbolSpan
	shown  bool
	render renderer
}

// renderer turns a source's text into the text its preview shows, with the
// colours it wants as ANSI codes. mdcat and lynx are both renderers.
type renderer func(src string) ([]byte, error)

// previewBindings are a preview's keys: the viewer's. Nothing in a preview is
// typed into, so these are its commands, and a letter with none does nothing
// but say the buffer is read-only. Every other key falls through to the global
// map, so C-s still searches.
var previewBindings = []struct{ Spec, Command string }{
	{"q", "preview-close"},
	{"SPC", "scroll-up-command"},
	{"b", "scroll-down-command"},
	{"<", "beginning-of-buffer"},
	{">", "end-of-buffer"},
	{"/", "isearch-forward"},
}

// newPreviewKeymap builds the preview keymap from previewBindings.
func newPreviewKeymap() (*keymap.Map, error) {
	m := keymap.New()
	for _, b := range previewBindings {
		if err := bindSpec(m, b.Spec, b.Command); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// registerMarkdownCommands adds the preview commands.
func registerMarkdownCommands(e *Editor, reg *command.Registry) error {
	cmds := []command.Command{
		{Name: "markdown-preview", Doc: "Show the buffer rendered as Markdown, in its place, as mdcat renders it. The preview is updated on each save of the buffer; q shows the buffer again.",
			Fn: func(command.Env) error { return e.showRendered(e.active.Buf, mdcatRender) }},
		{Name: "html-preview", Doc: "Show the buffer rendered as HTML text, in its place, as lynx lays it out. An .html file opens this way; the preview is updated on each save of the buffer; q shows the buffer again.",
			Fn: func(command.Env) error { return e.showRendered(e.active.Buf, htmlRender) }},
		{Name: "preview-close", Doc: "Close the preview and show the buffer it renders in its place.",
			Fn: func(command.Env) error { return e.previewClose() }},
	}
	for _, c := range cmds {
		c.Interactive = true
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// previewRenderer is the renderer that a file opens its preview with, by its
// extension, or nil for a file that opens as its text.
func previewRenderer(path string) renderer {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md":
		return mdcatRender
	case ".html", ".htm":
		return htmlRender
	}
	return nil
}

// VisitOpened shows b, a buffer just opened from a file, in the active window.
// A .md file is shown as its Markdown preview instead, and an .html or .htm
// file as its HTML preview; q turns the preview into the editable buffer. Without
// the renderer the file is shown as it is, since the preview is a convenience
// and not something to fail an open over.
func (e *Editor) VisitOpened(b *text.Buffer) {
	e.active.Visit(b)
	render := previewRenderer(b.Path())
	if render == nil {
		return
	}
	st, err := e.renderWith(b, render)
	if err != nil {
		return
	}
	if !st.shown {
		e.showPreview(st)
	}
	e.active.Visit(st.buf)
}

// showRendered shows src rendered by render, in its place. A preview is not
// previewed again.
func (e *Editor) showRendered(src *text.Buffer, render renderer) error {
	if e.previewOf(src) != nil {
		return nil
	}
	st, err := e.renderWith(src, render)
	if err != nil {
		return err
	}
	if !st.shown {
		e.showPreview(st)
	}
	return nil
}

// previewClose shows the buffer again in place of the preview in the active
// window.
func (e *Editor) previewClose() error {
	st := e.previewOf(e.active.Buf)
	if st == nil {
		return fmt.Errorf("this buffer is not a preview")
	}
	e.hidePreview(st)
	return nil
}

// renderWith renders src's text with render into its preview, creating the
// preview on first use.
func (e *Editor) renderWith(src *text.Buffer, render renderer) (*previewState, error) {
	out, err := render(string(src.Text(text.Pos{}, src.End())))
	if err != nil {
		return nil, err
	}

	st := e.previews[src]
	if st == nil {
		st = &previewState{source: src, buf: text.NewBuffer()}
		e.previews[src] = st
	}
	st.render = render
	plain, spans := ansistyle.Parse(string(out))
	st.buf.SetReadOnly(false)
	if st.buf.End() != (text.Pos{}) {
		if err := st.buf.Delete(text.Pos{}, st.buf.End()); err != nil {
			st.buf.SetReadOnly(true)
			return nil, err
		}
	}
	if err := st.buf.Insert(text.Pos{}, []rune(plain)); err != nil {
		st.buf.SetReadOnly(true)
		return nil, err
	}
	// The rendering is what the source's text came to, not an edit to keep:
	// quitting must not ask about it.
	st.buf.SetModified(false)
	st.buf.SetReadOnly(true)
	st.lines = previewLines(plain, spans)
	return st, nil
}

// showPreview puts the preview in its source's place.
func (e *Editor) showPreview(st *previewState) {
	e.names[st.buf] = e.names[st.source]
	e.replaceBuffer(st.source, st.buf)
	st.shown = true
}

// hidePreview puts the source back in the preview's place.
func (e *Editor) hidePreview(st *previewState) {
	e.replaceBuffer(st.buf, st.source)
	st.shown = false
}

// replaceBuffer puts neu where old is in the buffer lists, and shows neu in
// every window that showed old.
func (e *Editor) replaceBuffer(old, neu *text.Buffer) {
	for i, b := range e.buffers {
		if b == old {
			e.buffers[i] = neu
		}
	}
	for i, b := range e.tabs {
		if b == old {
			e.tabs[i] = neu
		}
	}
	for _, w := range e.tree.Windows() {
		if w.Buf == old {
			w.Visit(neu)
		}
	}
}

// previewOf is the preview state of b when b is a preview's rendering, or nil.
func (e *Editor) previewOf(b *text.Buffer) *previewState {
	for _, st := range e.previews {
		if st.buf == b {
			return st
		}
	}
	return nil
}

// refreshPreviews re-renders src's preview, after src is saved.
func (e *Editor) refreshPreviews(src *text.Buffer) {
	if e.previews[src] == nil {
		return
	}
	if _, err := e.renderWith(src, e.previews[src].render); err != nil {
		e.Echo("Preview not updated: %s", err.Error())
	}
}

// previewLines splits styled spans, which run over the whole text by rune
// offset, into the spans of each line. A span over a line break becomes one
// span on each line it touches.
func previewLines(plain string, spans []ansistyle.Span) map[int][]ui.SymbolSpan {
	out := map[int][]ui.SymbolSpan{}
	rs := []rune(plain)
	starts := []int{0} // the rune offset where each line begins
	for i, r := range rs {
		if r == '\n' {
			starts = append(starts, i+1)
		}
	}
	for _, sp := range spans {
		for line, ls := range starts {
			le := len(rs)
			if line+1 < len(starts) {
				le = starts[line+1] - 1 // the line's own break is not styled
			}
			lo, hi := max(sp.Start, ls), min(sp.End, le)
			if lo >= hi {
				continue
			}
			out[line] = append(out[line], ui.SymbolSpan{
				Start:  text.RuneIdx(lo - ls),
				End:    text.RuneIdx(hi - ls),
				Styled: true,
				Style:  tcellStyle(sp.Style),
			})
		}
	}
	return out
}

// mdcatRender renders Markdown with mdcat, in colour.
func mdcatRender(src string) ([]byte, error) {
	mdcat, err := exec.LookPath("mdcat")
	if err != nil {
		return nil, fmt.Errorf("the Markdown preview needs mdcat, which is not on the PATH")
	}
	return runRenderer(mdcat, []string{"--ansi", "--theme", "dark", "--local",
		"--no-pager", "--columns", fmt.Sprint(previewWidth), "-"}, src)
}

// htmlRender lays HTML out as plain text, with lynx, which prints the page
// without colour codes.
func htmlRender(src string) ([]byte, error) {
	lynx, err := exec.LookPath("lynx")
	if err != nil {
		return nil, fmt.Errorf("the HTML preview needs lynx, which is not on the PATH")
	}
	return runRenderer(lynx, []string{"-stdin", "-dump",
		fmt.Sprintf("-width=%d", previewWidth)}, src)
}

// runRenderer runs tool with args, src on its stdin, and returns what it
// prints. A tool that fails is reported by its first line of stderr.
func runRenderer(tool string, args []string, src string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), markdownTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, tool, args...)
	cmd.Stdin = strings.NewReader(src)
	out, err := cmd.Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			msg = vcFirstLine(string(ee.Stderr), err)
		}
		return nil, fmt.Errorf("%s: %s", filepath.Base(tool), msg)
	}
	return out, nil
}

// tcellStyle is an ansistyle style as the screen draws it.
func tcellStyle(s ansistyle.Style) tcell.Style {
	st := tcell.StyleDefault.Foreground(tcellColor(s.FG)).Background(tcellColor(s.BG))
	return st.Bold(s.Bold).Italic(s.Italic).Underline(s.Underline)
}

// tcellColor is an ansistyle colour as tcell has it.
func tcellColor(c ansistyle.Color) tcell.Color {
	if idx, ok := c.PaletteIndex(); ok {
		return tcell.PaletteColor(idx)
	}
	if r, g, b, ok := c.IsRGB(); ok {
		return tcell.NewRGBColor(int32(r), int32(g), int32(b))
	}
	return tcell.ColorDefault
}

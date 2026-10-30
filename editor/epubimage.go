package editor

import (
	"strings"

	"github.com/Borderliner/nem/ui"
	"github.com/Borderliner/nem/view"
)

// The pictures of an EPUB preview are drawn by the terminal over the blank rows
// reserved for them, after each redraw, so they move with the text as it
// scrolls. A picture is drawn only where its whole box is in the window's text,
// unwrapped and unscrolled sideways, since that is where its rows are. None is
// drawn while a panel is open: the terminal draws pictures over text, and would
// cover the panel.

// pictureSequence returns the escape sequence that brings the screen's pictures
// up to date: it clears the pictures drawn last, then draws those of the
// previews on screen. w and h are the frame's size less the echo row. kitty
// reports whether the terminal draws kitty graphics; nothing is drawn without
// it, and nothing is drawn while panels cover the text.
func (e *Editor) pictureSequence(w, h int, panels, kitty bool) string {
	var sb strings.Builder
	if e.picturesDrawn {
		sb.WriteString(ui.KittyClearPlacements)
		e.picturesDrawn = false
	}
	if panels || !kitty {
		return sb.String()
	}
	for win, rect := range e.tree.Layout(w, h) {
		st := e.previewOf(win.Buf)
		if st == nil {
			continue
		}
		textH := view.TextHeight(rect)
		textW := ui.TextWidth(rect, win, false, e.th)
		if e.th.Wrap || win.LeftCol != 0 || textW <= 0 {
			continue
		}
		x := rect.X + rect.W - textW
		for _, p := range st.pictures {
			if p.Rows == 0 || p.Cols > textW || p.Line < win.Top || p.Line+p.Rows > win.Top+textH {
				continue
			}
			if !p.sent {
				sb.WriteString(ui.KittyTransmit(p.png, p.id))
				p.sent = true
			}
			sb.WriteString(ui.KittyPlace(p.id, x, rect.Y+p.Line-win.Top, p.Cols, p.Rows))
			e.picturesDrawn = true
		}
	}
	return sb.String()
}

// forgetPictures notes that the terminal has dropped every picture, as it does
// when the images are cleared, so each is sent again before it is drawn.
func (e *Editor) forgetPictures() {
	for _, st := range e.previews {
		for _, p := range st.pictures {
			p.sent = false
		}
	}
}

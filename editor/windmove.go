package editor

import (
	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/view"
)

// Shift-left and shift-right select the window beside the current one, as
// emacs's windmove does. The side panel is a window like any other, so these
// are also the way in and out of it.

// windowBeside is the window next to the active one on the left or right, or
// nil when there is none. Of the windows on that side that overlap it in rows,
// the nearest wins, and then the one whose top is closest.
func (e *Editor) windowBeside(right bool) *view.Window {
	sw, sh := e.scr.Size()
	rects := e.tree.Layout(sw, sh)
	cur, ok := rects[e.active]
	if !ok {
		return nil
	}
	var best *view.Window
	var bestGap, bestDY int
	for w, r := range rects {
		if w == e.active || r.Y >= cur.Y+cur.H || cur.Y >= r.Y+r.H {
			continue // no rows in common
		}
		var gap int
		if right {
			gap = r.X - (cur.X + cur.W)
		} else {
			gap = cur.X - (r.X + r.W)
		}
		if gap < 0 {
			continue // not on that side
		}
		dy := r.Y - cur.Y
		if dy < 0 {
			dy = -dy
		}
		if best == nil || gap < bestGap || (gap == bestGap && dy < bestDY) {
			best, bestGap, bestDY = w, gap, dy
		}
	}
	return best
}

// windmove selects the window beside the active one.
func (e *Editor) windmove(right bool) error {
	w := e.windowBeside(right)
	if w == nil {
		if right {
			e.Echo("No window to the right")
		} else {
			e.Echo("No window to the left")
		}
		return nil
	}
	e.active = w
	// What the panel opens is shown in the window last used besides it.
	if w != e.side {
		e.sidePrev = w
	}
	return nil
}

// registerWindmoveCommands adds windmove-left and windmove-right.
func registerWindmoveCommands(e *Editor, reg *command.Registry) error {
	for _, c := range []command.Command{
		{Name: "windmove-left", Doc: "Select the window to the left of this one.",
			Interactive: true, Fn: func(command.Env) error { return e.windmove(false) }},
		{Name: "windmove-right", Doc: "Select the window to the right of this one.",
			Interactive: true, Fn: func(command.Env) error { return e.windmove(true) }},
	} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

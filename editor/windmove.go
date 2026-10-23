package editor

import (
	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/view"
)

// Shift-arrow selects the window beside the current one in that direction, as
// emacs's windmove does. The side panel is a window like any other, so these
// are also the way in and out of it.

type direction int

const (
	dirLeft direction = iota
	dirRight
	dirUp
	dirDown
)

var directionName = map[direction]string{
	dirLeft: "left", dirRight: "right", dirUp: "above", dirDown: "below",
}

// windowBeside is the window next to the active one in direction d, or nil
// when there is none. Of the windows on that side that overlap it across the
// other axis, the nearest wins, and then the one whose edge is closest.
func (e *Editor) windowBeside(d direction) *view.Window {
	sw, sh := e.scr.Size()
	rects := e.tree.Layout(sw, sh)
	cur, ok := rects[e.active]
	if !ok {
		return nil
	}
	var best *view.Window
	var bestGap, bestOff int
	for w, r := range rects {
		if w == e.active {
			continue
		}
		var gap, off int
		switch d {
		case dirLeft, dirRight:
			if r.Y >= cur.Y+cur.H || cur.Y >= r.Y+r.H {
				continue // no rows in common
			}
			if d == dirRight {
				gap = r.X - (cur.X + cur.W)
			} else {
				gap = cur.X - (r.X + r.W)
			}
			off = r.Y - cur.Y
		default:
			if r.X >= cur.X+cur.W || cur.X >= r.X+r.W {
				continue // no columns in common
			}
			if d == dirDown {
				gap = r.Y - (cur.Y + cur.H)
			} else {
				gap = cur.Y - (r.Y + r.H)
			}
			off = r.X - cur.X
		}
		if gap < 0 {
			continue // not on that side
		}
		if off < 0 {
			off = -off
		}
		if best == nil || gap < bestGap || (gap == bestGap && off < bestOff) {
			best, bestGap, bestOff = w, gap, off
		}
	}
	return best
}

// windmove selects the window beside the active one.
func (e *Editor) windmove(d direction) error {
	w := e.windowBeside(d)
	if w == nil {
		if d == dirUp || d == dirDown {
			e.Echo("No window %s", directionName[d])
		} else {
			e.Echo("No window to the %s", directionName[d])
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

// registerWindmoveCommands adds windmove-left, -right, -up and -down.
func registerWindmoveCommands(e *Editor, reg *command.Registry) error {
	for _, c := range []struct {
		name string
		d    direction
	}{
		{"windmove-left", dirLeft}, {"windmove-right", dirRight},
		{"windmove-up", dirUp}, {"windmove-down", dirDown},
	} {
		d := c.d
		if err := reg.Register(command.Command{
			Name: c.name, Doc: "Select the window " + map[direction]string{
				dirLeft: "to the left of", dirRight: "to the right of", dirUp: "above", dirDown: "below",
			}[d] + " this one.",
			Interactive: true, Fn: func(command.Env) error { return e.windmove(d) },
		}); err != nil {
			return err
		}
	}
	return nil
}

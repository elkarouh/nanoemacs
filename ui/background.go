package ui

import (
	"strconv"
	"strings"
)

// detectedLight holds what the terminal itself said its background is, once
// DetectBackground has asked. nil means it was never asked or did not answer,
// and TerminalIsLight then falls back to COLORFGBG.
var detectedLight *bool

// parseOSC11 reads a terminal's reply to the OSC 11 background query,
// "ESC ] 11 ; rgb:RRRR/GGGG/BBBB" ended by BEL or ST, and reports whether that
// colour is light. Each channel is one to four hex digits; xterm sends four,
// others two. ok is false for any reply it cannot read.
func parseOSC11(reply string) (light, ok bool) {
	i := strings.Index(reply, "rgb:")
	if i < 0 {
		return false, false
	}
	var ch [3]float64
	parts := strings.SplitN(reply[i+len("rgb:"):], "/", 3)
	if len(parts) != 3 {
		return false, false
	}
	for n, s := range parts {
		end := 0
		for end < len(s) && end < 4 && isHex(s[end]) {
			end++
		}
		if end == 0 {
			return false, false
		}
		v, err := strconv.ParseUint(s[:end], 16, 32)
		if err != nil {
			return false, false
		}
		ch[n] = float64(v) / float64(uint64(1)<<(4*uint(end))-1)
	}
	// Perceived brightness, Rec. 601 weights. Half is the line: the dark
	// palette is faint on mid grey and the light one is too, so a tie hardly
	// matters.
	return 0.299*ch[0]+0.587*ch[1]+0.114*ch[2] > 0.5, true
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

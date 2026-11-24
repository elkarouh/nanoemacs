package editor

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

// TEMPORARY debugging: writes each raw key event to the file named by
// NEM_KEYLOG. Remove before merging.
func logKeyEvent(ev *tcell.EventKey, k fmt.Stringer) {
	path := os.Getenv("NEM_KEYLOG")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "key=%d rune=%q mod=%d decoded=%v\n", ev.Key(), ev.Rune(), ev.Modifiers(), k)
}

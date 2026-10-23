package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/ui"
	"github.com/gdamore/tcell/v2"
)

// registerImageCommands adds image-view. It is an experiment in the kitty
// graphics protocol: in a terminal that has it, an image shows as pixels;
// elsewhere the command says so and does nothing.
func registerImageCommands(e *Editor, reg *command.Registry) error {
	return reg.Register(command.Command{
		Name:        "image-view",
		Doc:         "Show the active buffer's file, a PNG, JPEG or GIF, as an image over the window, in terminals that draw kitty graphics (kitty, Ghostty, WezTerm). Any key closes it.",
		Interactive: true,
		Fn:          func(command.Env) error { return e.imageView() },
	})
}

// isImageFile reports whether path names a picture the image viewer can show.
func isImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	}
	return false
}

// canShowImages reports whether opening an image file shows it, rather than
// refusing it as the binary file it is.
func (e *Editor) canShowImages() bool { return e.scr != nil && ui.KittySupported() }

func (e *Editor) imageView() error {
	if !e.canShowImages() {
		return fmt.Errorf("this terminal does not show images (needs kitty, Ghostty or WezTerm outside tmux)")
	}
	path := e.active.Buf.Path()
	if path == "" {
		return fmt.Errorf("this buffer has no file")
	}
	return e.showImage(path)
}

// showImage draws the image at path over the frame and waits for a key.
func (e *Editor) showImage(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	png, iw, ih, err := ui.ImagePNG(data)
	if err != nil {
		return err
	}
	e.Redraw()
	w, h := e.scr.Size()
	cw, ch := ui.CellPixels(e.scr)
	cols, rows := ui.FitImage(iw, ih, w, h-1, cw, ch)
	if cols == 0 {
		return fmt.Errorf("no room for the image")
	}
	if !ui.WriteTTY(e.scr, ui.KittyDraw(png, (w-cols)/2, (h-1-rows)/2, cols, rows)) {
		return fmt.Errorf("could not reach the terminal")
	}
	e.Echo("%s (%dx%d) - any key closes", path, iw, ih)
	e.Redraw()
	for {
		ev := e.nextEvent()
		if _, isKey := ev.(*tcell.EventKey); isKey || ev == nil {
			break
		}
	}
	ui.WriteTTY(e.scr, ui.KittyClear)
	e.Echo("")
	e.scr.Sync()
	return nil
}

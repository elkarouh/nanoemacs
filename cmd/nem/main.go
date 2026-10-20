// Command nem is a terminal text editor with nano's shape and emacs's
// keybindings.
//
// Usage:
//
//	nem [[+LINE[:COLUMN]] file...]
//
// Each named file is opened into a buffer; the first is shown. A +LINE
// argument, as in emacs, opens the file after it at that line (and column).
// With no arguments nem runs yazi on the current directory, falling back to
// dired on it if yazi opens nothing.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Borderliner/nem/backup"
	"github.com/Borderliner/nem/command"
	"github.com/Borderliner/nem/editor"
	"github.com/Borderliner/nem/ui"
)

// version is stamped at build time with -ldflags "-X main.version=v1.2.3".
// A local go build leaves it "dev", which is the honest answer for a binary
// that came from a working tree rather than a tagged release.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [[+LINE[:COLUMN]] file...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("nem", version)
		return
	}

	targets, err := parseTargets(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "nem:", err)
		flag.Usage()
		os.Exit(2)
	}

	if err := run(targets); err != nil {
		fmt.Fprintln(os.Stderr, "nem:", err)
		// Ended by a signal, nem exits as the signal would have ended it, so
		// a shell or a service manager reads the same status.
		var se *editor.SignalError
		if errors.As(err, &se) {
			if n, ok := se.Signal.(syscall.Signal); ok {
				os.Exit(128 + int(n))
			}
		}
		os.Exit(1)
	}
}

// rescue writes e's unsaved work away as nem crashes, and reports where. A
// fault in writing is swallowed: the crash being reported matters more.
func rescue(e *editor.Editor) (saved []string) {
	defer func() { _ = recover() }()
	saved, _ = e.EmergencySave()
	return saved
}

func run(targets []target) (err error) {
	scr, err := ui.NewScreen()
	if err != nil {
		return fmt.Errorf("opening terminal: %w", err)
	}

	// Restoring the terminal is not optional, and it must happen before the
	// panic is allowed to continue. A panic that unwinds past here with the
	// screen still in raw mode and the alternate buffer active leaves the
	// user's shell echoing nothing and rendering nowhere — the editor's bug
	// becomes the shell's problem. So: stop the panic, restore the terminal,
	// then re-panic so the trace prints onto a terminal that works.
	//
	// The editor catches its own bugs as they happen, so one reaching here
	// got past that; the unsaved work is written away before anything else.
	var e *editor.Editor
	defer func() {
		r := recover()
		var saved []string
		if r != nil && e != nil {
			saved = rescue(e)
		}
		scr.Close()
		if r != nil {
			if len(saved) > 0 {
				fmt.Fprintf(os.Stderr, "nem: unsaved work written to:\n  %s\n", strings.Join(saved, "\n  "))
			}
			panic(r)
		}
	}()

	e, err = editor.New(scr)
	if err != nil {
		return err
	}

	// The terminal closing, an SSH connection dropping or the system shutting
	// down ends the session with the unsaved work written away, rather than
	// killing nem where it stands with the work in it. Ctrl-C is a key in a
	// raw terminal, so an interrupt comes only from outside too.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sigs)
	e.SetSignals(sigs)

	// A broken config must not stop nem from starting: LoadConfig reports the
	// failure in the echo area and leaves built-in defaults in place, so the
	// error is deliberately not propagated here.
	_ = e.LoadConfig("")
	defer e.CloseConfig()

	// What the last session remembered - prompt history, recent files, where
	// point was in each file - kept beside the backups. Without a state
	// directory nem simply starts with none.
	if dir, err := backup.DefaultRoot(); err == nil {
		_ = e.UseStateDir(dir)
	}

	// With no file named, nem runs yazi on the current directory below, and
	// lists it in dired if yazi opens nothing.
	bare := len(targets) == 0

	for i, t := range targets {
		b, err := e.OpenFile(t.Path)
		if errors.Is(err, command.ErrOpenedElsewhere) {
			continue // handed to the system's app; nothing to show here
		}
		if err != nil {
			return fmt.Errorf("opening %s: %w", t.Path, err)
		}
		t.place(b)
		if i == 0 {
			e.VisitOpened(b)
		}
	}

	if bare {
		e.StartYazi(".")
	}

	err = e.Loop()
	e.SaveMemory()
	return err
}

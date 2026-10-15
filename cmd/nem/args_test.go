package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Borderliner/nem/text"
)

func TestParseTargets(t *testing.T) {
	tests := []struct {
		args []string
		want []target
	}{
		{nil, nil},
		{[]string{"a.go"}, []target{{Path: "a.go"}}},
		{[]string{"+44", "file.go"}, []target{{Path: "file.go", Line: 44}}},
		{[]string{"+44:7", "file.go"}, []target{{Path: "file.go", Line: 44, Col: 7}}},
		// The line applies to the next file only.
		{[]string{"+3", "a.go", "b.go"}, []target{{Path: "a.go", Line: 3}, {Path: "b.go"}}},
		{[]string{"a.go", "+9", "b.go"}, []target{{Path: "a.go"}, {Path: "b.go", Line: 9}}},
		// The last +LINE before a file wins.
		{[]string{"+1", "+2", "a.go"}, []target{{Path: "a.go", Line: 2}}},
		// +0 is line one, as in emacs.
		{[]string{"+0", "a.go"}, []target{{Path: "a.go", Line: 1}}},
		// Not a number, so a file name.
		{[]string{"+notes.txt"}, []target{{Path: "+notes.txt"}}},
		{[]string{"+"}, []target{{Path: "+"}}},
		{[]string{"+4:x"}, []target{{Path: "+4:x"}}},
	}
	for _, tt := range tests {
		got, err := parseTargets(tt.args)
		if err != nil {
			t.Errorf("parseTargets(%q): %v", tt.args, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseTargets(%q) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestParseTargetsDanglingLine(t *testing.T) {
	if _, err := parseTargets([]string{"a.go", "+44"}); err == nil {
		t.Fatal("a +LINE with no file after it was accepted")
	}
}

func TestPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\nfour\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		t    target
		want text.Pos
	}{
		{target{Line: 3}, text.Pos{Line: 2}},
		{target{Line: 2, Col: 3}, text.Pos{Line: 1, Col: 2}},
		{target{Line: 99}, text.Pos{Line: 3}},        // past the end clamps
		{target{Line: 1, Col: 99}, text.Pos{Col: 3}}, // past the line's end clamps
		{target{}, text.Pos{}},                       // no +LINE leaves point alone
	}
	for _, tt := range tests {
		b, err := text.LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		tt.t.place(b)
		if got := b.SavePoint(); got != tt.want {
			t.Errorf("place(%+v) put point at %+v, want %+v", tt.t, got, tt.want)
		}
	}
}

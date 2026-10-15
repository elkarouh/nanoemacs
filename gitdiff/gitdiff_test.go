package gitdiff

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseAddedChangedDeleted(t *testing.T) {
	diff := `diff --git a/f b/f
index 1..2 100644
--- a/f
+++ b/f
@@ -2 +2 @@
-old
+new
@@ -5,0 +6,2 @@
+one
+two
@@ -8,3 +8,0 @@
-gone
-gone
-gone
`
	got := Parse(diff)
	want := Marks{1: Changed, 5: Added, 6: Added, 8: Deleted}
	if len(got) != len(want) {
		t.Fatalf("Parse = %v, want %v", got, want)
	}
	for line, c := range want {
		if got[line] != c {
			t.Errorf("line %d = %v, want %v", line, got[line], c)
		}
	}
}

func TestParseDeletionAtTopIsLineZero(t *testing.T) {
	got := Parse("@@ -1,2 +0,0 @@\n")
	if got[0] != Deleted {
		t.Errorf("Parse = %v, want deletion at line 0", got)
	}
}

func TestParseChangeOutranksDeletionOnSameLine(t *testing.T) {
	// A removal after line 3 and a change to line 3 must show as a change, in
	// either order.
	for _, diff := range []string{
		"@@ -4,2 +3,0 @@\n@@ -3 +3 @@\n",
		"@@ -3 +3 @@\n@@ -4,2 +3,0 @@\n",
	} {
		if got := Parse(diff); got[2] != Changed {
			t.Errorf("Parse(%q) line 2 = %v, want Changed", diff, got[2])
		}
	}
}

func TestParseNoHunksIsEmpty(t *testing.T) {
	if got := Parse("diff --git a/f b/f\n"); len(got) != 0 {
		t.Errorf("Parse = %v, want empty", got)
	}
}

func TestRunAgainstARepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	file := filepath.Join(dir, "f.txt")
	os.WriteFile(file, []byte("a\nb\nc\n"), 0o644)
	run("add", "f.txt")
	run("commit", "-q", "-m", "init")
	os.WriteFile(file, []byte("a\nB\nc\nd\n"), 0o644)

	got, err := Run(file, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got[1] != Changed || got[3] != Added {
		t.Errorf("Run = %v, want line 1 Changed and line 3 Added", got)
	}
}

func TestRunOutsideRepositoryIsAnError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	file := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(file, []byte("x\n"), 0o644)
	if _, err := Run(file, ""); err == nil {
		t.Error("Run outside a repository: want an error")
	}
}

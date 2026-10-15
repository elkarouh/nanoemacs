// Package gitdiff says which lines of a file differ from the version git has at
// HEAD, for the gutter's change marks.
//
// This is the one place nem runs git. The branch in the modeline is read from
// .git directly, because it is needed on every frame; a gutter is needed only
// when a file is saved or opened, so one process per save is cheap, and
// computing a diff in Go would mean reading git's packed objects. A git1 file
// is diffed against its own repository by pointing git at it with GIT_DIR and
// GIT_WORK_TREE.
package gitdiff

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Change is what happened to one line.
type Change byte

const (
	None Change = iota
	// Added is a line that is new since HEAD.
	Added
	// Changed is a line whose text differs from HEAD.
	Changed
	// Deleted marks the line after which lines were removed since HEAD.
	Deleted
)

// Marks maps a 0-based line to its change. A line with no entry is unchanged.
type Marks map[int]Change

// timeout bounds the git process. A gutter that waits on a stuck git would hold
// up saving.
const timeout = 5 * time.Second

// hunk is the header of one diff hunk: -old,count +new,count.
var hunk = regexp.MustCompile(`^@@ -\d+(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// Parse reads the hunk headers of a `git diff -U0` and says which lines of the
// new file they cover. With no context lines, a hunk's header alone gives the
// lines exactly.
func Parse(diff string) Marks {
	marks := Marks{}
	sc := bufio.NewScanner(strings.NewReader(diff))
	for sc.Scan() {
		m := hunk.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		oldCount := 1
		if m[1] != "" {
			oldCount, _ = strconv.Atoi(m[1])
		}
		start, _ := strconv.Atoi(m[2])
		newCount := 1
		if m[3] != "" {
			newCount, _ = strconv.Atoi(m[3])
		}

		if newCount == 0 {
			// Lines were removed after new line start, so the mark sits on the
			// line that now follows the gap - which is index start, 0-based. A
			// removal at the very top is start 0.
			if _, ok := marks[start]; !ok {
				marks[start] = Deleted
			}
			continue
		}
		kind := Changed
		if oldCount == 0 {
			kind = Added
		}
		for i := start - 1; i < start-1+newCount; i++ {
			marks[i] = kind
		}
	}
	return marks
}

// Run diffs file against HEAD. repo, when not empty, is a git directory to use
// instead of the one enclosing file: the git1 repository that tracks it. A file
// git knows nothing of, or a missing git, is an error, and the caller shows no
// marks.
func Run(file, repo string) (Marks, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(abs)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "--no-color", "--no-ext-diff", "-U0", "HEAD", "--", filepath.Base(abs))
	cmd.Dir = dir
	if repo != "" {
		cmd.Env = append(os.Environ(), "GIT_DIR="+repo, "GIT_WORK_TREE="+dir)
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return Parse(string(out)), nil
}

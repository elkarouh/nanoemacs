package editor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Borderliner/nem/command"
)

// Version control on the file being edited, as C-x v does in emacs. The
// commands run git, as the gutter does: on a command the user asked for, never
// while drawing. A git1-tracked file is committed in its own repository.

// vcTimeout bounds one git command. A commit runs the repository's hooks, which
// can be slow, so the bound is generous; it is there so a stuck hook cannot hang
// the editor for good.
const vcTimeout = 2 * time.Minute

// registerVCCommands adds C-x v v.
func registerVCCommands(e *Editor, reg *command.Registry) error {
	cmds := []command.Command{
		{Name: "vc-next-action", Doc: "Save the file, stage it and commit it, with the message typed on the prompt. The commit holds this file alone, as C-x v v does in emacs.",
			Fn: func(command.Env) error { return e.vcCommit() }},
	}
	for _, c := range cmds {
		c.Interactive = true
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// vcCommit saves the active buffer if it has changes, asks for a message, then
// stages and commits the file. A file that no repository tracks - neither its
// git1 repository nor an enclosing git one - is first started in a git1
// repository, as `git1 init` does, so the commit has somewhere to go. Cancelling
// at the prompt changes nothing.
func (e *Editor) vcCommit() error {
	b := e.active.Buf
	p := b.Path()
	if p == "" {
		return fmt.Errorf("this buffer has no file to commit")
	}
	if b.Modified() {
		if err := e.SaveBuffer(b, ""); err != nil {
			return err
		}
	}
	msg, err := e.ReadString(command.ReadOpts{Prompt: "Commit message: ", History: "commit"})
	if err != nil {
		return err
	}
	if strings.TrimSpace(msg) == "" {
		e.Echo("Commit abandoned: no message")
		return nil
	}

	name := filepath.Base(p)
	repo, tracked := git1Repo(p) // "" when not git1-tracked
	created := false
	if !tracked {
		if _, err := vcGit(p, "", "ls-files", "--error-unmatch", "--", name); err != nil {
			// No git knows this file: start it in its own git1 repository.
			repo, err = git1Init(p)
			if err != nil {
				e.Echo("%s", err.Error())
				return nil
			}
			created = true
		}
	}

	if out, err := vcGit(p, repo, "add", "--", name); err != nil {
		e.Echo("git add failed: %s", vcFirstLine(out, err))
		return nil
	}
	if out, err := vcGit(p, repo, "commit", "-m", msg, "--", name); err != nil {
		if created {
			_ = os.RemoveAll(repo) // leave no half-made repository behind
		}
		e.Echo("Commit failed: %s", vcFirstLine(out, err))
		return nil
	}
	e.refreshGutter(b)
	if created {
		e.Echo("Tracking %s in git1, committed", name)
	} else {
		e.Echo("Committed %s", name)
	}
	return nil
}

// git1Init starts tracking the file at p in a new git1 repository beside it, as
// `git1 init` does, and returns that repository. Unlike `git1 init` it makes no
// commit: the caller commits with the message the user typed.
func git1Init(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if fi, err := os.Stat(abs); err != nil || !fi.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %s", filepath.Base(abs))
	}
	dir := filepath.Dir(abs)
	name := filepath.Base(abs)
	repo := filepath.Join(dir, git1Container, name)
	if _, err := os.Stat(repo); err == nil {
		return "", fmt.Errorf("already tracked: %s", name)
	}
	if err := os.MkdirAll(filepath.Dir(repo), 0o755); err != nil {
		return "", err
	}
	if out, err := exec.Command("git", "init", "--quiet", "--bare", "--", repo).CombinedOutput(); err != nil {
		_ = os.RemoveAll(repo)
		return "", fmt.Errorf("git init: %s", vcFirstLine(string(out), err))
	}
	// The work tree is named relatively, so the directory can move with its
	// repositories, as git1 does.
	for _, kv := range [][2]string{{"core.bare", "false"}, {"core.worktree", "../.."}} {
		if out, err := exec.Command("git", "--git-dir="+repo, "config", kv[0], kv[1]).CombinedOutput(); err != nil {
			_ = os.RemoveAll(repo)
			return "", fmt.Errorf("git config: %s", vcFirstLine(string(out), err))
		}
	}
	// The repository ignores every path but its own file, as git1's does.
	exclude := filepath.Join(repo, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		_ = os.RemoveAll(repo)
		return "", err
	}
	if err := os.WriteFile(exclude, []byte("/*\n!/"+gitIgnoreEscape(name)+"\n"), 0o644); err != nil {
		_ = os.RemoveAll(repo)
		return "", err
	}
	return repo, nil
}

// gitIgnoreEscape escapes a file name for a .gitignore line.
func gitIgnoreEscape(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if strings.ContainsRune(`\*?[#!`, r) {
			sb.WriteByte('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// vcGit runs git with args in the directory of file, returning its output. repo,
// when not empty, is the git directory to use rather than the enclosing one.
func vcGit(file, repo string, args ...string) (string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(abs)
	ctx, cancel := context.WithTimeout(context.Background(), vcTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if repo != "" {
		cmd.Env = append(os.Environ(), "GIT_DIR="+repo, "GIT_WORK_TREE="+dir)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// vcFirstLine is the first line git printed, which is what the echo area can
// show, or the error if git printed nothing.
func vcFirstLine(out string, err error) string {
	if line, _, _ := strings.Cut(strings.TrimSpace(out), "\n"); line != "" {
		return line
	}
	return err.Error()
}

package editor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Borderliner/nem/text"
)

// An archive chosen in yazi is not opened as bytes: its table of contents is
// listed in a read-only buffer, as tar, unzip or gzip print it.

// archiveTimeout bounds the listing of one archive.
const archiveTimeout = 10 * time.Second

// archiveCommand is the command that lists the archive at path, or nil when
// path is not an archive nem knows. tar reads the compression of a tarball
// itself; a lone .gz holds one file, which gzip -l describes.
func archiveCommand(path string) *exec.Cmd {
	name := strings.ToLower(filepath.Base(path))
	has := func(suffixes ...string) bool {
		for _, s := range suffixes {
			if strings.HasSuffix(name, s) {
				return true
			}
		}
		return false
	}
	switch {
	case has(".zip", ".jar", ".whl"):
		return exec.Command("unzip", "-l", path)
	case has(".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz", ".tar.zst"):
		return exec.Command("tar", "-tvf", path)
	case has(".gz"):
		return exec.Command("gzip", "-l", path)
	}
	return nil
}

// isArchive reports whether path is an archive that archiveList can show.
func isArchive(path string) bool { return archiveCommand(path) != nil }

// archiveList shows the contents of the archive at path in a read-only buffer.
func (e *Editor) archiveList(path string) (*text.Buffer, error) {
	cmd := archiveCommand(path)
	if cmd == nil {
		return nil, fmt.Errorf("%s is not an archive", filepath.Base(path))
	}
	ctx, cancel := context.WithTimeout(context.Background(), archiveTimeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("%s: %s", cmd.Args[0], vcFirstLine(string(ee.Stderr), nil))
		}
		return nil, fmt.Errorf("%s: %w", cmd.Args[0], err)
	}
	b := e.NewBuffer(e.uniqueName(filepath.Base(path)))
	if err := fillReadOnly(b, string(out)); err != nil {
		return nil, err
	}
	return b, nil
}

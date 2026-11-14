package lane

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// guardRelPath is where the commit guard lives, in the repository and in the
// deploy mirror `dotf hooks install` writes under DOTFILES_DIR.
var guardRelPath = filepath.Join("git-hooks", "lib", "lane-guard.sh")

// GuardDrift warns when the guard that runs, the deployed copy under
// deployRoot, differs from the repository's own: a deploy that lags keeps the
// old rules in force on every commit of the machine. A repository without a
// copy of its own has nothing to lag, and a copy that cannot be read is left to
// the commit that trips on it. Line endings are not drift.
func GuardDrift(repoRoot, deployRoot string) []Finding {
	want, err := os.ReadFile(filepath.Join(repoRoot, guardRelPath))
	if err != nil {
		return nil
	}
	deployed := filepath.Join(deployRoot, guardRelPath)
	got, err := os.ReadFile(deployed)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return []Finding{{deployed, "the guard is not deployed here, so the commit guard does not run on this machine", "dotf hooks install"}}
	case err != nil:
		return nil
	case !bytes.Equal(lf(got), lf(want)):
		return []Finding{{deployed, "differs from " + filepath.Join(repoRoot, guardRelPath) + ", so the rules in force are not the repository's", "dotf hooks install"}}
	}
	return nil
}

func lf(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }

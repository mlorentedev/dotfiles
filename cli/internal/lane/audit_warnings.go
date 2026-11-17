package lane

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GuardRelPath is where the commit guard sits under a hooks directory: in the
// repository's git-hooks/, and in whatever core.hooksPath names once `dotf hooks
// install` has mirrored the tree there.
const GuardRelPath = "lib/lane-guard.sh"

// GuardDrift warns when the guard that runs differs from the repository's own:
// a deploy that lags keeps the old rules in force on every commit of the
// machine. The guard that runs is the one under the hooks directory git
// answers for repoRoot (core.hooksPath when set, $GIT_DIR/hooks otherwise), so
// a hooksPath that is the checkout's own git-hooks is no drift by construction.
// A repository without a copy of its own has nothing to lag, and a copy that
// cannot be read is left to the commit that trips on it. Line endings are not
// drift. A hooks directory git cannot answer for is an error, not a guess.
func GuardDrift(repoRoot string) ([]Finding, error) {
	own := filepath.Join(repoRoot, "git-hooks", filepath.FromSlash(GuardRelPath))
	want, err := os.ReadFile(own)
	if err != nil {
		return nil, nil
	}
	hooks, err := hooksDir(repoRoot)
	if err != nil {
		return nil, err
	}
	deployed := filepath.Join(hooks, filepath.FromSlash(GuardRelPath))
	if samePath(deployed, own) {
		return nil, nil
	}
	got, err := os.ReadFile(deployed)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return []Finding{{deployed, "the guard is not deployed there, so it does not run on this machine", "dotf hooks install"}}, nil
	case err != nil:
		return nil, nil
	case !bytes.Equal(lf(got), lf(want)):
		return []Finding{{deployed, "differs from " + own + ", so the rules in force are not the repository's", "dotf hooks install"}}, nil
	}
	return nil, nil
}

// hooksDir asks git where it runs hooks from for repoRoot, so core.hooksPath
// (relative, ~-prefixed, from any config level) and the default $GIT_DIR/hooks
// are git's answer rather than a second reading of the config. A relative
// answer is relative to repoRoot, where git was run.
func hooksDir(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "--git-path", "hooks").Output()
	p := strings.TrimSpace(string(out))
	if err != nil || p == "" {
		return "", fmt.Errorf("cannot read the hooks path of %s: %v", repoRoot, err)
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(repoRoot, p)
	}
	return filepath.Clean(p), nil
}

func lf(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }

package lane

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// guardRelPath is where the commit guard lives, in the repository and under
// the hooks directory git runs hooks from.
var guardRelPath = filepath.Join("git-hooks", "lib", "lane-guard.sh")

// GuardDrift warns when the guard that runs differs from the repository's own:
// a deploy that lags keeps the old rules in force on every commit of the
// machine. The guard that runs is the one under core.hooksPath, as git resolves
// it for repoRoot, so a hooksPath that points at the checkout's own git-hooks
// is no drift by construction, and an unset one means no guard is deployed. A
// repository without a copy of its own has nothing to lag, and a copy that
// cannot be read is left to the commit that trips on it. Line endings are not
// drift.
func GuardDrift(repoRoot string) []Finding {
	own := filepath.Join(repoRoot, guardRelPath)
	want, err := os.ReadFile(own)
	if err != nil {
		return nil
	}
	hooks := hooksPath(repoRoot)
	if hooks == "" {
		return []Finding{{"core.hooksPath", "is unset, so the commit guard does not run on this machine", "dotf hooks install"}}
	}
	deployed := filepath.Join(hooks, "lib", filepath.Base(guardRelPath))
	if samePath(deployed, own) {
		return nil
	}
	got, err := os.ReadFile(deployed)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return []Finding{{deployed, "the guard is not deployed there (core.hooksPath), so it does not run on this machine", "dotf hooks install"}}
	case err != nil:
		return nil
	case !bytes.Equal(lf(got), lf(want)):
		return []Finding{{deployed, "differs from " + own + ", so the rules in force are not the repository's", "dotf hooks install"}}
	}
	return nil
}

// hooksPath is git's core.hooksPath for repoRoot as an absolute, symlink-free
// path, or empty when it is unset. A relative value is relative to the
// directory hooks run from, which for the main checkout is repoRoot; a leading
// ~ is expanded by git itself.
func hooksPath(repoRoot string) string {
	out, err := exec.Command("git", "-C", repoRoot, "config", "--get", "--type=path", "core.hooksPath").Output()
	p := strings.TrimSpace(string(out))
	if err != nil || p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(repoRoot, p)
	}
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}

func lf(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }

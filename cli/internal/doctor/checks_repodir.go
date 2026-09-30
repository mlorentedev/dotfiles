package doctor

import (
	"path/filepath"
	"runtime"
	"strings"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

// checkRepoDirResolves verifies the DOTFILES_REPO_DIR *cascade* (env ->
// machine.json -> contract default, ADR-025) points at a real dotfiles checkout.
//
// It resolves through envpkg.ResolvePath — the exact seam `dotf update` and
// `dotf mem` use — and deliberately does NOT apply the .git walk-up that
// resolveRepoDir (the deploy-drift check) does. The walk-up would find the
// checkout from doctor's own cwd and mask a phantom default; but update/mem run
// where no walk-up saves them (a systemd/Task-Scheduler timer, a session hook),
// so this check must fail exactly when those consumers would silently no-op on a
// fresh machine with an unseeded machine.json (#696).
func checkRepoDirResolves(sys *System, rep *Report) {
	rep.Section("Repo-dir resolution")
	repo := envpkg.ResolvePath("DOTFILES_REPO_DIR")
	if repo == "" {
		rep.Warn("DOTFILES_REPO_DIR does not resolve (no env var, machine.json override, or contract default)")
		return
	}
	if !isDir(repo) {
		rep.Fail("DOTFILES_REPO_DIR resolves to a missing path: " + repo +
			" — `dotf update`/`mem` will no-op; run setup (seeds machine.json) or `dotf env set DOTFILES_REPO_DIR <checkout>`")
		return
	}
	root := gitCheckoutRoot(sys, repo)
	switch {
	case root == "":
		rep.Fail("DOTFILES_REPO_DIR resolves to " + repo +
			" which is not a git checkout — run setup or `dotf env set DOTFILES_REPO_DIR <checkout>`")
	case !sameCheckoutRoot(repo, root):
		rep.Fail("DOTFILES_REPO_DIR resolves to " + repo + " which is not the checkout root " + root +
			" — run setup or `dotf env set DOTFILES_REPO_DIR <checkout>`")
	default:
		rep.Pass("DOTFILES_REPO_DIR cascade resolves to a checkout: " + repo)
	}
}

func gitCheckoutRoot(sys *System, path string) string {
	if sys == nil || sys.CommandOutput == nil {
		return ""
	}
	out, err := sys.CommandOutput("git", "-C", path, "rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(out))
}

func sameCheckoutRoot(configured, actual string) bool {
	a, errA := filepath.Abs(configured)
	b, errB := filepath.Abs(actual)
	if errA != nil || errB != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

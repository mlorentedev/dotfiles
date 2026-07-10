package doctor

import (
	"path/filepath"

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
func checkRepoDirResolves(rep *Report) {
	rep.Section("Repo-dir resolution")
	repo := envpkg.ResolvePath("DOTFILES_REPO_DIR")
	switch {
	case repo == "":
		rep.Warn("DOTFILES_REPO_DIR does not resolve (no env var, machine.json override, or contract default)")
	case !isDir(repo):
		rep.Fail("DOTFILES_REPO_DIR resolves to a missing path: " + repo +
			" — `dotf update`/`mem` will no-op; run setup (seeds machine.json) or `dotf env set DOTFILES_REPO_DIR <checkout>`")
	case !isDir(filepath.Join(repo, ".git")):
		rep.Fail("DOTFILES_REPO_DIR resolves to " + repo +
			" which is not a git checkout — run setup or `dotf env set DOTFILES_REPO_DIR <checkout>`")
	default:
		rep.Pass("DOTFILES_REPO_DIR cascade resolves to a checkout: " + repo)
	}
}

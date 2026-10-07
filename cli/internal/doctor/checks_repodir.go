package doctor

import (
	"os"
	"path/filepath"
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
	if sys == nil {
		return ""
	}
	var (
		out string
		err error
	)
	if sys.CommandOutputEnv != nil {
		out, err = sys.CommandOutputEnv(cleanGitRepositoryEnv(os.Environ()),
			"git", "-C", path, "rev-parse", "--show-toplevel")
	} else if sys.CommandOutput != nil {
		out, err = sys.CommandOutput("git", "-C", path, "rev-parse", "--show-toplevel")
	} else {
		return ""
	}
	if err != nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(out))
}

func cleanGitRepositoryEnv(environ []string) []string {
	repositoryLocal := map[string]bool{
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": true,
		"GIT_COMMON_DIR":                   true,
		"GIT_DIR":                          true,
		"GIT_GRAFT_FILE":                   true,
		"GIT_IMPLICIT_WORK_TREE":           true,
		"GIT_INDEX_FILE":                   true,
		"GIT_INTERNAL_SUPER_PREFIX":        true,
		"GIT_NO_REPLACE_OBJECTS":           true,
		"GIT_OBJECT_DIRECTORY":             true,
		"GIT_PREFIX":                       true,
		"GIT_REPLACE_REF_BASE":             true,
		"GIT_SHALLOW_FILE":                 true,
		"GIT_WORK_TREE":                    true,
	}
	clean := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		if !repositoryLocal[strings.ToUpper(key)] {
			clean = append(clean, entry)
		}
	}
	return clean
}

// sameCheckoutRoot reports whether the configured repo dir is the checkout
// root git reports, through whatever symlink, spelling or case reaches it.
func sameCheckoutRoot(configured, actual string) bool {
	return sameDir(configured, actual)
}

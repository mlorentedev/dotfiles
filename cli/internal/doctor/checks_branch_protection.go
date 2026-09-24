package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/forge"
)

// branchProtectionTimeout bounds one protection read, so an offline machine
// degrades to a WARN instead of stalling the sweep (doctor ends setup-linux.sh).
const branchProtectionTimeout = 15 * time.Second

// checkBranchProtection diffs every repository declared in
// forge/branch-protection.json against its live protection (GUARD-017, #1451):
// the detective half of protection-as-code. Branch protection leaves no trace
// in git, so without this a dropped required context is invisible until a
// merge that should have been impossible.
//
// Drift is a FAIL naming the repo and field. A declared state is a Skip that
// shows its reason. An unanswerable read is a WARN, and gh absent is a Skip:
// neither is ever a PASS, because "not asked" is not "matches".
func checkBranchProtection(sys *System, rep *Report) {
	rep.Section("branch-protection")

	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — set DOTFILES_REPO_DIR or run from a checkout")
		return
	}
	if _, err := os.Stat(filepath.Join(repo, forge.DeclarationFile)); os.IsNotExist(err) {
		rep.Skip("no " + forge.DeclarationFile + " in " + repo + " — nothing is declared")
		return
	}
	if _, err := sys.LookPath("gh"); err != nil {
		rep.Skip("gh not on PATH — cannot read live branch protection")
		return
	}
	decl, err := forge.Load(repo)
	if err != nil {
		rep.Fail("cannot load the declaration: " + err.Error())
		return
	}
	results := forge.CheckAll(decl, func(args ...string) (string, string, error) {
		return sys.CommandOutputBounded(branchProtectionTimeout, "gh", args...)
	})
	reportBranchProtection(results, rep)
}

func reportBranchProtection(results []forge.RepoResult, rep *Report) {
	ok := 0
	for _, r := range results {
		switch r.Status {
		case forge.StatusOK:
			ok++
		case forge.StatusDrift:
			for _, ch := range r.Changes {
				rep.Fail(fmt.Sprintf("%s: %s: declared %s, live %s — the forge moved, or the declaration is stale; `dotf forge protection check --repo %s`",
					r.Repo, ch.Field, ch.Declared, ch.Live, r.Repo))
			}
		case forge.StatusState:
			rep.Skip(r.Repo + ": " + r.Detail)
		default:
			rep.Warn(r.Repo + ": not verified — " + r.Detail)
		}
	}
	if ok > 0 {
		rep.Pass(fmt.Sprintf("%d repositories match their declaration", ok))
	}
}

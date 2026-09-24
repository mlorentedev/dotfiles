package doctor

import (
	"fmt"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// specIssueStateTimeout bounds one forge lookup, so an offline machine
// degrades to a WARN instead of stalling the sweep (doctor ends setup-linux.sh).
const specIssueStateTimeout = 15 * time.Second

// checkSpecIssueState runs `dotf spec audit` (SDD-041) over the dotfiles
// checkout: every active spec must track an open issue. archive-on-merge only
// sees issues closed through a PR's closing keyword, so a hand-closed issue
// leaves its spec active forever (#1087) — this is the net for that path.
//
// A lookup the forge could not answer is a WARN that says so, and gh absent
// is a Skip: neither is ever a PASS, because "not asked" is not "all clear".
func checkSpecIssueState(sys *System, rep *Report) {
	rep.Section("spec-issue-state")

	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — set DOTFILES_REPO_DIR or run from a checkout")
		return
	}
	if _, err := sys.LookPath("gh"); err != nil {
		rep.Skip("gh not on PATH — cannot ask the forge whether each spec's issue is still open")
		return
	}
	origin, err := sys.CommandOutput("git", "-C", repo, "remote", "get-url", "origin")
	home, perr := initrepo.ParseOriginRepo(origin)
	if err != nil || perr != nil {
		rep.Warn("no parseable origin remote in " + repo + " — cannot tell which repo owns specs/, audit not run")
		return
	}

	lookup := spec.GHIssueStateLookup(func(args ...string) (string, string, error) {
		return sys.CommandOutputBounded(specIssueStateTimeout, "gh", args...)
	})
	findings, err := spec.AuditIssueState(repo, home, lookup)
	if err != nil {
		rep.Warn("cannot read specs/: " + err.Error())
		return
	}
	reportSpecIssueState(findings, rep)
}

func reportSpecIssueState(findings []spec.AuditFinding, rep *Report) {
	ok := 0
	for _, f := range findings {
		switch f.Severity {
		case spec.SeverityFail:
			rep.Fail(f.SpecID + ": " + f.Detail + " — disposition it: `dotf spec archive`, or --abandoned")
		case spec.SeverityWarn:
			rep.Warn(f.SpecID + ": " + f.Detail)
		case spec.SeverityUnanswerable:
			rep.Warn(f.SpecID + ": not verified — " + f.Detail)
		default:
			ok++
		}
	}
	switch {
	case len(findings) == 0:
		rep.Pass("no active specs")
	case ok > 0:
		rep.Pass(fmt.Sprintf("%d active spec(s) track an open issue", ok))
	}
}

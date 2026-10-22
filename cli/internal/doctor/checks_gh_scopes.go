package doctor

import (
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/forge"
)

// ghScopesTimeout bounds the one API call, so an offline machine degrades to a
// WARN instead of stalling the sweep.
const ghScopesTimeout = 15 * time.Second

// checkGHScopes (#2204) compares the OAuth scopes of the gh token with the ones
// the board tooling needs. A token without read:project fails `gh project
// item-list` with an error that reads like a permissions bug in the repo, hours
// after the login that should have asked for it.
//
// Not logged in and an unreachable API are WARNs; gh absent, and a token that
// does not report scopes (fine-grained, app), are Skips: "not asked" is never a
// PASS.
func checkGHScopes(sys *System, rep *Report) {
	rep.Section("GitHub token scopes")
	if _, err := sys.LookPath("gh"); err != nil {
		rep.Skip("gh not on PATH — cannot read the token's scopes")
		return
	}
	headers, stderr, err := sys.CommandOutputBounded(ghScopesTimeout, "gh", "api", "-i", "user")
	scopes, present := forge.ParseOAuthScopes(headers)
	if !present {
		if err != nil {
			if strings.Contains(stderr, "gh auth login") {
				rep.Warn("gh is not logged in: " + forge.GHLoginCommand())
				return
			}
			reason := firstLine(stderr)
			if reason == "" {
				reason = err.Error()
			}
			rep.Warn("could not read the token's scopes: " + reason)
			return
		}
		rep.Skip("this token does not report its scopes (fine-grained or app token)")
		return
	}
	if missing := forge.MissingGHScopes(scopes); len(missing) > 0 {
		rep.Warn("the gh token lacks " + strings.Join(missing, ", ") + ": " + forge.GHRefreshCommand(missing))
		return
	}
	rep.Pass("the gh token carries every scope the board tooling needs")
}

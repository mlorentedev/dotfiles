#!/usr/bin/env bash
# gh's auth precondition for the real-dependency suites. Load with: load 'lib/gh-auth'
#
# `gh pr list --json` checks that it has a token, then validates the field
# names, and only then resolves a repository. So what a real-gh suite depends
# on is a token, not a reachable API. The suites used to ask `gh auth status`,
# which validates the token over the network: one API blip read as "not
# authenticated" and, under CI, failed the suite. Main went red that way on
# run 38496700430 and passed on the next push (#2442).
#
# This asks only the question the suites depend on, and asks it offline. A
# genuinely missing token still fails closed under CI, which is why the
# suites fail rather than skip (#2021).

# gh_has_token succeeds when gh has a token to send: GH_TOKEN or GITHUB_TOKEN
# in the environment, else a stored login. The token is never printed.
gh_has_token() {
    [ -n "${GH_TOKEN:-}" ] || [ -n "${GITHUB_TOKEN:-}" ] || gh auth token >/dev/null 2>&1
}

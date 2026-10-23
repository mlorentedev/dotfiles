package forge

import (
	"slices"
	"strings"
)

// RequiredGHScopes are the OAuth scopes the machine's gh token must carry: the
// board tooling (`gh project item-list`, the catchup skill, the status checks)
// needs read:project, and the rest is what the repo's own flows already use.
// It is the one list: doctor checks against it, and the runbooks quote the
// login command built from it (#2204).
var RequiredGHScopes = []string{"gist", "read:org", "read:project", "repo", "workflow"}

// impliedBy lists the broader scopes that grant a required one: GitHub grants
// the read scope to a token that holds the write or admin scope of its group.
var impliedBy = map[string][]string{
	"read:project": {"project"},
	"read:org":     {"write:org", "admin:org"},
}

// ParseOAuthScopes reads the token's scopes from the response headers of
// `gh api -i`. present is false when the response carries no X-OAuth-Scopes
// header at all, which is what a fine-grained or app token returns: those do
// not report scopes, and "none reported" is not "none granted".
func ParseOAuthScopes(headers string) (scopes []string, present bool) {
	for _, line := range strings.Split(headers, "\n") {
		name, value, ok := strings.Cut(strings.TrimRight(line, "\r"), ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "x-oauth-scopes") {
			continue
		}
		present = true
		for _, s := range strings.Split(value, ",") {
			if s = strings.TrimSpace(s); s != "" {
				scopes = append(scopes, s)
			}
		}
	}
	return scopes, present
}

// MissingGHScopes returns the required scopes that have does not grant, in the
// order RequiredGHScopes declares them.
func MissingGHScopes(have []string) []string {
	var missing []string
	for _, want := range RequiredGHScopes {
		if slices.Contains(have, want) {
			continue
		}
		if slices.ContainsFunc(impliedBy[want], func(s string) bool { return slices.Contains(have, s) }) {
			continue
		}
		missing = append(missing, want)
	}
	return missing
}

// GHLoginCommand is the login that asks for every required scope at once.
func GHLoginCommand() string {
	return "gh auth login -s " + strings.Join(RequiredGHScopes, ",")
}

// GHRefreshCommand adds the missing scopes to an existing login.
func GHRefreshCommand(missing []string) string {
	return "gh auth refresh -h github.com -s " + strings.Join(missing, ",")
}

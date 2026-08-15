package secrets

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// GitHubSecretSetter sets one repository Actions secret (create or update). The seam that
// keeps `dotf secrets sync ci` unit-testable with no gh, no network, no real secret —
// tests inject a fake; production is GHSecretSet (a `gh secret set` shell-out). It is the
// GitHub-Actions write analog of BWWriter (Bitwarden).
type GitHubSecretSetter interface {
	SetSecret(repo, name, value string) error
}

// GHSecretSet is the production GitHubSecretSetter: it shells out to the GitHub CLI to set
// a repository Actions secret, delivering the value on stdin — never an argv element or a
// temp file (the plaintext-at-rest minimization of ADR-028). `gh secret set` creates or
// overwrites, so a re-run is idempotent. Live-verified with the operator's `gh auth` (the
// canary smoke, #612 C8), not in CI — like BWGet/BWPut.
type GHSecretSet struct {
	Bin string // gh binary name/path; "" → "gh"
}

// SetSecret uploads name=value to repo's Actions secrets via one `gh secret set` call.
func (g GHSecretSet) SetSecret(repo, name, value string) error {
	bin := g.Bin
	if bin == "" {
		bin = "gh"
	}
	cmd := exec.Command(bin, "secret", "set", name, "--repo", repo) //nolint:gosec // name/repo are operator-controlled registry/flag data
	cmd.Stdin = strings.NewReader(value)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("gh secret set %s --repo %s: %s", name, repo, msg)
	}
	return nil
}

// CISkip records one secret/var excluded from a CI sync, with the reason (for reporting).
type CISkip struct {
	ID     string
	Reason string
}

// CISelection is the outcome of SelectCI: the flattened env entries to resolve + upload,
// and the secrets/vars excluded with a reason.
type CISelection struct {
	Upload  []Entry
	Skipped []CISkip
}

// SelectCI picks the secrets feeding repo's CI — those whose consumers contains
// "ci:<repo>" — and splits them into the env vars to upload and the ones excluded. The
// repo (not a workflow "purpose") is the routing key, because `gh secret set` is per-repo
// (ADR-029). A secret is excluded — never silently — when it is a floor/age-offline secret
// (needed before the store is reachable; never a CI secret), a file secret (not a GitHub
// Actions secret), or exposes a GITHUB_*-prefixed var (Actions reserves the prefix). The
// result is backend-agnostic: age- and bw-backed entries flatten identically, so the
// caller resolves both through one Loader path — the whole point of ADR-029.
func (r *Registry) SelectCI(repo string) CISelection {
	tag := "ci:" + repo
	var sel CISelection
	for i := range r.Secrets {
		s := &r.Secrets[i]
		if !slices.Contains(s.Consumers, tag) {
			continue
		}
		switch {
		case s.Plane == "floor" || s.Backend == BackendAgeOffline:
			sel.Skipped = append(sel.Skipped, CISkip{s.ID, "floor/offline secret — never pushed to CI"})
			continue
		case s.Expose.File != nil:
			sel.Skipped = append(sel.Skipped, CISkip{s.ID, "file secret — not a GitHub Actions secret"})
			continue
		}
		for _, e := range s.envEntries() {
			if strings.HasPrefix(e.Var, "GITHUB_") {
				sel.Skipped = append(sel.Skipped, CISkip{e.Var, "GITHUB_* prefix is reserved by Actions — rename the var at the workflow"})
				continue
			}
			// Validate arrives on the entry from the flattening itself — it used
			// to be re-applied here, the only place that carried it, which is why
			// every other Entries() consumer saw an empty Validate (REFACTOR-012).
			sel.Upload = append(sel.Upload, e)
		}
	}
	return sel
}

// envEntries flattens a non-file secret to its env entries, dispatching on backend like
// Registry.Entries (home is irrelevant — env vars never materialize a file).
func (s *Secret) envEntries() []Entry {
	if s.Backend == BackendBW {
		return s.bwEntries("")
	}
	return s.ageEntries("")
}

// GitHubTokenValidator checks that a resolved GitHub token actually authenticates. The
// seam that lets `dotf secrets sync ci` refuse to upload a dead PAT, while staying
// testable with no gh and no network (tests inject a fake). Opt-in per registry entry
// (validate: github-token); other secrets are not GitHub tokens and can't be probed
// this way — liveness validation does NOT generalize across providers (ADR-028).
type GitHubTokenValidator interface {
	Validate(token string) error
}

// GHTokenValidate is the production GitHubTokenValidator: it runs `gh api user` with the
// token under test as GH_TOKEN. An expired/revoked token makes the GitHub API return
// 401/403, so gh exits non-zero and the error propagates. Live-path only (like
// GHSecretSet/BWGet), never exercised in CI.
type GHTokenValidate struct {
	Bin string // gh binary name/path; "" → "gh"
}

// Validate returns nil when the token authenticates, else an error describing the failure.
func (g GHTokenValidate) Validate(token string) error {
	bin := g.Bin
	if bin == "" {
		bin = "gh"
	}
	cmd := exec.Command(bin, "api", "user", "-q", ".login") //nolint:gosec // fixed args, no user input
	// Authenticate AS the token under test, not the ambient `gh auth`: strip any inherited
	// GH_TOKEN/GITHUB_TOKEN before injecting ours, so the probe can't pass on the wrong creds.
	filtered := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GH_TOKEN=") || strings.HasPrefix(kv, "GITHUB_TOKEN=") {
			continue
		}
		filtered = append(filtered, kv)
	}
	cmd.Env = append(filtered, "GH_TOKEN="+token)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("gh api user: %s", msg)
	}
	return nil
}

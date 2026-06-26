package secrets

import (
	"bytes"
	"fmt"
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
		case s.Plane == "floor" || s.Backend == "age-offline":
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
			sel.Upload = append(sel.Upload, e)
		}
	}
	return sel
}

// envEntries flattens a non-file secret to its env entries, dispatching on backend like
// Registry.Entries (home is irrelevant — env vars never materialize a file).
func (s *Secret) envEntries() []Entry {
	if s.Backend == "bw" {
		return s.bwEntries("")
	}
	return s.ageEntries("")
}

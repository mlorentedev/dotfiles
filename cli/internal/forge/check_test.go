package forge

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

// ghFake answers `gh api repos/<repo>/branches/<b>/protection` from a table:
// a body, or a gh-style stderr line with an exit error. It records calls.
type ghFake struct {
	bodies map[string]string
	errs   map[string]string
	calls  []string
	mu     sync.Mutex
}

func (g *ghFake) run(args ...string) (string, string, error) {
	path := args[len(args)-1]
	g.mu.Lock()
	g.calls = append(g.calls, path)
	g.mu.Unlock()
	if b, ok := g.bodies[path]; ok {
		return b, "", nil
	}
	if e, ok := g.errs[path]; ok {
		return "", e, errors.New("exit status 1")
	}
	return "", "error connecting to api.github.com\n", errors.New("exit status 1")
}

const dotPath = "repos/mlorentedev/dotfiles/branches/main/protection"

func declaredLikeLive(t *testing.T) RepoDecl {
	t.Helper()
	p := normalised(t, "get-dotfiles.json").Protection
	return RepoDecl{Branch: "main", Protection: &p}
}

func TestProtectionCheckMatchesLive(t *testing.T) {
	g := &ghFake{bodies: map[string]string{dotPath: string(fixture(t, "get-dotfiles.json"))}}
	r := CheckRepo("mlorentedev/dotfiles", declaredLikeLive(t), g.run)
	if r.Status != StatusOK || len(r.Changes) != 0 {
		t.Fatalf("a declaration equal to live must be ok: %+v", r)
	}
}

func TestProtectionCheckReportsDrift(t *testing.T) {
	d := declaredLikeLive(t)
	d.Protection.EnforceAdmins = false
	g := &ghFake{bodies: map[string]string{dotPath: string(fixture(t, "get-dotfiles.json"))}}
	r := CheckRepo("mlorentedev/dotfiles", d, g.run)
	if r.Status != StatusDrift || len(r.Changes) != 1 || r.Changes[0].Field != "enforce_admins" {
		t.Fatalf("want drift on enforce_admins, got %+v", r)
	}
}

func TestProtectionCheckProtectionRemovedIsDrift(t *testing.T) {
	g := &ghFake{errs: map[string]string{dotPath: "gh: Branch not protected (HTTP 404)\n"}}
	r := CheckRepo("mlorentedev/dotfiles", declaredLikeLive(t), g.run)
	if r.Status != StatusDrift || len(r.Changes) != 1 || r.Changes[0].Field != "protection" || r.Changes[0].Live != "none" {
		t.Fatalf("declared protection with none live must drift on the whole object: %+v", r)
	}
}

func TestProtectionCheckUnprotectedStates(t *testing.T) {
	d := RepoDecl{Branch: "main", State: StateUnprotected, Reason: "no protection yet"}
	notProtected := &ghFake{errs: map[string]string{dotPath: "gh: Branch not protected (HTTP 404)\n"}}
	if r := CheckRepo("mlorentedev/dotfiles", d, notProtected.run); r.Status != StatusState || !strings.Contains(r.Detail, "no protection yet") {
		t.Errorf("declared unprotected and confirmed must be a state carrying its reason: %+v", r)
	}
	protected := &ghFake{bodies: map[string]string{dotPath: string(fixture(t, "get-dotfiles.json"))}}
	if r := CheckRepo("mlorentedev/dotfiles", d, protected.run); r.Status != StatusDrift {
		t.Errorf("declared unprotected but protected live must drift: %+v", r)
	}
}

func TestProtectionCheckUnavailableIsNotQueried(t *testing.T) {
	g := &ghFake{}
	r := CheckRepo("mlorentedev/knowledge", RepoDecl{Branch: "master", State: StateUnavailable, Reason: "private, free plan"}, g.run)
	if r.Status != StatusState || len(g.calls) != 0 {
		t.Fatalf("an unavailable repo is reported by its declared reason, without a call: %+v calls=%v", r, g.calls)
	}
}

func TestProtectionCheckUnanswerable(t *testing.T) {
	for name, stderr := range map[string]string{
		"403 plan limit": "gh: Upgrade to GitHub Pro or make this repository public to enable this feature. (HTTP 403)\n",
		"network":        "error connecting to api.github.com\n",
		"auth":           "gh: Bad credentials (HTTP 401)\n",
	} {
		g := &ghFake{errs: map[string]string{dotPath: stderr}}
		r := CheckRepo("mlorentedev/dotfiles", declaredLikeLive(t), g.run)
		if r.Status != StatusUnanswerable || r.Detail == "" {
			t.Errorf("%s: a question the forge could not answer must be unanswerable, never ok: %+v", name, r)
		}
	}
	g := &ghFake{errs: map[string]string{dotPath: "gh: Upgrade to GitHub Pro or make this repository public to enable this feature. (HTTP 403)\n"}}
	if r := CheckRepo("mlorentedev/dotfiles", declaredLikeLive(t), g.run); !strings.Contains(r.Detail, StateUnavailable) {
		t.Errorf("a 403 should point at the unavailable state: %q", r.Detail)
	}
}

func TestProtectionCheckAllIsSortedAndAttentive(t *testing.T) {
	d := Declaration{Repos: map[string]RepoDecl{
		"mlorentedev/dotfiles":  declaredLikeLive(t),
		"mlorentedev/knowledge": {Branch: "master", State: StateUnavailable, Reason: "private, free plan"},
	}}
	g := &ghFake{bodies: map[string]string{dotPath: string(fixture(t, "get-dotfiles.json"))}}
	rs := CheckAll(d, g.run)
	if len(rs) != 2 || rs[0].Repo != "mlorentedev/dotfiles" || rs[1].Repo != "mlorentedev/knowledge" {
		t.Fatalf("results must cover every repo, sorted: %+v", rs)
	}
	if NeedsAttention(rs) {
		t.Error("ok plus a declared state needs no attention")
	}
	if !NeedsAttention(append(rs, RepoResult{Status: StatusUnanswerable})) || !NeedsAttention(append(rs, RepoResult{Status: StatusDrift})) {
		t.Error("drift and unanswerable both need attention")
	}
}

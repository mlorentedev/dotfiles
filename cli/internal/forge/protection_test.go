package forge

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func normalised(t *testing.T, name string) Live {
	t.Helper()
	live, err := Normalise(fixture(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return live
}

// The fixtures are real GET responses captured 2026-09-23 (URLs stripped).
func TestProtectionNormaliseUnwrapsTheGetShape(t *testing.T) {
	p := normalised(t, "get-dotfiles.json").Protection
	if p.RequiredStatusChecks == nil || !p.RequiredStatusChecks.Strict || len(p.RequiredStatusChecks.Checks) != 5 {
		t.Fatalf("status checks not normalised: %+v", p.RequiredStatusChecks)
	}
	for _, c := range p.RequiredStatusChecks.Checks {
		if c.AppID == nil || *c.AppID != 15368 {
			t.Errorf("check %q lost its source app: %v", c.Context, c.AppID)
		}
	}
	if p.RequiredPullRequestReviews == nil || p.RequiredPullRequestReviews.RequiredApprovingReviewCount != 0 {
		t.Fatalf("reviews not normalised: %+v", p.RequiredPullRequestReviews)
	}
	if !p.EnforceAdmins || p.AllowForcePushes || p.RequiredLinearHistory {
		t.Errorf("{enabled} wrappers not unwrapped: %+v", p)
	}
}

// A block the API omits is "not required", which the declaration spells null.
func TestProtectionNormaliseAbsentBlocksAreNull(t *testing.T) {
	if p := normalised(t, "get-iris.json").Protection; p.RequiredStatusChecks != nil {
		t.Errorf("iris requires no status checks, so the block must be null: %+v", p.RequiredStatusChecks)
	}
	if p := normalised(t, "get-pollex.json").Protection; p.RequiredPullRequestReviews != nil {
		t.Errorf("pollex requires no pull request, so the block must be null: %+v", p.RequiredPullRequestReviews)
	}
}

func TestProtectionDiffEqualIsEmpty(t *testing.T) {
	live := normalised(t, "get-dotfiles.json")
	if changes := Diff(live.Protection, live); len(changes) != 0 {
		t.Fatalf("a declaration equal to live must not drift: %v", changes)
	}
}

func TestProtectionDiffIgnoresCheckOrder(t *testing.T) {
	live := normalised(t, "get-dotfiles.json")
	declared := live.Protection
	checks := append([]Check(nil), declared.RequiredStatusChecks.Checks...)
	checks[0], checks[len(checks)-1] = checks[len(checks)-1], checks[0]
	declared.RequiredStatusChecks = &StatusChecks{Strict: true, Checks: checks}
	if changes := Diff(declared, live); len(changes) != 0 {
		t.Fatalf("check order is not a requirement: %v", changes)
	}
}

func TestProtectionDiffNamesEveryDriftedField(t *testing.T) {
	live := normalised(t, "get-dotfiles.json")
	declared := live.Protection
	other := 99
	checks := append([]Check(nil), declared.RequiredStatusChecks.Checks[1:]...) // drop "lint"
	checks[0].AppID = &other                                                    // re-source "lint-powershell"
	declared.RequiredStatusChecks = &StatusChecks{Strict: true, Checks: append(checks, Check{Context: "spec-gate", AppID: nil})}
	declared.EnforceAdmins = false
	declared.RequiredPullRequestReviews = &Reviews{RequiredApprovingReviewCount: 1}

	got := map[string]Change{}
	for _, c := range Diff(declared, live) {
		got[c.Field] = c
	}
	for _, field := range []string{"required_status_checks.checks", "enforce_admins", "required_pull_request_reviews.required_approving_review_count"} {
		if _, ok := got[field]; !ok {
			t.Errorf("drift in %s not reported; got %v", field, got)
		}
	}
	cc := got["required_status_checks.checks"]
	for _, want := range []string{"lint@15368", "spec-gate@any", "lint-powershell@99"} {
		if !strings.Contains(cc.Declared+" "+cc.Live, want) {
			t.Errorf("checks drift should show %q, got declared=%q live=%q", want, cc.Declared, cc.Live)
		}
	}
	if len(got) != 3 {
		t.Errorf("only the three drifted fields may be reported, got %v", got)
	}
}

// A declared null block against a live one is drift, and vice versa: "not
// required" and "required" differ even when every sub-field would default.
func TestProtectionDiffNullBlockAgainstLive(t *testing.T) {
	live := normalised(t, "get-dotfiles.json")
	declared := live.Protection
	declared.RequiredStatusChecks = nil
	changes := Diff(declared, live)
	if len(changes) == 0 || changes[0].Field != "required_status_checks" {
		t.Fatalf("declaring no status checks against required ones must drift on the block: %v", changes)
	}
}

func TestProtectionDiffSeesRestrictions(t *testing.T) {
	data := strings.Replace(string(fixture(t, "get-dotfiles.json")), `"required_signatures"`, `"restrictions": {"users": [], "teams": []}, "required_signatures"`, 1)
	live, err := Normalise([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	changes := Diff(live.Protection, live)
	if len(changes) != 1 || changes[0].Field != "restrictions" {
		t.Fatalf("push restrictions are never declared, so a live one is drift: %v", changes)
	}
}

func validDoc(repos string) string {
	return `{"policy":{"required_approving_review_count":0,"rationale":"single maintainer: self-approval is impossible and enforce_admins is on"},"repos":{` + repos + `}}`
}

func TestProtectionSchemaAcceptsEachDeclaredShape(t *testing.T) {
	schema := readRepoFile(t, SchemaFile)
	for name, repos := range map[string]string{
		"state":      `"o/a":{"branch":"main","state":"unavailable","reason":"private repository on the free plan"}`,
		"protection": `"o/b":{"branch":"main","protection":` + string(protectionJSON(t)) + `}`,
	} {
		if err := Validate([]byte(validDoc(repos)), schema); err != nil {
			t.Errorf("%s: a valid declaration was rejected: %v", name, err)
		}
	}
}

func TestProtectionSchemaRejectsAnIncompleteRepo(t *testing.T) {
	schema := readRepoFile(t, SchemaFile)
	for name, repos := range map[string]string{
		"neither object nor state":    `"o/a":{"branch":"main"}`,
		"state without reason":        `"o/a":{"branch":"main","state":"unprotected"}`,
		"both object and state":       `"o/a":{"branch":"main","state":"unprotected","reason":"xxxxxxxxxxxx","protection":` + string(protectionJSON(t)) + `}`,
		"object missing a field":      `"o/a":{"branch":"main","protection":{"enforce_admins":true}}`,
		"check without its source":    `"o/a":{"branch":"main","protection":` + strings.Replace(string(protectionJSON(t)), `"app_id":15368`, `"x":1`, 1) + `}`,
		"repo key that is not a slug": `"not-a-slug":{"branch":"main","state":"unavailable","reason":"xxxxxxxxxxxx"}`,
	} {
		if err := Validate([]byte(validDoc(repos)), schema); err == nil {
			t.Errorf("%s: an incomplete declaration was accepted", name)
		}
	}
}

func protectionJSON(t *testing.T) []byte {
	t.Helper()
	data, err := marshalProtection(normalised(t, "get-dotfiles.json").Protection)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readRepoFile(t *testing.T, rel string) []byte {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, SchemaFile)); err == nil {
			return dir
		}
		if filepath.Dir(dir) == dir {
			t.Fatalf("no %s above %s", SchemaFile, wd)
		}
	}
}

// D-1 (#1625), held by a test so the approvals policy cannot change quietly.
// Raising the count to 1 or more locks the only maintainer out of every merge
// (self-approval is impossible and enforce_admins is on). Changing either the
// value or its rationale therefore requires editing these pins in the same
// PR, where the diff makes the decision visible to a reviewer.
const (
	pinnedApprovals    = 0
	pinnedRationaleSHA = "ba5430c813d388708685d4dab0abe0c8b788e7398c03e375205877b71914308c"
)

func TestProtectionApprovalsRationalePinned(t *testing.T) {
	d, err := Load(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if d.Policy.RequiredApprovingReviewCount != pinnedApprovals {
		t.Errorf("policy.required_approving_review_count = %d, pinned %d — re-read D-1 on #1625, then update the rationale and this pin together",
			d.Policy.RequiredApprovingReviewCount, pinnedApprovals)
	}
	if got := sha256Hex(d.Policy.Rationale); got != pinnedRationaleSHA {
		t.Errorf("policy.rationale changed (sha256 %s, pinned %s) — a changed rationale is a changed decision; update the pin in the same PR", got, pinnedRationaleSHA)
	}
	checked := 0
	for repo, r := range d.Repos {
		if r.Protection == nil || r.Protection.RequiredPullRequestReviews == nil {
			continue
		}
		checked++
		if n := r.Protection.RequiredPullRequestReviews.RequiredApprovingReviewCount; n != d.Policy.RequiredApprovingReviewCount && r.ApprovalsOverrideReason == "" {
			t.Errorf("%s requires %d approvals against a policy of %d, with no approvals_override_reason", repo, n, d.Policy.RequiredApprovingReviewCount)
		}
	}
	if checked == 0 {
		t.Fatal("no repository declares a review requirement — the policy check covered nothing")
	}
}

func TestProtectionDeclarationLoads(t *testing.T) {
	d, err := Load(repoRoot(t))
	if err != nil {
		t.Fatalf("%s must satisfy its schema: %v", DeclarationFile, err)
	}
	if _, ok := d.Repos["mlorentedev/dotfiles"]; !ok {
		t.Fatal("the declaration must cover this repository")
	}
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Every {enabled} wrapper must be READ, not merely unwrapped: the captured
// fixtures happen to share most values (enforce_admins is true in all of them),
// so a normaliser that hard-coded one would pass every other test. Flip each
// flag in the raw response and require the normalised field to flip with it.
func TestProtectionNormaliseReadsEveryFlag(t *testing.T) {
	base := normalised(t, "get-dotfiles.json")
	raw := string(fixture(t, "get-dotfiles.json"))
	for _, field := range []string{
		"enforce_admins", "required_signatures", "required_linear_history", "allow_force_pushes",
		"allow_deletions", "block_creations", "required_conversation_resolution", "lock_branch", "allow_fork_syncing",
	} {
		was := flatten(base.Protection)[field]
		flipped := map[string]string{"true": "false", "false": "true"}[was]
		re := regexp.MustCompile(`("` + field + `":\s*\{\s*"enabled":\s*)` + was)
		mutated := re.ReplaceAllString(raw, "${1}"+flipped)
		if mutated == raw {
			t.Fatalf("fixture: could not flip %s", field)
		}
		live, err := Normalise([]byte(mutated))
		if err != nil {
			t.Fatal(err)
		}
		if got := flatten(live.Protection)[field]; got != flipped {
			t.Errorf("%s: flipped to %s in the response, normalised as %s", field, flipped, got)
		}
	}
}

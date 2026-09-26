package forge

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// forgeFake is a stateful fake forge: a PUT replaces the protection a later
// GET returns, as GitHub's does, unless ignorePut simulates a write the forge
// accepted and did not apply. Merged pull requests and what reported on each
// head drive the reported-context preflight.
type forgeFake struct {
	t          *testing.T
	live       string            // current GET body; "" means "Branch not protected"
	reported   map[string]string // head sha -> check-runs JSON
	statuses   map[string]string // head sha -> combined status JSON
	pulls      string            // closed pulls JSON
	ignorePut  bool
	puts       []map[string]any
	signatures []string // methods sent to required_signatures
	calls      []string
}

const dotRepo = "mlorentedev/dotfiles"

func (f *forgeFake) run(args ...string) (string, string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	method, path, input := "GET", "", ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-X":
			method = args[i+1]
			i++
		case "--input":
			input = args[i+1]
			i++
		default:
			path = args[i]
		}
	}
	switch {
	case strings.HasSuffix(path, "/required_signatures"):
		f.signatures = append(f.signatures, method)
		return "{}", "", nil
	case strings.HasSuffix(path, "/protection") && method == "PUT":
		return f.put(input)
	case strings.HasSuffix(path, "/protection"):
		if f.live == "" {
			return `{"message":"Branch not protected","status":"404"}`, "gh: Branch not protected (HTTP 404)\n", errors.New("exit status 1")
		}
		return f.live, "", nil
	case strings.Contains(path, "/pulls?"):
		return f.pulls, "", nil
	case strings.HasSuffix(path, "/check-runs?per_page=100"):
		return orEmpty(f.reported[shaOf(path)], `{"check_runs":[]}`), "", nil
	case strings.HasSuffix(path, "/status"):
		return orEmpty(f.statuses[shaOf(path)], `{"statuses":[]}`), "", nil
	}
	f.t.Fatalf("unexpected gh call: %v", args)
	return "", "", nil
}

// put records the body and, unless ignorePut, makes it the live state in the
// GET shape, so the re-read sees what a real forge would.
func (f *forgeFake) put(input string) (string, string, error) {
	raw, err := os.ReadFile(input) // #nosec G304 -- a temp file this package wrote
	if err != nil {
		f.t.Fatalf("PUT body unreadable: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		f.t.Fatalf("PUT body is not JSON: %v", err)
	}
	f.puts = append(f.puts, body)
	if !f.ignorePut {
		f.live = getShapeOf(f.t, body, f.live)
	}
	return "{}", "", nil
}

// getShapeOf renders a PUT body as the GET response it produces: flags gain
// their {"enabled"} wrapper; required_signatures, which PUT cannot set, keeps
// its previous value.
func getShapeOf(t *testing.T, put map[string]any, prev string) string {
	t.Helper()
	get := map[string]any{}
	for k, v := range put {
		switch v.(type) {
		case bool:
			get[k] = map[string]any{"enabled": v}
		default:
			if v != nil {
				get[k] = v
			}
		}
	}
	get["required_signatures"] = map[string]any{"enabled": false}
	if prev != "" {
		var old map[string]any
		_ = json.Unmarshal([]byte(prev), &old)
		if sig, ok := old["required_signatures"]; ok {
			get["required_signatures"] = sig
		}
	}
	out, err := json.Marshal(get)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func shaOf(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "commits" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func orEmpty(s, empty string) string {
	if s == "" {
		return empty
	}
	return s
}

// mergedPulls renders n merged pull requests with heads sha0..sha<n-1>, plus
// one closed-unmerged PR that must never count.
func mergedPulls(n int) string {
	items := []string{`{"merged_at":null,"head":{"sha":"unmerged"}}`}
	for i := 0; i < n; i++ {
		items = append(items, fmt.Sprintf(`{"merged_at":"2026-09-2%dT00:00:00Z","head":{"sha":"sha%d"}}`, i, i))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func checkRun(name string, app int) string {
	return fmt.Sprintf(`{"check_runs":[{"name":%q,"app":{"id":%d}}]}`, name, app)
}

func withSpecGate(t *testing.T) RepoDecl {
	t.Helper()
	d := declaredLikeLive(t)
	app := 15368
	d.Protection.RequiredStatusChecks.Checks = append(d.Protection.RequiredStatusChecks.Checks, Check{Context: "spec-gate", AppID: &app})
	return d
}

func TestProtectionApplyMatchingLiveChangesNothing(t *testing.T) {
	f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json"))}
	r := ApplyRepo(dotRepo, declaredLikeLive(t), f.run, false)
	if r.Status != ApplyUnchanged || len(r.Changes) != 0 || len(f.puts) != 0 {
		t.Fatalf("a declaration equal to live must write nothing: %+v, puts=%d", r, len(f.puts))
	}
}

// AC5: the PUT body is complete, the result is re-read, and a second run is a
// no-op. An omitted field is one PUT silently clears (#1451 note 1).
func TestProtectionApplySendsCompleteBodyAndConverges(t *testing.T) {
	f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json")), pulls: mergedPulls(5),
		reported: map[string]string{"sha2": checkRun("spec-gate", 15368)}}
	d := withSpecGate(t)

	r := ApplyRepo(dotRepo, d, f.run, false)
	if r.Status != ApplyApplied || len(f.puts) != 1 {
		t.Fatalf("want one PUT and applied, got %+v (puts=%d)", r, len(f.puts))
	}
	want := []string{"required_status_checks", "required_pull_request_reviews", "enforce_admins", "restrictions",
		"required_linear_history", "allow_force_pushes", "allow_deletions", "block_creations",
		"required_conversation_resolution", "lock_branch", "allow_fork_syncing"}
	for _, k := range want {
		if _, ok := f.puts[0][k]; !ok {
			t.Errorf("PUT body omits %q, which the forge would silently clear", k)
		}
	}
	if len(f.puts[0]) != len(want) {
		t.Errorf("PUT body has %d keys, want exactly %d: %v", len(f.puts[0]), len(want), f.puts[0])
	}
	if strings.Count(strings.Join(f.calls, "\n"), "branches/main/protection\n") < 1 {
		t.Errorf("apply must re-read after writing: %v", f.calls)
	}

	again := ApplyRepo(dotRepo, d, f.run, false)
	if again.Status != ApplyUnchanged || len(f.puts) != 1 {
		t.Fatalf("the second run must report changed=0 and write nothing, got %+v (puts=%d)", again, len(f.puts))
	}
}

// A 200 means the request was accepted, not that the fields took effect.
func TestProtectionApplyFailsWhenTheReReadDisagrees(t *testing.T) {
	f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json")), ignorePut: true, pulls: mergedPulls(1),
		reported: map[string]string{"sha0": checkRun("spec-gate", 15368)}}
	r := ApplyRepo(dotRepo, withSpecGate(t), f.run, false)
	if r.Status != ApplyFailed || !strings.Contains(r.Detail, "required_status_checks.checks") {
		t.Fatalf("an accepted-but-unapplied write must fail naming the field, got %+v", r)
	}
}

// AC5, R-1: with enforce_admins, a required context that never reports makes
// the branch unmergeable for the owner too.
func TestProtectionApplyRefusesUnreportedContext(t *testing.T) {
	cases := map[string]*forgeFake{
		"never reported":                 {pulls: mergedPulls(5)},
		"reported by another app":        {pulls: mergedPulls(5), reported: map[string]string{"sha1": checkRun("spec-gate", 99)}},
		"reported only on an older pull": {pulls: mergedPulls(7), reported: map[string]string{"sha6": checkRun("spec-gate", 15368)}},
		"reported only on an unmerged":   {pulls: mergedPulls(5), reported: map[string]string{"unmerged": checkRun("spec-gate", 15368)}},
	}
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			f.t, f.live = t, string(fixture(t, "get-dotfiles.json"))
			r := ApplyRepo(dotRepo, withSpecGate(t), f.run, false)
			if r.Status != ApplyRefused || !strings.Contains(r.Detail, "spec-gate") || len(f.puts) != 0 {
				t.Fatalf("want refused naming spec-gate with no write, got %+v (puts=%d)", r, len(f.puts))
			}
		})
	}
}

// A commit status counts as a report too: review-attestation is one.
func TestProtectionApplyAcceptsAContextReportedAsAStatus(t *testing.T) {
	f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json")), pulls: mergedPulls(5),
		statuses: map[string]string{"sha4": `{"statuses":[{"context":"spec-gate"}]}`}}
	if r := ApplyRepo(dotRepo, withSpecGate(t), f.run, false); r.Status != ApplyApplied {
		t.Fatalf("a context reported as a commit status is reported, got %+v", r)
	}
}

// Contexts already required are not re-checked: the preflight guards the
// change, not the status quo.
func TestProtectionApplyPreflightsOnlyNewContexts(t *testing.T) {
	f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json")), pulls: mergedPulls(5)}
	d := declaredLikeLive(t)
	d.Protection.RequiredLinearHistory = true
	if r := ApplyRepo(dotRepo, d, f.run, false); r.Status != ApplyApplied {
		t.Fatalf("a change that adds no context needs no report, got %+v", r)
	}
	for _, c := range f.calls {
		if strings.Contains(c, "/pulls?") {
			t.Fatalf("no new context, so no pull request should be read: %v", f.calls)
		}
	}
}

func TestProtectionApplyDryRunWritesNothing(t *testing.T) {
	f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json")), pulls: mergedPulls(5),
		reported: map[string]string{"sha0": checkRun("spec-gate", 15368)}}
	r := ApplyRepo(dotRepo, withSpecGate(t), f.run, true)
	if r.Status != ApplyPlanned || len(r.Changes) != 1 || r.Changes[0].Field != "required_status_checks.checks" {
		t.Fatalf("want a plan naming the one changed field, got %+v", r)
	}
	if len(f.puts) != 0 || len(f.signatures) != 0 {
		t.Fatal("--dry-run must not write")
	}
}

// The dry run runs the preflight too: a plan the real run would refuse is not
// a plan.
func TestProtectionApplyDryRunStillRefuses(t *testing.T) {
	f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json")), pulls: mergedPulls(5)}
	if r := ApplyRepo(dotRepo, withSpecGate(t), f.run, true); r.Status != ApplyRefused {
		t.Fatalf("want refused, got %+v", r)
	}
}

// PUT cannot set required_signatures; it has its own endpoint.
func TestProtectionApplySetsSignaturesThroughTheirEndpoint(t *testing.T) {
	f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json"))}
	d := declaredLikeLive(t)
	d.Protection.RequiredSignatures = true
	f.ignorePut = false
	// The fake's signatures endpoint does not change live, so the re-read
	// disagrees; what this test pins is that the call is made at all.
	r := ApplyRepo(dotRepo, d, f.run, false)
	if len(f.signatures) != 1 || f.signatures[0] != "POST" {
		t.Fatalf("want one POST to required_signatures, got %v (%+v)", f.signatures, r)
	}
}

func TestProtectionApplyProtectsAnUnprotectedBranch(t *testing.T) {
	f := &forgeFake{t: t, pulls: mergedPulls(5)}
	d := declaredLikeLive(t)
	d.Protection.RequiredStatusChecks = nil
	if r := ApplyRepo(dotRepo, d, f.run, false); r.Status != ApplyApplied || len(f.puts) != 1 {
		t.Fatalf("an unprotected branch with a declared object must be protected, got %+v", r)
	}
}

// Apply writes declared protection objects only. Removing protection is never
// a side effect of a sync.
func TestProtectionApplySkipsDeclaredStates(t *testing.T) {
	for _, st := range []string{StateUnavailable, StateUnprotected} {
		f := &forgeFake{t: t, live: string(fixture(t, "get-dotfiles.json"))}
		r := ApplyRepo(dotRepo, RepoDecl{Branch: "main", State: st, Reason: "x"}, f.run, false)
		if r.Status != ApplySkipped || len(f.puts) != 0 || len(f.calls) != 0 {
			t.Fatalf("state %s: want skipped with no call, got %+v, calls %v", st, r, f.calls)
		}
	}
}

func TestProtectionApplyUnanswerableReadFails(t *testing.T) {
	g := &ghFake{}
	if r := ApplyRepo(dotRepo, declaredLikeLive(t), g.run, false); r.Status != ApplyFailed {
		t.Fatalf("a read that could not be answered must fail, never pass, got %+v", r)
	}
}

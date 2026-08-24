package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// declareIdentity points machine.json at a fixture that declares one, so a case
// exercising the dispatch path is not stopped by the identity gate.
//
// It is a fixture rather than the machine's real file for the reason the gate
// exists: a test that passed only on a machine whose owner had configured it
// would be green here and red in CI, and the version that "fixed" that would be
// one that skipped the gate.
func declareIdentity(t *testing.T, deny ...string) {
	t.Helper()
	isolatePATH(t)
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dir := filepath.Join(cfg, "dotfiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	list, err := json.Marshal(deny)
	if err != nil {
		t.Fatalf("marshal deny: %v", err)
	}
	if deny == nil {
		list = []byte("[]")
	}
	body := `{"machine": {"id": "test-machine"}, "pools": {"deny": ` + string(list) + `}}`
	if err := os.WriteFile(filepath.Join(dir, "machine.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("seed machine.json: %v", err)
	}
}

// isolatePATH empties PATH for the duration of a test, so no backend probe can
// find a real harness binary.
//
// Without it these cases dispatch for real the moment --backend stops being
// required: one run reached `claude -p`, took 35 seconds and spent a live
// request. A test that spends quota is not hermetic, and one whose result
// depends on which binaries the machine happens to have is not a test.
func isolatePATH(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

// AC1. This is a stdout contract, so it is tested through captureRealStreams
// rather than execute(): the consumer is a dispatcher reading a pipe, and a
// record written to stderr is an empty string at that call site while looking
// perfectly fine in a merged capture (BUG-070 #915).
func TestAgentRun_WritesOneJSONObjectToStdout(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t)

	stdout, stderr, err := captureRealStreams(t,
		"agent", "run",
		"--role", "reviewer",
		"--task", "review this diff",
		"--tier", "mid",
		"--backend", "dry-run",
		"--timeout", "90s",
		"--repo-root", root,
	)
	if err != nil {
		t.Fatalf("agent run: %v (stderr: %s)", err, stderr)
	}

	var rec struct {
		Status     string `json:"status"`
		Tier       string `json:"tier"`
		Pool       string `json:"pool"`
		Model      string `json:"model"`
		Exit       int    `json:"exit"`
		DurationMS *int64 `json:"duration_ms"`
		Output     string `json:"output"`
	}
	dec := json.NewDecoder(strings.NewReader(stdout))
	if err := dec.Decode(&rec); err != nil {
		t.Fatalf("stdout is not a JSON object: %v\nstdout was: %q", err, stdout)
	}
	// Exactly ONE object: a second value on the stream would make the contract
	// "a JSON stream", which a `| jq .status` consumer does not implement.
	if dec.More() {
		t.Errorf("stdout carries more than one JSON value: %q", stdout)
	}

	if rec.Status != "dry_run" {
		t.Errorf("status = %q, want %q", rec.Status, "dry_run")
	}
	if rec.Tier != "mid" {
		t.Errorf("tier = %q, want %q", rec.Tier, "mid")
	}
	// The mid chain's first entry is claude:sonnet in the shipped map; asserting
	// the resolved route is the point of the record.
	if rec.Pool == "" || rec.Model == "" {
		t.Errorf("record names no route: pool=%q model=%q", rec.Pool, rec.Model)
	}
	// duration_ms must be PRESENT, not merely non-negative. A pointer
	// distinguishes "absent" from "zero"; `>= 0` would be an assertion that
	// cannot fail.
	if rec.DurationMS == nil {
		t.Error("duration_ms is absent from the record")
	}
	if rec.Output == "" {
		t.Error("output is empty; the dry run reports nothing about the route it resolved")
	}
	if strings.Contains(stderr, "{") {
		t.Errorf("stderr carries JSON, so stdout is not the whole contract: %q", stderr)
	}
}

// The refusals are the other half of the machine contract: each one must reach
// stderr and leave stdout EMPTY, because a consumer that parses stdout on a
// failed dispatch would otherwise read a truncated object.
func TestAgentRun_RefusalsLeaveStdoutEmpty(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t)
	base := []string{"agent", "run", "--repo-root", root}

	tests := []struct {
		name    string
		args    []string
		errSubs []string
		// wantExit defaults to 1: a refusal before dispatch must never read as
		// `pool unavailable`. The chain-exhausted case is the one exception and
		// says so.
		wantExit int
	}{
		{
			name:    "no task",
			args:    []string{"--role", "reviewer", "--tier", "mid", "--backend", "dry-run", "--timeout", "1m"},
			errSubs: []string{"--role and --task"},
		},
		{
			name:    "no tier",
			args:    []string{"--role", "r", "--task", "t", "--backend", "dry-run", "--timeout", "1m"},
			errSubs: []string{"--tier is required"},
		},
		{
			name:    "no timeout",
			args:    []string{"--role", "r", "--task", "t", "--tier", "mid", "--backend", "dry-run"},
			errSubs: []string{"--timeout is required", "not eligible"},
		},
		{
			name:    "unknown backend",
			args:    []string{"--role", "r", "--task", "t", "--tier", "mid", "--timeout", "1m", "--backend", "orca"},
			errSubs: []string{"unknown backend", "orca"},
		},
		{
			name:    "undeclared tier",
			args:    []string{"--role", "r", "--task", "t", "--tier", "colossal", "--backend", "dry-run", "--timeout", "1m"},
			errSubs: []string{"colossal"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdout, _, err := captureRealStreams(t, append(base, tc.args...)...)
			if err == nil {
				t.Fatal("expected a refusal, got none")
			}
			if strings.TrimSpace(stdout) != "" {
				t.Errorf("stdout is not empty on a refusal: %q", stdout)
			}
			for _, sub := range tc.errSubs {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("error %q does not name %q", err.Error(), sub)
				}
			}
			want := tc.wantExit
			if want == 0 {
				want = 1
			}
			if got := ExitCode(err); got != want {
				t.Errorf("exit = %d, want %d", got, want)
			}
		})
	}
}

// A map the schema refuses never reaches the walk at all, and that is the
// intended layering: the schema stops a broken chain being written, and the
// dispatcher's own fail-closed branch (agent.Dispatch, covered in that package)
// is the second line for a chain that did not come from a validated map.
//
// What this pins at the command layer is the consequence: no record, empty
// stdout, exit 1 — never exit 3, which a composer would read as "busy, try the
// next pool" and retry against a registry that is broken for every pool.
func TestAgentRun_AMapTheSchemaRefusesStopsBeforeDispatch(t *testing.T) {
	declareIdentity(t)
	root := t.TempDir()
	writeMapFixture(t, root, mapFixtureWithMidChain(`["nan-deepseek-v4-flash"]`))
	copySchemaFixture(t, root)

	stdout, _, err := captureRealStreams(t,
		"agent", "run", "--role", "r", "--task", "t", "--tier", "mid",
		"--backend", "dry-run", "--timeout", "1m", "--repo-root", root,
	)
	if err == nil {
		t.Fatal("a chain entry that names no route was dispatched without complaint")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout is not empty for a dispatch that never happened: %q", stdout)
	}
	if got := ExitCode(err); got != 1 {
		t.Errorf("exit = %d, want 1: a broken registry must not read as a busy pool", got)
	}
}

// The happy path against a FIXTURE map rather than the shipped one, so the
// record's route is an exact expectation. The shipped-map test above proves the
// command works against reality; this one proves it reports the route it was
// given rather than one it invented.
func TestAgentRun_ReportsTheRouteTheMapDeclares(t *testing.T) {
	declareIdentity(t)
	root := t.TempDir()
	writeMapFixture(t, root, mapFixtureWithMidChain(`["nan:mimo-v2.5"]`))
	copySchemaFixture(t, root)

	stdout, _, err := captureRealStreams(t,
		"agent", "run", "--role", "reviewer", "--task", "t", "--tier", "mid",
		"--backend", "dry-run", "--timeout", "1m", "--repo-root", root,
	)
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	var rec struct {
		Pool     string `json:"pool"`
		Model    string `json:"model"`
		Attempts []struct {
			Pool   string `json:"pool"`
			Model  string `json:"model"`
			Status string `json:"status"`
		} `json:"attempts"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
		t.Fatalf("stdout is not a record: %v (%q)", jsonErr, stdout)
	}
	if rec.Pool != "nan" || rec.Model != "mimo-v2.5" {
		t.Errorf("route = %s:%s, want nan:mimo-v2.5", rec.Pool, rec.Model)
	}
	if len(rec.Attempts) != 1 || rec.Attempts[0].Status != "dry_run" {
		t.Errorf("attempts = %+v, want one dry_run attempt", rec.Attempts)
	}
}

func mapFixtureWithMidChain(chain string) string {
	return `{
  "$comment": ["fixture"],
  "version": 1,
  "pools": {"nan": {"auth": "subscription", "probe": "env:NAN_API_KEY"}},
  "harnesses": {"pi": {"pools": ["nan"], "render": "adapter"}},
  "tiers": {"mid": {"pi": "mimo-v2.5"}},
  "chains": {"mid": ` + chain + `},
  "services": {}
}`
}

// copySchemaFixture puts the SHIPPED schema beside the fixture map, so the
// fixture is held to the contract the repository actually enforces. A relaxed
// copy would let a test pass on a map production would reject.
func copySchemaFixture(t *testing.T, root string) {
	t.Helper()
	src := filepath.Join(repoRootForTest(t), harness.ModelMapSchemaFile)
	body, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read shipped schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, harness.ModelMapSchemaFile), body, 0o644); err != nil {
		t.Fatalf("write schema fixture: %v", err)
	}
}

func TestExitCode(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", got)
	}
	if got := ExitCode(errors.New("plain")); got != 1 {
		t.Errorf("ExitCode(untagged) = %d, want 1", got)
	}
	tagged := withExitCode(3, errors.New("no pool could serve"))
	if got := ExitCode(tagged); got != 3 {
		t.Errorf("ExitCode(tagged 3) = %d, want 3", got)
	}
	// The tag must survive wrapping: main() sees whatever cobra hands back.
	if got := ExitCode(errors.Join(errors.New("context"), tagged)); got != 3 {
		t.Errorf("ExitCode(wrapped) = %d, want 3 — the class is lost if it does not survive a wrap", got)
	}
	if !strings.Contains(tagged.Error(), "no pool could serve") {
		t.Errorf("tagged error lost its message: %q", tagged.Error())
	}
}

func writeMapFixture(t *testing.T, root, body string) {
	t.Helper()
	path := filepath.Join(root, "harness")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "model-map.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

// --backend became optional in PR D: the probe chooses. That moves the
// no-backend case out of the refusal table, because it is no longer a refusal
// before dispatch — the walk runs, finds no transport for any entry, and
// reports it. The record still reaches stdout, and the exit code is 3 rather
// than 1: nothing ran, so another machine may well be able to run it.
func TestAgentRun_NoHarnessOnPathExhaustsTheChainRatherThanRefusing(t *testing.T) {
	declareIdentity(t) // also isolates PATH
	root := repoRootForTest(t)

	stdout, _, err := captureRealStreams(t,
		"agent", "run", "--role", "r", "--task", "t", "--tier", "mid",
		"--timeout", "1m", "--repo-root", root,
	)
	if err == nil {
		t.Fatal("a dispatch with no transport for any pool succeeded")
	}
	if got := ExitCode(err); got != 3 {
		t.Errorf("exit = %d, want 3", got)
	}

	var rec struct {
		Status   string `json:"status"`
		Attempts []struct {
			Status string `json:"status"`
		} `json:"attempts"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
		t.Fatalf("no record for a walk that happened: %v (%q)", jsonErr, stdout)
	}
	if rec.Status != "chain_exhausted" {
		t.Errorf("status = %q, want chain_exhausted", rec.Status)
	}
	if len(rec.Attempts) != 3 {
		t.Errorf("attempts = %d, want 3 (the whole mid chain was tried)", len(rec.Attempts))
	}
	// Critically NOT dry_run: an unforced dispatch must never silently resolve
	// to the backend that does nothing.
	if rec.Status == "dry_run" {
		t.Error("the probe selected dry-run")
	}
}

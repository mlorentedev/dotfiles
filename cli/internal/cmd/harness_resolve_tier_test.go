package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// repoRootForTest walks up from the test's working directory to the repo root,
// so the happy path resolves against the map the repository actually ships
// rather than a fixture that could drift away from it.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, harness.ModelMapFile)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find %s walking up from %s", harness.ModelMapFile, dir)
		}
		dir = parent
	}
}

// TestHarnessResolveTier covers the resolution itself against the shipped map.
//
// It uses captureRealStreams rather than execute() deliberately. The sole
// consumer is `scripts/compile-harness.sh`, which reads the model id through a
// shell $(...) — so a value written to stderr would be an empty string at the
// call site, and execute() cannot tell the two apart (BUG-070 #915). This is a
// stdout contract, so it is tested as one.
func TestHarnessResolveTier(t *testing.T) {
	root := repoRootForTest(t)

	tests := []struct {
		name string
		// tier and harness as passed on the command line.
		tier, harnessName string
		wantStdout        string
		// wantErrSubs are substrings the failure must name. A resolution error
		// that does not name both the tier and the harness sends the operator
		// back to the map to guess which half was wrong.
		wantErrSubs []string
	}{
		{
			name:        "the tier the only deployed agent declares",
			tier:        "top",
			harnessName: "claude",
			wantStdout:  "opus",
		},
		{
			name:        "a harness keyed by harness rather than by pool",
			tier:        "mid",
			harnessName: "opencode",
			wantStdout:  "qwen3.6-plus",
		},
		{
			name:        "mid tier for the harness with all three",
			tier:        "mid",
			harnessName: "claude",
			wantStdout:  "sonnet",
		},
		{
			name:        "low tier for the harness with all three",
			tier:        "low",
			harnessName: "claude",
			wantStdout:  "haiku",
		},
		{
			// copilot is declared in `harnesses` and in no tier. Resolving it to
			// anything would render an agent definition naming a model the
			// operator never chose.
			name:        "a declared harness that no tier declares",
			tier:        "top",
			harnessName: "pi", // an adapter: tierless by design (copilot served here until #1170)
			wantErrSubs: []string{"top", "pi"},
		},
		{
			// The `top` tier deliberately has no NaN arm (ADR-032 section 4), so
			// opencode cannot answer it. Failing is the decision, not a gap.
			name:        "a harness declared in one tier asked for another",
			tier:        "top",
			harnessName: "opencode",
			wantErrSubs: []string{"top", "opencode"},
		},
		{
			name:        "a tier the map does not declare at all",
			tier:        "ultra",
			harnessName: "claude",
			wantErrSubs: []string{"ultra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := captureRealStreams(t,
				"harness", "resolve-tier", tt.tier, "--harness", tt.harnessName, "--repo-root", root)

			if len(tt.wantErrSubs) > 0 {
				if err == nil {
					t.Fatalf("expected an error, got stdout %q", stdout)
				}
				// An unresolvable tier must print no model id. A caller doing
				// model="$(dotf harness resolve-tier ...)" without checking $?
				// would otherwise embed whatever leaked onto stdout.
				if strings.TrimSpace(stdout) != "" {
					t.Errorf("failed resolution wrote %q to stdout; it must write nothing", stdout)
				}
				msg := err.Error() + stderr
				for _, sub := range tt.wantErrSubs {
					if !strings.Contains(msg, sub) {
						t.Errorf("error does not name %q: %s", sub, msg)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v (stderr=%q)", err, strings.TrimSpace(stderr))
			}
			if got := strings.TrimSpace(stdout); got != tt.wantStdout {
				t.Errorf("stdout = %q, want %q (stderr=%q)", got, tt.wantStdout, strings.TrimSpace(stderr))
			}
		})
	}
}

// TestHarnessResolveTierFailsLoudWithoutAMap is C15 at the command boundary:
// where the map cannot be read, this errors. It does not fall back to a
// build-time default and it does not resolve to an empty id, because a caller
// that captures an empty string writes an agent definition naming no model —
// the silent degrade the whole registry exists to end.
func TestHarnessResolveTierFailsLoudWithoutAMap(t *testing.T) {
	tests := []struct {
		name string
		// seed prepares a repo root; returning the dir to pass as --repo-root.
		seed func(t *testing.T) string
	}{
		{
			name: "no map at all",
			seed: func(t *testing.T) string { return t.TempDir() },
		},
		{
			name: "a map with no schema beside it",
			seed: func(t *testing.T) string {
				dir := t.TempDir()
				writeFixture(t, dir, harness.ModelMapFile, minimalModelMap)
				return dir
			},
		},
		{
			name: "a map the schema rejects",
			seed: func(t *testing.T) string {
				dir := t.TempDir()
				root := repoRootForTest(t)
				schema, err := os.ReadFile(filepath.Join(root, harness.ModelMapSchemaFile))
				if err != nil {
					t.Fatalf("read shipped schema: %v", err)
				}
				writeFixture(t, dir, harness.ModelMapSchemaFile, string(schema))
				// `chains` names a pool `pools` never declares — the cross-block
				// rule a stock JSON Schema cannot express.
				writeFixture(t, dir, harness.ModelMapFile, ghostPoolModelMap)
				return dir
			},
		},
		{
			name: "a map that is not JSON",
			seed: func(t *testing.T) string {
				dir := t.TempDir()
				writeFixture(t, dir, harness.ModelMapFile, "{ not json")
				writeFixture(t, dir, harness.ModelMapSchemaFile, "{}")
				return dir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.seed(t)
			stdout, stderr, err := captureRealStreams(t,
				"harness", "resolve-tier", "top", "--harness", "claude", "--repo-root", dir)
			if err == nil {
				t.Fatalf("expected an error, got stdout %q", stdout)
			}
			if strings.TrimSpace(stdout) != "" {
				t.Errorf("wrote %q to stdout on an unreadable map; it must write nothing", stdout)
			}
			_ = stderr
		})
	}
}

// TestHarnessHelpListsSubcommands pins the capability probe in
// scripts/compile-harness.sh, which decides whether the dotf on PATH is new
// enough by grepping `dotf harness --help` for a subcommand name.
//
// The probe exists because the exit status cannot answer the question: a dotf
// predating a subcommand rejects its flags with exit 1, which is
// indistinguishable from a genuine refusal (#1158). Renaming one of these
// without updating that grep would make the probe answer "too old" forever,
// silently degrading every agent render — a guard that fails OPEN and reports
// health. This test is the tripwire, and it covers BOTH registries' consumers
// because the shell now probes for each independently.
func TestHarnessHelpListsSubcommands(t *testing.T) {
	stdout, stderr, err := captureRealStreams(t, "harness", "--help")
	if err != nil {
		t.Fatalf("harness --help failed: %v (stderr=%q)", err, stderr)
	}
	out := stdout + stderr
	for _, sub := range []string{"resolve-tier", "resolve-capabilities", "resolve-skills"} {
		// Same shape the shell greps for: the name at the start of its line in
		// the command list, followed by whitespace before its summary.
		if !regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(sub) + `\s`).MatchString(out) {
			t.Errorf("`dotf harness --help` does not list %q as a command.\n"+
				"scripts/compile-harness.sh greps for exactly this to decide whether the "+
				"installed dotf can resolve it; without it every agent renders without that "+
				"field.\ngot:\n%s", sub, out)
		}
	}
}

func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

const minimalModelMap = `{
  "version": 1,
  "pools": {"claude": {"auth": "subscription", "probe": "bin:claude"}},
  "harnesses": {"claude": {"pools": ["claude"], "render": "agent-md", "spawn": "task"}},
  "tiers": {"top": {"claude": "opus"}},
  "chains": {"top": ["claude:opus"]}
}`

// ghostPoolModelMap is type-correct and complete; only the cross-block pool
// reference is wrong, so nothing but checkPoolReferences rejects it.
const ghostPoolModelMap = `{
  "version": 1,
  "pools": {"claude": {"auth": "subscription", "probe": "bin:claude"}},
  "harnesses": {"claude": {"pools": ["claude"], "render": "agent-md", "spawn": "task"}},
  "tiers": {"top": {"claude": "opus"}},
  "chains": {"top": ["ghost:opus"]}
}`

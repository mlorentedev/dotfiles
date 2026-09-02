package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// nextSteps must find the remedy in a FAIL line regardless of which of the
// package's few remedy verbs introduces it, must ignore a diagnostic backtick
// span that names what was checked rather than what to run, must never pick
// up a WARN's remedy (Next steps is for what is actually blocking, i.e. what
// drives the non-zero exit), and must dedupe a command repeated across
// sections.
func TestNextSteps(t *testing.T) {
	transcript := strings.Join([]string{
		"  [FAIL] Bitwarden session is gone (`bw status`: unauthenticated) — run `bw login`, not `bw unlock`",
		"  [FAIL] git 2.54.0 is below the git-for-windows floor 2.55.0 — upgrade with `winget upgrade Git.Git`",
		"  [WARN] git 2.54.0 is below the git-for-windows floor 2.55.0 — upgrade with `winget upgrade Git.Git`",
		"  [FAIL] env-contract.json unreadable — contract checks skipped",
		"  [FAIL] dispatcher not found — run `dotf tools install`",
		"  [FAIL] core.hooksPath unset — run `dotf tools install`",
	}, "\n")

	got := nextSteps(transcript)
	want := []string{"bw login", "winget upgrade Git.Git", "dotf tools install"}
	if len(got) != len(want) {
		t.Fatalf("got %d steps %q, want %d %q", len(got), got, len(want), want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("step %d: got %q, want %q", i, got[i], w)
		}
	}
}

func TestNextSteps_NoFailNoBlock(t *testing.T) {
	if got := nextSteps("  [WARN] upgrade with `winget upgrade Git.Git`\n  [FAIL] no default available (required)"); len(got) != 0 {
		t.Errorf("want no steps when no FAIL line carries a remedy, got %q", got)
	}
}

// End-to-end: a real Run() with a FAIL that carries `bw login` renders the
// Next-steps block after Results:, and a clean run renders neither.
func TestRun_NextStepsBlock(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")
	writeFile(t, filepath.Join(dotfiles, "env-contract.json"),
		`{"env_vars":[],"required_path_entries":{"linux":[]},"required_binaries":[],"optional_binaries":[]}`)
	writeFile(t, filepath.Join(dotfiles, "versions.conf"), "GO_VERSION=1.26.0\n")
	env := map[string]string{"HOME": home, "DOTFILES_DIR": dotfiles}

	t.Run("FAIL with a remedy surfaces it after Results", func(t *testing.T) {
		// Nothing on PATH: opencode missing is one of the FAILs this fixture
		// produces, with a `run \`...\`` remedy — no extra fixture (a populated
		// secrets registry, a real Bitwarden session) needed to exercise the
		// wiring end to end.
		sys := newSys(env, nil, nil)
		var buf bytes.Buffer
		if _, err := Run(Options{Out: &buf, System: sys, StartDir: home, Verbose: true}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
		out := buf.String()
		results := strings.Index(out, "Results:")
		next := strings.Index(out, "Next steps:")
		if results == -1 || next == -1 || next < results {
			t.Fatalf("want Next steps: after Results: in\n%s", out)
		}
		if !strings.Contains(out, "  dotf tools install opencode\n") {
			t.Errorf("want the opencode install remedy listed\n%s", out)
		}
	})

	t.Run("quick mode with no FAIL prints no block", func(t *testing.T) {
		sys := newSys(env, nil, nil)
		var buf bytes.Buffer
		if _, err := Run(Options{Out: &buf, System: sys, StartDir: home, Verbose: true, Quick: true}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
		if strings.Contains(buf.String(), "Next steps:") {
			t.Errorf("want no Next steps: block on a clean run\n%s", buf.String())
		}
	})
}

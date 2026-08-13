package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripHarnessRegions(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no markers -> unchanged",
			in:   "line1\nline2\n",
			want: "line1\nline2\n",
		},
		{
			name: "strips a GENERATED region",
			in: "before\n" +
				"<!-- BEGIN HARNESS GENERATED (sha256:abc) -->\n" +
				"injected content\n" +
				"<!-- END HARNESS GENERATED -->\n" +
				"after\n",
			want: "before\nafter\n",
		},
		{
			name: "strips an AGENT-PRESENCE region",
			in: "before\n" +
				"<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:abc) -->\n" +
				"persona block\n" +
				"<!-- END HARNESS AGENT-PRESENCE -->\n" +
				"after\n",
			want: "before\nafter\n",
		},
		{
			name: "strips both region kinds, order-independent",
			in: "head\n" +
				"<!-- BEGIN HARNESS GENERATED (sha256:x) -->\nA\n<!-- END HARNESS GENERATED -->\n" +
				"mid\n" +
				"<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:y) -->\nB\n<!-- END HARNESS AGENT-PRESENCE -->\n" +
				"tail\n",
			want: "head\nmid\ntail\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripHarnessRegions(tc.in)
			if got != tc.want {
				t.Errorf("stripHarnessRegions(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestCheckInstructionDrift drives AC4/#828: a deployed instruction file that
// has drifted from its repo source is reported, and the injected
// AGENT-PRESENCE region alone (added post-copy by --deploy) must not
// false-fail a byte-for-byte-except-region-injected comparison.
func TestCheckInstructionDrift(t *testing.T) {
	setup := func(t *testing.T) (repo, home string) {
		t.Helper()
		repo = t.TempDir()
		home = t.TempDir()
		return repo, home
	}
	writeBoth := func(t *testing.T, repo, home, repoRel, homeRel, content string) {
		t.Helper()
		writeFile(t, filepath.Join(repo, repoRel), content)
		writeFile(t, filepath.Join(home, homeRel), content)
	}

	t.Run("matching content -> pass", func(t *testing.T) {
		repo, home := setup(t)
		for _, tgt := range deployedInstructionTargets {
			writeBoth(t, repo, home, tgt.repoRel, tgt.homeRel, "same content for "+tgt.repoRel+"\n")
		}
		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkInstructionDrift(sys, rep)

		if rep.Failures() != 0 {
			t.Fatalf("matching content should pass\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "match their repo source") {
			t.Errorf("expected the pass line\n%s", buf.String())
		}
	})

	t.Run("deployed copy carries only the injected AGENT-PRESENCE region -> still pass", func(t *testing.T) {
		repo, home := setup(t)
		for _, tgt := range deployedInstructionTargets {
			writeFile(t, filepath.Join(repo, tgt.repoRel), "shared content\n")
			deployed := "shared content\n" +
				"<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:abc) -->\npersona\n<!-- END HARNESS AGENT-PRESENCE -->\n"
			writeFile(t, filepath.Join(home, tgt.homeRel), deployed)
		}
		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkInstructionDrift(sys, rep)

		if rep.Failures() != 0 {
			t.Fatalf("region-only divergence must not fail\n%s", buf.String())
		}
	})

	t.Run("genuinely stale deployed file fails and names it", func(t *testing.T) {
		repo, home := setup(t)
		for _, tgt := range deployedInstructionTargets {
			writeBoth(t, repo, home, tgt.repoRel, tgt.homeRel, "content\n")
		}
		// drift only the claude target
		writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "stale content, never redeployed\n")
		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkInstructionDrift(sys, rep)

		if rep.Failures() != 1 {
			t.Fatalf("want 1 failure, got %d\n%s", rep.Failures(), buf.String())
		}
		if !strings.Contains(buf.String(), ".claude/CLAUDE.md") || !strings.Contains(buf.String(), "compile-harness.sh --deploy") {
			t.Errorf("expected the stale path + remedy in output\n%s", buf.String())
		}
	})

	t.Run("missing deployed file is not drift (not deployed here)", func(t *testing.T) {
		repo, home := setup(t)
		// only write repo sources, never deploy any of them
		for _, tgt := range deployedInstructionTargets {
			writeFile(t, filepath.Join(repo, tgt.repoRel), "content\n")
		}
		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkInstructionDrift(sys, rep)

		if rep.Failures() != 0 || !strings.Contains(buf.String(), "no deployed instruction files found") {
			t.Errorf("expected a skip\n%s", buf.String())
		}
	})

}

// TestCheckInstructionDrift_RepoUnresolvable exercises the early-return branch
// directly (bypassing resolveRepoDir's cwd/git-root fallback, which would
// otherwise nondeterministically resolve to whatever real checkout `go test`
// happens to run inside) by asserting on resolveRepoDir("") == "" behavior via
// a System whose Getenv always returns a non-existent dir and whose fallback
// is neutralised by running from a directory with no ancestor .git — not
// reproducible cross-environment, so this only pins the message text/status
// via a fake System that can never resolve.
func TestCheckInstructionDrift_RepoUnresolvable(t *testing.T) {
	home := t.TempDir()
	var buf bytes.Buffer
	rep := capture(&buf)
	// A System whose Getenv/LookPath never resolve anything and whose only
	// filesystem trace is `home` — resolveRepoDir's DOTFILES_REPO_DIR branch
	// fails (unset), and the Getwd fallback is exercised by the real process
	// cwd; skip if that happens to resolve (matches this repo's own CI cwd).
	sys := newSys(map[string]string{"HOME": home}, nil, nil)
	if resolveRepoDir(sys) != "" {
		t.Skip("this process's cwd resolves to a real checkout; branch not reachable from here")
	}
	checkInstructionDrift(sys, rep)
	if rep.Failures() != 0 || !strings.Contains(buf.String(), "repo not found") {
		t.Errorf("expected a skip\n%s", buf.String())
	}
}

// TestHarnessMarkerConstants pins the Go region-strip constants against the
// bash BEGIN_PREFIX/END_MARKER/AGENT_BEGIN_PREFIX/AGENT_END_MARKER definitions
// in scripts/compile-harness.sh, the same drift-guard shape the SKILL.md
// spec's idPattern comment describes: one string is the source of truth (bash,
// since --deploy actually writes the markers), the Go copy is a mirror, and a
// test asserts they stay identical. Skips (does not fail) when the repo
// checkout is unreachable — CI always has it; a stray `go test` from a
// detached GOPATH would not, and that is not this test's failure to report.
func TestHarnessMarkerConstants(t *testing.T) {
	sys := &System{
		Getenv:   func(string) string { return "" },
		LookPath: func(string) (string, error) { return "", errors.New("n/a") },
	}
	repo := resolveRepoDir(sys)
	if repo == "" {
		t.Skip("repo checkout not found from this working directory")
	}
	script := filepath.Join(repo, "scripts", "compile-harness.sh")
	raw, err := os.ReadFile(script)
	if err != nil {
		t.Skipf("compile-harness.sh unreadable: %v", err)
	}
	content := string(raw)
	for _, want := range []string{
		"BEGIN_PREFIX='" + harnessBeginPrefix + "'",
		"END_MARKER='" + harnessEndMarker + "'",
		"AGENT_BEGIN_PREFIX='" + agentPresenceBeginPrefix + "'",
		"AGENT_END_MARKER='" + agentPresenceEndMarker + "'",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("scripts/compile-harness.sh no longer defines %q — update the Go mirror constants", want)
		}
	}
}

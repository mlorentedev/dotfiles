package doctor

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
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
		{
			// inject_agent_presence / replace_region's append branch write a
			// region as "\n" + BEGIN + body + END + "\n" onto untouched
			// content -- the blank line right before BEGIN must not survive
			// the strip, or a freshly-appended region reads as drift forever.
			name: "drops the blank separator line an appended region leaves behind",
			in:   "shared content\n" + "\n<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:abc) -->\npersona\n<!-- END HARNESS AGENT-PRESENCE -->\n",
			want: "shared content\n",
		},
		{
			name: "a genuine blank line NOT before a region is preserved",
			in:   "para one\n\npara two\n",
			want: "para one\n\npara two\n",
		},
		{
			// A deployed copy written CRLF by a Windows tool (WIN-008/#1289):
			// the END marker must still close the region — a "\r"-suffixed
			// marker used to leave skip mode on to EOF — and no "\r" survives
			// into the comparison.
			name: "CRLF input: the region closes and line endings normalise",
			in:   "before\r\n<!-- BEGIN HARNESS GENERATED (sha256:abc) -->\r\ninjected\r\n<!-- END HARNESS GENERATED -->\r\nafter\r\n",
			want: "before\nafter\n",
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

	t.Run("matching content -> pass (copilot absent, 3 of 4 checked)", func(t *testing.T) {
		repo, home := setup(t)
		for _, tgt := range deployedInstructionTargets {
			writeBoth(t, repo, home, tgt.repoRel, tgt.homeRel, "same content for "+tgt.repoRel+"\n")
		}
		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil) // "copilot" NOT on PATH

		var buf bytes.Buffer
		rep := capture(&buf)
		checkInstructionDrift(sys, rep)

		if rep.Failures() != 0 {
			t.Fatalf("matching content should pass\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "match their repo source (3 checked)") {
			t.Errorf("expected the 3-checked pass line (copilot gated out)\n%s", buf.String())
		}
	})

	t.Run("matching content -> pass (copilot present, all 4 checked)", func(t *testing.T) {
		repo, home := setup(t)
		for _, tgt := range deployedInstructionTargets {
			writeBoth(t, repo, home, tgt.repoRel, tgt.homeRel, "same content for "+tgt.repoRel+"\n")
		}
		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, []string{"copilot"}, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkInstructionDrift(sys, rep)

		if rep.Failures() != 0 {
			t.Fatalf("matching content should pass\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "match their repo source (4 checked)") {
			t.Errorf("expected the 4-checked pass line (copilot on PATH)\n%s", buf.String())
		}
	})

	t.Run("copilot present but genuinely stale -> fails", func(t *testing.T) {
		repo, home := setup(t)
		for _, tgt := range deployedInstructionTargets {
			writeBoth(t, repo, home, tgt.repoRel, tgt.homeRel, "content\n")
		}
		writeFile(t, filepath.Join(home, ".copilot", "copilot-instructions.md"), "stale copilot content\n")
		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, []string{"copilot"}, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkInstructionDrift(sys, rep)

		if rep.Failures() != 1 || !strings.Contains(buf.String(), ".copilot/copilot-instructions.md") {
			t.Fatalf("stale copilot file with copilot present must fail and name it\n%s", buf.String())
		}
	})

	t.Run("deployed copy carries only the injected AGENT-PRESENCE region -> still pass", func(t *testing.T) {
		repo, home := setup(t)
		for _, tgt := range deployedInstructionTargets {
			writeFile(t, filepath.Join(repo, tgt.repoRel), "shared content\n")
			// Faithful to inject_agent_presence's actual append shape
			// (compile-harness.sh): "\n" + BEGIN + body + END + "\n" tacked
			// onto the untouched source — including the blank separator line,
			// which is exactly what stripHarnessRegions must also drop.
			deployed := "shared content\n" +
				"\n<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:abc) -->\npersona\n<!-- END HARNESS AGENT-PRESENCE -->\n"
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

	t.Run("gated target with an absent command is never drift, even if the stale file exists", func(t *testing.T) {
		repo, home := setup(t)
		for _, tgt := range deployedInstructionTargets {
			writeFile(t, filepath.Join(repo, tgt.repoRel), "content\n")
		}
		// copilot-instructions.md exists at $HOME and genuinely differs from the
		// repo source (a leftover from before copilot was uninstalled, say) --
		// but `copilot` is not on PATH, so --deploy would never have touched it
		// and this must not be reported as drift (it would be a FAIL no remedy
		// could ever clear).
		writeFile(t, filepath.Join(home, ".copilot", "copilot-instructions.md"), "very stale, copilot not installed\n")
		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil) // "copilot" deliberately not on PATH

		var buf bytes.Buffer
		rep := capture(&buf)
		checkInstructionDrift(sys, rep)

		if rep.Failures() != 0 {
			t.Fatalf("a gated target with its command absent must never fail\n%s", buf.String())
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
	} {
		if !strings.Contains(content, want) {
			t.Errorf("scripts/compile-harness.sh no longer defines %q — update the Go mirror constants", want)
		}
	}
	// The AGENT-PRESENCE pair moved to the harness package with HARNESS-092
	// (#1326): doctor must read the package's constants, and the shell must
	// not grow a copy back — two spellings of one marker is the drift class.
	if agentPresenceBeginPrefix != harness.PresenceBeginPrefix || agentPresenceEndMarker != harness.PresenceEndMarker {
		t.Errorf("doctor's AGENT-PRESENCE markers differ from the harness package's")
	}
	for _, stale := range []string{"AGENT_BEGIN_PREFIX=", "AGENT_END_MARKER="} {
		if strings.Contains(content, stale) {
			t.Errorf("scripts/compile-harness.sh defines %s again; the presence markers live in cli/internal/harness", stale)
		}
	}
}

// manifestPresenceEntry is the subset of harness/manifest.json's
// agents.presence[] shape TestCheckInstructionDrift_MatchesManifest needs.
type manifestPresenceEntry struct {
	Agent           string `json:"agent"`
	File            string `json:"file"`
	Source          string `json:"source"`
	RequiresCommand string `json:"requires_command"`
}

// TestCheckInstructionDrift_MatchesManifest is the drift guard
// deployedInstructionTargets' doc comment promises: harness/manifest.json's
// agents.presence[] is the actual SSOT compile-harness.sh's deploy_instructions
// reads at deploy time, and this Go list is a hand-kept mirror the doctor
// binary uses (it must run with no repo present). If the two diverge, doctor
// silently checks the wrong thing — this test makes that loud instead. Skips
// (does not fail) when the repo checkout is unreachable, same shape as
// TestHarnessMarkerConstants.
func TestCheckInstructionDrift_MatchesManifest(t *testing.T) {
	sys := &System{
		Getenv:   func(string) string { return "" },
		LookPath: func(string) (string, error) { return "", errors.New("n/a") },
	}
	repo := resolveRepoDir(sys)
	if repo == "" {
		t.Skip("repo checkout not found from this working directory")
	}
	raw, err := os.ReadFile(filepath.Join(repo, "harness", "manifest.json"))
	if err != nil {
		t.Skipf("harness/manifest.json unreadable: %v", err)
	}
	var m struct {
		Agents struct {
			Presence []manifestPresenceEntry `json:"presence"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("harness/manifest.json invalid JSON: %v", err)
	}
	if len(m.Agents.Presence) == 0 {
		t.Fatal("manifest agents.presence is empty — did the schema change under this test?")
	}

	byFile := make(map[string]manifestPresenceEntry, len(m.Agents.Presence))
	for _, e := range m.Agents.Presence {
		byFile[e.File] = e
	}
	if len(byFile) != len(deployedInstructionTargets) {
		t.Fatalf("deployedInstructionTargets has %d entries, manifest agents.presence has %d — keep them in sync",
			len(deployedInstructionTargets), len(byFile))
	}
	for _, tgt := range deployedInstructionTargets {
		e, ok := byFile[tgt.homeRel]
		if !ok {
			t.Errorf("deployedInstructionTargets has %q but manifest agents.presence does not", tgt.homeRel)
			continue
		}
		if e.Source != tgt.repoRel {
			t.Errorf("%s: Go repoRel=%q, manifest source=%q — keep deployedInstructionTargets in sync", tgt.homeRel, tgt.repoRel, e.Source)
		}
		if e.RequiresCommand != tgt.requiresCommand {
			t.Errorf("%s: Go requiresCommand=%q, manifest requires_command=%q — keep deployedInstructionTargets in sync",
				tgt.homeRel, tgt.requiresCommand, e.RequiresCommand)
		}
	}
}

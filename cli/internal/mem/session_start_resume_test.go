package mem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// resumeVault lays out a vault whose project memory holds one stored thread,
// wt-x, written by claude, and returns the vault and a checkout directory
// that keys to that thread and project.
func resumeVault(t *testing.T) (vaultDir, cwd string) {
	t.Helper()
	vaultDir = t.TempDir()
	cwd = filepath.Join(t.TempDir(), "proj-wt-x")
	mustMkdirAll(t, cwd)
	memory, _ := ProjectMemory(vaultDir, cwd)
	body, err := os.ReadFile(filepath.Join("testdata", "resume", "thread.md"))
	if err != nil {
		t.Fatal(err)
	}
	// The fixture is a whole block; the store takes the body under the heading.
	_, rest, _ := strings.Cut(string(body), "\n")
	res, err := StoreThreadAs("# M\n\n## Session Handoff\n", memory, "wt-x", "claude", strings.TrimLeft(rest, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	mustMkdirAll(t, filepath.Dir(res.File))
	mustWrite(t, res.File, res.FileBody)
	mustWrite(t, memory, res.Content)
	return vaultDir, cwd
}

// AC3: a session that lost its conversation (startup, clear, compact) begins
// with the full resume block; one that kept it (resume, fork) gets one line.
// Each source's output is pinned by a golden file.
func TestSessionStartResumeBySource(t *testing.T) {
	vaultDir, cwd := resumeVault(t)
	memory, _ := ProjectMemory(vaultDir, cwd)
	for _, source := range []string{"startup", "clear", "compact", "resume", "fork"} {
		got := sessionResume(source, memory, ThreadKey(cwd), "claude")
		// The source path is a temp dir; pin the golden on the part under it.
		got = filepath.ToSlash(strings.ReplaceAll(got, filepath.Dir(memory), "<memory>"))
		golden := filepath.Join("testdata", "session_start_resume", source+".golden")
		if *updateGolden {
			mustMkdirAll(t, filepath.Dir(golden))
			mustWrite(t, golden, got)
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("source %s differs from %s:\n%s", source, golden, got)
		}
	}

	ctx := ClaudeContext(ClaudeContextInput{
		Cwd: cwd, Source: "compact", Vault: vaultDir, Home: t.TempDir(),
		Now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
	})
	if !strings.HasPrefix(ctx, "## Resume: thread wt-x\n") {
		t.Errorf("a compacted session must begin with the resume block, got: %q", ctx[:min(120, len(ctx))])
	}
	if !strings.Contains(ctx, "\n"+sddReminder) {
		t.Errorf("the rest of the brief must follow the resume block")
	}
}

// AC3: another agent's thread, or no memory at all, adds nothing.
func TestSessionStartResumeInjectsNothingThatIsNotThisAgents(t *testing.T) {
	vaultDir, cwd := resumeVault(t)
	memory, _ := ProjectMemory(vaultDir, cwd)
	if got := sessionResume("compact", memory, ThreadKey(cwd), "pi"); got != "" {
		t.Errorf("pi must not be handed claude's thread, got:\n%s", got)
	}
	if got := sessionResume("compact", filepath.Join(t.TempDir(), "MEMORY.md"), "wt-x", "claude"); got != "" {
		t.Errorf("an absent memory must add nothing, got:\n%s", got)
	}
}

// AC3: the one line a resumed or forked session gets carries a command the
// operator copies, so a thread key from MEMORY.md is one shell word in it
// (MEMORY-008 review round 5). The plain key stays bare in the goldens above.
func TestSessionStartResumeQuotesTheOneLiner(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	mustWrite(t, memory, "# M\n\n## Session Handoff\n\n### thread: wt-x;echo (writer: claude)\n\n**Next action:** a.\n")
	for _, source := range []string{"resume", "fork"} {
		got := sessionResume(source, memory, "wt-x;echo", "claude")
		if want := "`dotf mem resume --thread='wt-x;echo'`"; !strings.Contains(got, want) {
			t.Errorf("source %s: want %s in %q", source, want, got)
		}
	}
}

// A dash-leading key, quoted or not, reaches cobra as a flag; the = form keeps
// it the value of --thread (#3002).
func TestSessionStartResumeBindsADashLeadingKey(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	mustWrite(t, memory, "# M\n\n## Session Handoff\n\n### thread: -x (writer: claude)\n\n**Next action:** a.\n")
	for _, source := range []string{"resume", "fork"} {
		got := sessionResume(source, memory, "-x", "claude")
		if want := "`dotf mem resume --thread=-x`"; !strings.Contains(got, want) {
			t.Errorf("source %s: want %s in %q", source, want, got)
		}
	}
}

// AC3: the handoff_resume injector flag turns the block off.
func TestSessionStartResumeCanBeTurnedOff(t *testing.T) {
	vaultDir, cwd := resumeVault(t)
	cfg := filepath.Join(t.TempDir(), "session-start-config.json")
	mustWrite(t, cfg, `{"injectors": {"handoff_resume": {"enabled": false}}}`)
	ctx := ClaudeContext(ClaudeContextInput{
		Cwd: cwd, Source: "compact", Vault: vaultDir, Home: t.TempDir(), ConfigPath: cfg,
		Now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
	})
	if strings.Contains(ctx, "## Resume:") || strings.Contains(ctx, "[resume]") {
		t.Errorf("a disabled injector must add nothing, got: %q", ctx[:min(120, len(ctx))])
	}
}

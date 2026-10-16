package worktree

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func touch(t *testing.T, path string, age time.Duration) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := time.Now().Add(-age)
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

// fakePiSession writes a pi session for cwd under sessions, in the directory
// name pi uses (measured on the Mac, 2026-10-16: /Users/manu/Projects/dotfiles
// is --Users-manu-Projects-dotfiles--).
func fakePiSession(t *testing.T, sessions, cwd string, age time.Duration) {
	t.Helper()
	dir := "--" + strings.ReplaceAll(strings.TrimPrefix(filepath.ToSlash(cwd), "/"), "/", "-") + "--"
	touch(t, filepath.Join(sessions, dir, "2026-10-16T01-00-00-000Z_0b6c.jsonl"), age)
}

// fakeCopilotSession writes a Copilot session-state entry whose workspace.yaml
// names cwd, as Copilot CLI writes it (measured on the Mac, 2026-10-16). The
// workspace.yaml is a day old: a session's age is its events.jsonl.
func fakeCopilotSession(t *testing.T, state, cwd string, age time.Duration) {
	t.Helper()
	dir := filepath.Join(state, "4da4dcaf-f324-4d84-9183-65d8b49a068e")
	touch(t, filepath.Join(dir, "events.jsonl"), age)
	ws := "id: 4da4dcaf-f324-4d84-9183-65d8b49a068e\ncwd: " + cwd + "\nclient_name: github/cli\n"
	if err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(ws), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "workspace.yaml"), mod, mod); err != nil {
		t.Fatal(err)
	}
}

// #2569: a stopped pi or Copilot session resumes from its directory, as Claude
// Code's does, so removing the worktree strands it the same way.
func TestRecentHarnessSession(t *testing.T) {
	wt := filepath.Join(t.TempDir(), "repo", "feat-x")
	for _, tc := range []struct {
		name    string
		write   func(t *testing.T, st SessionStores)
		harness string // "" means none found
	}{
		{"a pi session in the worktree", func(t *testing.T, st SessionStores) { fakePiSession(t, st.PiSessions, wt, time.Hour) }, "pi"},
		{"a pi session in a subdirectory", func(t *testing.T, st SessionStores) {
			fakePiSession(t, st.PiSessions, filepath.Join(wt, "cli"), time.Hour)
		}, "pi"},
		{"a pi session older than the window", func(t *testing.T, st SessionStores) { fakePiSession(t, st.PiSessions, wt, 7*time.Hour) }, ""},
		{"another worktree's pi session", func(t *testing.T, st SessionStores) {
			fakePiSession(t, st.PiSessions, filepath.Join(filepath.Dir(wt), "feat-y"), time.Hour)
		}, ""},
		{"a Copilot session in a subdirectory", func(t *testing.T, st SessionStores) {
			fakeCopilotSession(t, st.CopilotState, filepath.Join(wt, "cli"), time.Hour)
		}, "Copilot"},
		{"a Copilot session older than the window", func(t *testing.T, st SessionStores) {
			fakeCopilotSession(t, st.CopilotState, wt, 7*time.Hour)
		}, ""},
		{"a Copilot session in a sibling sharing the prefix", func(t *testing.T, st SessionStores) {
			fakeCopilotSession(t, st.CopilotState, wt+"-2", time.Hour)
		}, ""},
		{"a Claude Code session", func(t *testing.T, st SessionStores) { fakeTranscript(t, st.ClaudeProjects, wt, time.Hour) }, "Claude Code"},
		{"no store holds anything", func(*testing.T, SessionStores) {}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.harness == "pi" || strings.Contains(tc.name, "pi session") {
				skipPiOffPOSIX(t)
			}
			home := t.TempDir()
			st := SessionStores{
				ClaudeProjects: filepath.Join(home, ".claude", "projects"),
				PiSessions:     filepath.Join(home, ".pi", "agent", "sessions"),
				CopilotState:   filepath.Join(home, ".copilot", "session-state"),
			}
			tc.write(t, st)
			s, found, err := RecentHarnessSession(st, wt, time.Now(), SessionWindow)
			if err != nil {
				t.Fatal(err)
			}
			if got := map[bool]string{true: s.Harness, false: ""}[found]; got != tc.harness {
				t.Errorf("harness = %q (%+v), want %q", got, s, tc.harness)
			}
		})
	}
}

// A store that exists and cannot be read must not read as one with no session
// in it; the error names the harness. A store that is absent is a harness that
// is not installed: nothing to consult.
func TestRecentHarnessSessionFailsClosedOnAnUnreadableStore(t *testing.T) {
	wt := filepath.Join(t.TempDir(), "feat-x")
	home := t.TempDir()
	st := SessionStores{CopilotState: filepath.Join(home, "session-state")}
	if err := os.MkdirAll(filepath.Join(st.CopilotState, "s1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(st.CopilotState, "s1", "workspace.yaml"), []byte("cwd: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecentHarnessSession(st, wt, time.Now(), SessionWindow); err == nil || !strings.Contains(err.Error(), "Copilot") {
		t.Errorf("want an error naming Copilot, got %v", err)
	}

	if runtime.GOOS == "windows" {
		return // the pi row keys no Windows path, so it never reads its store there
	}
	st = SessionStores{PiSessions: filepath.Join(home, "sessions-file")}
	if err := os.WriteFile(st.PiSessions, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecentHarnessSession(st, wt, time.Now(), SessionWindow); err == nil || !strings.Contains(err.Error(), "pi") {
		t.Errorf("a store that is a file is unreadable, want an error naming pi, got %v", err)
	}

	absent := SessionStores{PiSessions: filepath.Join(home, "none"), CopilotState: filepath.Join(home, "none2")}
	if _, found, err := RecentHarnessSession(absent, wt, time.Now(), SessionWindow); err != nil || found {
		t.Errorf("absent stores: found=%v err=%v, want none", found, err)
	}
}

// skipPiOffPOSIX: the pi row keys only an absolute POSIX path, so a pi
// fixture under a Windows temp directory has no name to be found by.
func skipPiOffPOSIX(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("pi's directory name for a Windows path is unmeasured; the row consults nothing there")
	}
}

// pi's directory name is measured for an absolute POSIX path only. A Windows
// path is not guessed at: the pi row consults nothing for it.
func TestPiSessionKeysOnlyForAPOSIXPath(t *testing.T) {
	if keys := piSessionKeys(`C:\Users\u\Projects\worktrees\dotfiles\feat-x`); len(keys) != 0 {
		t.Errorf("keys = %v, want none for a Windows path", keys)
	}
	if keys := piSessionKeys("/Users/manu/Projects/dotfiles"); len(keys) == 0 || keys[0] != "--Users-manu-Projects-dotfiles--" {
		t.Errorf("keys = %v, want the measured name first", keys)
	}
}

func TestSessionSkipReasonNamesTheHarness(t *testing.T) {
	wt := filepath.Join(t.TempDir(), "feat-x")
	home := t.TempDir()
	st := SessionStores{CopilotState: filepath.Join(home, "session-state")}
	fakeCopilotSession(t, st.CopilotState, wt, 2*time.Hour)
	if r := sessionSkipReason(st, wt, time.Now()); !strings.HasPrefix(r, "a Copilot session wrote ") {
		t.Errorf("reason = %q, want it to name Copilot", r)
	}
	if err := checkNoRecentSession(st, wt, time.Now()); err == nil || !strings.Contains(err.Error(), "a Copilot session") || !strings.Contains(err.Error(), "--ignore-recent-session") {
		t.Errorf("done's refusal = %v, want it to name Copilot and the override", err)
	}
}

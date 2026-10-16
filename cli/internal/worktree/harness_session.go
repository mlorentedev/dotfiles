package worktree

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// SessionStores are the on-disk session stores of the harnesses whose stopped
// sessions resume from their directory (#2569). An empty field, or a store that
// does not exist, is a harness that is not installed: nothing to consult.
//
// Not every harness is here. agy keeps its conversations in SQLite blobs with
// no readable cwd, and codex and opencode have no store on the boxes measured;
// a live session of any of them is still caught by the process check.
type SessionStores struct {
	ClaudeProjects string // <CLAUDE_CONFIG_DIR>/projects
	PiSessions     string // ~/.pi/agent/sessions
	CopilotState   string // ~/.copilot/session-state
}

// HarnessSession is a recent session and the harness that wrote it.
type HarnessSession struct {
	Harness string
	Session
}

type sessionRow struct {
	harness string
	find    func(st SessionStores, wt string) (Session, error)
}

var sessionRows = []sessionRow{
	{"Claude Code", func(st SessionStores, wt string) (Session, error) { return claudeSession(st.ClaudeProjects, wt) }},
	{"pi", func(st SessionStores, wt string) (Session, error) { return piSession(st.PiSessions, wt) }},
	{"Copilot", func(st SessionStores, wt string) (Session, error) { return copilotSession(st.CopilotState, wt) }},
}

// RecentHarnessSession reports the first harness with a session started in wt,
// or below it, whose store was written within window of now. A store that
// exists and cannot be read is an error naming the harness: it must not read
// as a store with no session in it.
func RecentHarnessSession(st SessionStores, wt string, now time.Time, window time.Duration) (HarnessSession, bool, error) {
	for _, row := range sessionRows {
		s, err := row.find(st, wt)
		if err != nil {
			return HarnessSession{}, false, fmt.Errorf("%s sessions: %w", row.harness, err)
		}
		if s.Transcript != "" && now.Sub(s.Modified) < window {
			return HarnessSession{Harness: row.harness, Session: s}, true, nil
		}
	}
	return HarnessSession{}, false, nil
}

// readStore lists a store directory. A missing one is (nil, nil); absence is
// decided by Stat, since on Windows reading a file as a directory maps to
// ErrNotExist too.
func readStore(dir string) ([]os.DirEntry, error) {
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if _, statErr := os.Stat(dir); errors.Is(statErr, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return entries, nil
}

// piSessionKeys are the directory names pi gives a session started in wt:
// "--" + the path without its leading "/", with "/" replaced by "-", + "--"
// (measured on macOS, 2026-10-16). Only an absolute POSIX path is keyed; pi's
// name for a Windows path is unmeasured and not guessed at.
func piSessionKeys(wt string) []string {
	if !strings.HasPrefix(wt, "/") {
		return nil
	}
	var keys []string
	for _, p := range []string{wt, resolvedPath(wt)} {
		if !strings.HasPrefix(p, "/") {
			continue
		}
		k := "--" + strings.ReplaceAll(strings.TrimPrefix(p, "/"), "/", "-") + "--"
		if len(keys) == 0 || keys[0] != k {
			keys = append(keys, k)
		}
	}
	return keys
}

// piSession is the newest pi session file for wt or a directory below it,
// whose name extends wt's stem by "-". As with Claude Code's keys, that also
// matches a sibling such as feat-x-y for feat-x: a false refusal, the safe side.
func piSession(sessions, wt string) (Session, error) {
	keys := piSessionKeys(wt)
	if len(keys) == 0 {
		return Session{}, nil
	}
	entries, err := readStore(sessions)
	if err != nil {
		return Session{}, err
	}
	var newest Session
	for _, e := range entries {
		if !e.IsDir() || !matchesPiKey(e.Name(), keys) {
			continue
		}
		s, err := newestTranscript(filepath.Join(sessions, e.Name()))
		if err != nil {
			return Session{}, err
		}
		if s.Modified.After(newest.Modified) {
			newest = s
		}
	}
	return newest, nil
}

func matchesPiKey(name string, keys []string) bool {
	for _, k := range keys {
		stem := strings.TrimSuffix(k, "--")
		if name == k || (strings.HasPrefix(name, stem+"-") && strings.HasSuffix(name, "--")) {
			return true
		}
	}
	return false
}

// copilotSession is the newest Copilot CLI session whose workspace.yaml names
// a cwd inside wt. Copilot keys its store by session id, not by directory, so
// every session's workspace.yaml is read; its events.jsonl is the transcript.
func copilotSession(state, wt string) (Session, error) {
	entries, err := readStore(state)
	if err != nil {
		return Session{}, err
	}
	roots := []string{wt, resolvedPath(wt)}
	var newest Session
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(state, e.Name())
		cwd, err := copilotCwd(filepath.Join(dir, "workspace.yaml"))
		if err != nil {
			return Session{}, err
		}
		if cwd == "" || !withinAny([]string{cwd, resolvedPath(cwd)}, roots) {
			continue
		}
		for _, f := range []string{"events.jsonl", "workspace.yaml"} {
			info, err := os.Stat(filepath.Join(dir, f))
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return Session{}, err
			}
			if info.ModTime().After(newest.Modified) {
				newest = Session{Transcript: filepath.Join(dir, f), Modified: info.ModTime()}
			}
		}
	}
	return newest, nil
}

// copilotCwd is "" for a session directory with no workspace.yaml, which is a
// session being created or removed, not a store that cannot be read.
func copilotCwd(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var ws struct {
		Cwd string `yaml:"cwd"`
	}
	if err := yaml.Unmarshal(raw, &ws); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	return ws.Cwd, nil
}

func withinAny(paths, roots []string) bool {
	for _, p := range paths {
		for _, r := range roots {
			if pathWithin(p, r) {
				return true
			}
		}
	}
	return false
}

// checkNoRecentSession is done's form of the check: a refusal naming the
// harness, the transcript and the override.
func checkNoRecentSession(st SessionStores, absWT string, now time.Time) error {
	s, live, err := RecentHarnessSession(st, absWT, now, SessionWindow)
	if err != nil {
		return fmt.Errorf("cannot tell whether a session is using %s: %w. Fix the read, or pass --ignore-recent-session", absWT, err)
	}
	if !live {
		return nil
	}
	return fmt.Errorf("a %s session for %s wrote %s %s ago; removing the worktree would strand it. End or resume it elsewhere, or pass --ignore-recent-session",
		s.Harness, absWT, s.Transcript, now.Sub(s.Modified).Round(time.Minute))
}

// sessionSkipReason is why sweep or migrate must leave wt alone for a session,
// or "". It names no override, since neither command has one.
func sessionSkipReason(st SessionStores, wt string, now time.Time) string {
	s, live, err := RecentHarnessSession(st, wt, now, SessionWindow)
	if err != nil {
		return "sessions could not be inspected: " + err.Error()
	}
	if live {
		return fmt.Sprintf("a %s session wrote %s %s ago", s.Harness, s.Transcript, now.Sub(s.Modified).Round(time.Minute))
	}
	return ""
}

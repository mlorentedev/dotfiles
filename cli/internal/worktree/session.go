package worktree

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

// SessionWindow is how recently a transcript must have been written for its
// session to count as live (#1653 AC1). A session idle for longer can still be
// resumed, but by then removing its worktree is the operator's call.
const SessionWindow = 6 * time.Hour

// Session is the newest transcript RecentSession found.
type Session struct {
	Transcript string
	Modified   time.Time
}

// RecentSession reports a Claude Code session started in wt, or in a
// directory below it, whose transcript was written within window of now.
//
// projectsDir is <CLAUDE_CONFIG_DIR>/projects; "" or a missing directory means
// none. Any other read failure is returned: a store that could not be
// inspected must not read as one with no session in it. A session started in wt/cli has the key of wt plus "-cli", so a project
// directory whose name extends wt's key by "-" counts too. That also matches a
// sibling such as feat-x-y for wt feat-x: a false refusal, the safe side, and
// the message names the transcript so the operator can tell.
//
// This is the Claude-only layer over checkCallerOutside's process check, which
// covers every harness but only while a process is alive.
func RecentSession(projectsDir, wt string, now time.Time, window time.Duration) (Session, bool, error) {
	if projectsDir == "" {
		return Session{}, false, nil
	}
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		// Absence is decided by Stat, not by ReadDir's error: on Windows,
		// reading a file as a directory maps to ErrNotExist too.
		if _, statErr := os.Stat(projectsDir); errors.Is(statErr, fs.ErrNotExist) {
			return Session{}, false, nil
		}
		return Session{}, false, err
	}
	keys := sessionKeys(wt)
	var newest Session
	for _, e := range entries {
		if !e.IsDir() || !matchesKey(e.Name(), keys) {
			continue
		}
		s, err := newestTranscript(filepath.Join(projectsDir, e.Name()))
		if err != nil {
			return Session{}, false, err
		}
		if s.Modified.After(newest.Modified) {
			newest = s
		}
	}
	if newest.Transcript == "" || now.Sub(newest.Modified) >= window {
		return Session{}, false, nil
	}
	return newest, true, nil
}

// sessionKeys are the project keys a session in wt can have. Claude Code keys
// on getcwd, the physical path, so the resolved form is the one that matters;
// the path as given is kept for a harness that records $PWD.
func sessionKeys(wt string) []string {
	keys := []string{memlink.ClaudeProjectKey(wt)}
	if r := resolvedPath(wt); r != wt {
		keys = append(keys, memlink.ClaudeProjectKey(r))
	}
	return keys
}

func matchesKey(name string, keys []string) bool {
	for _, k := range keys {
		if name == k || strings.HasPrefix(name, k+"-") {
			return true
		}
	}
	return false
}

// newestTranscript returns walk errors other than an entry vanishing mid-walk,
// which is a session ending, not a store that cannot be read.
func newestTranscript(dir string) (Session, error) {
	var newest Session
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.ModTime().After(newest.Modified) {
			newest = Session{Transcript: p, Modified: info.ModTime()}
		}
		return nil
	})
	return newest, err
}

// checkNoRecentSession is done's form of the check: a refusal naming the
// transcript and the override.
func checkNoRecentSession(projectsDir, absWT string, now time.Time) error {
	s, live, err := RecentSession(projectsDir, absWT, now, SessionWindow)
	if err != nil {
		return fmt.Errorf("cannot tell whether a Claude Code session is using %s: %w. Fix the read, or pass --ignore-recent-session", absWT, err)
	}
	if !live {
		return nil
	}
	return fmt.Errorf("a Claude Code session for %s wrote %s %s ago; removing the worktree would strand it. End or resume it elsewhere, or pass --ignore-recent-session",
		absWT, s.Transcript, now.Sub(s.Modified).Round(time.Minute))
}

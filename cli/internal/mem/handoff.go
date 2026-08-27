package mem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HandoffHeading is the section every session's handoff lives under.
const HandoffHeading = "## Session Handoff"

// ThreadPrefix opens one thread's sub-block.
const ThreadPrefix = "### "

// WriteThread replaces one thread's sub-block inside the handoff section,
// leaving every other thread byte-identical.
//
// WHY A COMMAND AND NOT A RULE. `harness/skills/handoff/SKILL.md` already says
// "if concurrent writes occurred, merge threads". It was violated twice in one
// evening — once by each of two sessions — and neither noticed, because the
// failure is invisible to the writer: last-writer-wins produces a well-formed
// file and a successful edit. Nothing looks wrong until a later session follows
// a pointer into a block that no longer exists.
//
// One shared mutable slot and N concurrent writers is a data-structure problem,
// not a discipline problem. This machine runs five worktrees. So the section
// becomes one sub-block per thread and a session rewrites ONLY its own; every
// other is foreign and untouchable — the same posture, and the same algorithm,
// as MergeHooks uses for a settings file a third party also writes.
//
// IT SHARES extractHandoffBlock's SECTION BOUNDARY rather than re-deriving it.
// That parser stops at the next "## ", so `###` sub-blocks fall inside it and
// archival keeps working unchanged. A second boundary rule here would be the
// fifth silently-divergent parser this repository has found in a week.
func WriteThread(content, threadKey, body string) (string, bool, error) {
	if strings.TrimSpace(threadKey) == "" {
		return "", false, fmt.Errorf("thread key is empty — a handoff with no thread is the shared slot this replaces")
	}
	if strings.Contains(threadKey, "\n") {
		return "", false, fmt.Errorf("thread key %q spans lines", threadKey)
	}

	lines := strings.Split(content, "\n")
	start, end := handoffSection(lines)
	if start < 0 {
		return "", false, fmt.Errorf("no %q section — refusing to invent one, because a handoff written where nothing reads it is worse than none", HandoffHeading)
	}

	want := renderThread(threadKey, body)
	tStart, tEnd := threadSpan(lines, start, end, threadKey)

	var out []string
	switch {
	case tStart >= 0:
		if equalBlocks(lines[tStart:tEnd], want) {
			return content, false, nil
		}
		out = append(out, lines[:tStart]...)
		out = append(out, want...)
		out = append(out, lines[tEnd:]...)
	default:
		// Appended at the END of the section, so existing threads keep their
		// positions. Reordering a foreign entry is a diff its author did not
		// make and would have to review.
		at := trimTrailingBlank(lines, start+1, end)
		out = append(out, lines[:at]...)
		if at > start+1 {
			out = append(out, "")
		}
		out = append(out, want...)
		out = append(out, lines[at:]...)
	}
	return strings.Join(out, "\n"), true, nil
}

// handoffSection returns the line range of the section body, exclusive of the
// heading, or (-1, -1). The end boundary is the next "## " heading or EOF —
// deliberately identical to extractHandoffBlock's.
func handoffSection(lines []string) (int, int) {
	start := -1
	for i, line := range lines {
		if isHandoffHeading(line) {
			start = i
			continue
		}
		if start >= 0 && strings.HasPrefix(line, "## ") {
			return start, i
		}
	}
	if start >= 0 {
		return start, len(lines)
	}
	return -1, -1
}

// threadSpan locates one thread's sub-block within the section.
func threadSpan(lines []string, start, end int, key string) (int, int) {
	want := ThreadPrefix + key
	for i := start + 1; i < end; i++ {
		if !isThreadHeading(lines[i], key) {
			continue
		}
		for j := i + 1; j < end; j++ {
			if strings.HasPrefix(lines[j], ThreadPrefix) {
				return i, j
			}
		}
		return i, end
	}
	_ = want
	return -1, -1
}

// isThreadHeading matches "### <key>" and "### <key> (anything)", so a thread can
// carry its branch in the heading without changing its identity — the branch
// moves, the worktree does not.
func isThreadHeading(line, key string) bool {
	rest, ok := strings.CutPrefix(strings.TrimRight(line, " \t\r"), ThreadPrefix)
	if !ok {
		return false
	}
	rest = strings.TrimSpace(rest)
	if rest == key {
		return true
	}
	head, _, found := strings.Cut(rest, " ")
	return found && head == key
}

func renderThread(key, body string) []string {
	out := []string{ThreadPrefix + key}
	body = strings.TrimRight(body, "\n")
	if body != "" {
		out = append(out, "")
		out = append(out, strings.Split(body, "\n")...)
	}
	out = append(out, "")
	return out
}

func equalBlocks(got, want []string) bool {
	return strings.Join(trimBlankEdges(got), "\n") == strings.Join(trimBlankEdges(want), "\n")
}

func trimBlankEdges(in []string) []string {
	s, e := 0, len(in)
	for s < e && strings.TrimSpace(in[s]) == "" {
		s++
	}
	for e > s && strings.TrimSpace(in[e-1]) == "" {
		e--
	}
	return in[s:e]
}

func trimTrailingBlank(lines []string, from, to int) int {
	for to > from && strings.TrimSpace(lines[to-1]) == "" {
		to--
	}
	return to
}

// ThreadKey derives a stable thread identity from a working directory.
//
// The worktree, because that is what distinguishes two concurrent sessions — not
// the agent and not the date. Journals were named `<date>-<project>-<agent>.md`
// and two WORKTREES on one day collided into `-2`/`-3` suffixes encoding nothing;
// six such files exist across two days, and no session could derive its own.
//
// IT ASKS GIT FIRST, and only then falls back to this repository's naming
// convention. That ordering is what makes the key agnostic across tools.
//
// Several tools create worktrees here and each names them differently: this
// repository writes `<repo>-wt-<slug>`, Claude Code's EnterWorktree writes
// `.claude/worktrees/<name>`, Orca writes its own. A key derived from one
// pattern resolves every other tool's worktree to "main" — so every session
// running under a different tool would share one thread and clobber the others,
// which is the bug this exists to fix, reintroduced for everyone not using our
// convention.
//
// git already knows, and states it on disk with no subprocess: a linked
// worktree's `.git` is a FILE reading `gitdir: …/.git/worktrees/<name>`, while a
// main checkout's `.git` is a directory.
//
// IT ALSO WALKS UP. The first version read only `filepath.Base(cwd)`, so a
// session in any subdirectory — `cli/`, where most work here happens — resolved
// to "main". The tests passed because every case they supplied was a worktree
// root; running `dotf mem thread` from `cli/` did not.
func ThreadKey(cwd string) string {
	dir := filepath.Clean(cwd)
	for {
		if name, ok := gitWorktreeName(dir); ok {
			return name
		}
		// Fallback: this repository's own convention, for a path that is not a
		// git worktree at all (a test fixture, or a tree checked out by copy).
		if _, slug, found := strings.Cut(filepath.Base(dir), "-wt-"); found && slug != "" {
			return "wt-" + slug
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "main"
		}
		dir = parent
	}
}

// gitWorktreeName reads `.git` at dir. It returns the linked worktree's name, or
// ("main", true) when `.git` is a directory — the main checkout. The bool
// distinguishes "this is a git root and here is the answer" from "keep walking".
func gitWorktreeName(dir string) (string, bool) {
	p := filepath.Join(dir, ".git")
	info, err := os.Lstat(p)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		return "main", true
	}
	raw, err := os.ReadFile(p) // #nosec G304 -- the .git pointer of the cwd being resolved
	if err != nil {
		return "", false
	}
	target, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "gitdir:")
	if !ok {
		return "", false
	}
	target = strings.TrimSpace(target)
	// `…/.git/worktrees/<name>`; the last element is git's own name for it.
	if parent := filepath.Base(filepath.Dir(filepath.Clean(target))); parent != "worktrees" {
		return "", false
	}
	name := filepath.Base(filepath.Clean(target))
	if name == "" || name == "." {
		return "", false
	}
	return name, true
}

// ThreadKeyForCwd is ThreadKey over the process's working directory.
func ThreadKeyForCwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "main"
	}
	return ThreadKey(wd)
}

// JournalName is the session record filename for a thread, so "my journal" is
// derivable from the working directory rather than remembered.
// The thread is appended ONLY when it disambiguates. A single-checkout session
// keeps the historical `<date>-<project>-<agent>.md`, because "main" carries no
// information and suffixing it would rename every archive on every
// single-session machine for nothing. Churn without a reader is not a migration,
// it is noise — and the collision being fixed only ever occurred between
// worktrees.
func JournalName(date, project, agent, thread string) string {
	if thread == "" || thread == "main" {
		return fmt.Sprintf("%s-%s-%s.md", date, project, agent)
	}
	return fmt.Sprintf("%s-%s-%s-%s.md", date, project, agent, thread)
}

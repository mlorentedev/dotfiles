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
// The worktree's basename, because that is what actually distinguishes two
// concurrent sessions — not the agent and not the date. Journal files were named
// `<date>-<project>-<agent>.md` and two WORKTREES on one day collided into `-2`
// and `-3` suffixes encoding nothing; six such files exist across two days. With
// the worktree in the key, a session can derive its own journal name from its cwd
// instead of guessing which suffix was its.
//
// A checkout that is not a worktree resolves to "main", so the ordinary
// single-session case reads naturally rather than carrying a hash.
// IT WALKS UP, and that is not defensive coding. The first version read only
// `filepath.Base(cwd)`, so a session working in any subdirectory — `cli/`, say,
// which is where most of this repository's work happens — resolved to "main".
// Every such session would have shared one thread key and clobbered each other
// exactly as before, while the tests passed because they all passed worktree
// roots. Found by running `dotf mem thread` from `cli/`.
func ThreadKey(cwd string) string {
	dir := filepath.Clean(cwd)
	for {
		base := filepath.Base(dir)
		// `<repo>-wt-<slug>` is this repository's worktree convention; keep the
		// slug, which is the part a human recognises.
		if _, slug, found := strings.Cut(base, "-wt-"); found && slug != "" {
			return "wt-" + slug
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "main"
		}
		dir = parent
	}
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
func JournalName(date, project, agent, thread string) string {
	if thread == "" {
		thread = "main"
	}
	return fmt.Sprintf("%s-%s-%s-%s.md", date, project, agent, thread)
}

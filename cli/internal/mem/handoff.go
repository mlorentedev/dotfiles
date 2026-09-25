package mem

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// HandoffHeading is the section every session's handoff lives under.
const HandoffHeading = "## Session Handoff"

// ThreadPrefix opens one thread's sub-block, and it is MARKED rather than merely
// shaped.
//
// A plain `### <key>` is indistinguishable from an ordinary subheading, and
// handoff bodies legitimately carry those — `### Next Actions`, `### Decisions`.
// The PR reviewer on #1279 found the consequence: such a heading truncated the
// thread's span, so a replacement rewrote only the part above it and stranded
// the rest under the previous thread. Data loss on ordinary content.
//
// Heuristics about which `###` is "really" a thread cannot fix that — the shapes
// are identical. An explicit marker can, and it is the same choice made for hook
// entries in #1272: identity is declared, never inferred.
const ThreadPrefix = "### thread: "

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
	return writeThread(content, threadKey, "", body)
}

// ThreadWrite is what WriteThreadAs did.
type ThreadWrite struct {
	Content string // the document after the write
	Key     string // the thread written: the key asked for, or its fork
	Kept    string // the agent whose block the key held, when the write forked
	Changed bool
}

// writerName is the shape of a writer, one word, because the stamp is read
// back out of the heading.
var writerName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// writerStamp is the stamp a thread heading carries after its key.
var writerStamp = regexp.MustCompile(`\(writer: ([^()\s]+)\)`)

// WriteThreadAs is WriteThread for a named writer, and it never replaces a block
// another agent wrote (MEMORY-009, #1690).
//
// On a default branch the key is `master@<host>`, which says nothing about WHO
// writes it, so two agents on one host shared one block and the later replaced
// the earlier. Four weeks of vault history hold three such losses, written 1.3,
// 22 and 71 hours after the block they erased: identity, not age, is what tells
// a session updating its own handoff from one erasing somebody else's.
//
// So the writer is stamped into the heading, and a block another agent wrote is
// left where it is: the write goes to `<key>+<agent>` and the caller reports the
// fork. A block written before stamps existed is attributed by the journal it
// points at. A block nothing attributes is replaced as it always was, the one
// case this cannot protect, and the stamp closes it for every block written from
// now on. With no agent this is WriteThread, byte for byte.
func WriteThreadAs(content, threadKey, agent, body string) (ThreadWrite, error) {
	if agent == "" {
		out, changed, err := WriteThread(content, threadKey, body)
		return ThreadWrite{Content: out, Key: threadKey, Changed: changed}, err
	}
	if !writerName.MatchString(agent) {
		return ThreadWrite{}, fmt.Errorf("agent %q: a writer is one lower-case word, such as claude or pi, because the heading's stamp is read back", agent)
	}
	key, kept := threadKey, ""
	if w := threadWriter(content, threadKey, agent); w != "" && w != agent {
		key, kept = threadKey+"+"+agent, w
	}
	out, changed, err := writeThread(content, key, " (writer: "+agent+")", body)
	return ThreadWrite{Content: out, Key: key, Kept: kept, Changed: changed}, err
}

// threadWriter names the agent that wrote key's block: its heading's stamp, or
// else the agent named by the journal the block points at. Empty when there is
// no such block or nothing names its writer.
func threadWriter(content, key, self string) string {
	lines := strings.Split(content, "\n")
	start, end := handoffSection(lines)
	if start < 0 {
		return ""
	}
	s, e := threadSpan(lines, start, end, key)
	if s < 0 {
		return ""
	}
	if m := writerStamp.FindStringSubmatch(lines[s]); m != nil {
		return m[1]
	}
	for _, l := range lines[s+1 : e] {
		if w := journalWriter(l, self); w != "" {
			return w
		}
	}
	return ""
}

// journalFile captures a Journal line's file name after its date, which
// JournalName builds as <project>-<agent>[-<thread>].
var journalFile = regexp.MustCompile(`^\s*(?:>\s*)?Journal:.*?\d{4}-\d{2}-\d{2}-([^\s/()\[\]]+?)\.md`)

// journalAgents are the agents that write journals: every one the vault's
// journal names held on 2026-09-25 (claude, pi, antigravity, agy, copilot) and
// the harness's other targets. No project name contains one, so the first
// match after the date is the writer, ahead of the thread's words. An agent
// missing here reads as unknown, and an unknown writer's block is replaced as it
// always was.
var journalAgents = map[string]bool{
	"agy": true, "antigravity": true, "claude": true, "codex": true,
	"copilot": true, "gemini": true, "opencode": true, "pi": true,
}

func journalWriter(line, self string) string {
	m := journalFile.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	for _, word := range strings.Split(m[1], "-") {
		if word == self || journalAgents[word] {
			return word
		}
	}
	return ""
}

// writeThread is WriteThread with a stamp after the key in the heading it writes.
func writeThread(content, threadKey, stamp, body string) (string, bool, error) {
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

	// Text written before threads existed has no owner, so no write would ever
	// replace it, and it sits first under the heading, where the next session
	// reads it as the handoff (#1651). It is lifted out here and re-appended as
	// a thread of its own once this write is done.
	var legacy []string
	legacyKey, migrating := "", false
	if ls, le, ok := legacyBlock(lines, start, end); ok {
		legacy, legacyKey, migrating = trimBlankEdges(lines[ls:le]), legacyThreadKey(lines, start, end, ls, le), true
		lines = append(append(append([]string{}, lines[:start+1]...), ""), lines[le:]...)
		start, end = handoffSection(lines)
	}

	want := renderThread(threadKey+stamp, body)
	tStart, tEnd := threadSpan(lines, start, end, threadKey)

	var out []string
	switch {
	case tStart >= 0:
		if equalBlocks(lines[tStart:tEnd], want) {
			if !migrating {
				return content, false, nil
			}
			return appendThread(lines, legacyKey, strings.Join(legacy, "\n")), true, nil
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
	if migrating {
		return appendThread(out, legacyKey, strings.Join(legacy, "\n")), true, nil
	}
	return strings.Join(out, "\n"), true, nil
}

// appendThread adds one thread at the end of the handoff section, the same way
// a new thread is placed, and returns the joined document.
func appendThread(lines []string, key, body string) string {
	start, end := handoffSection(lines)
	at := trimTrailingBlank(lines, start+1, end)
	out := append([]string{}, lines[:at]...)
	if at > start+1 {
		out = append(out, "")
	}
	out = append(out, renderThread(key, body)...)
	out = append(out, lines[at:]...)
	return strings.Join(out, "\n")
}

// updatedDate reads the `> Updated: YYYY-MM-DD` line a handoff block opens with.
var updatedDate = regexp.MustCompile(`^>\s*Updated:\s*(\d{4}-\d{2}-\d{2})`)

// LegacyThreadKey reports the thread an un-threaded handoff block would be moved
// into by the next write, or false when the section has none (#1651).
func LegacyThreadKey(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	start, end := handoffSection(lines)
	if start < 0 {
		return "", false
	}
	ls, le, ok := legacyBlock(lines, start, end)
	if !ok {
		return "", false
	}
	return legacyThreadKey(lines, start, end, ls, le), true
}

// legacyBlock returns the span between the section heading and its first marked
// thread when that span holds any non-blank text.
func legacyBlock(lines []string, start, end int) (int, int, bool) {
	first := end
	for i := start + 1; i < end; i++ {
		if _, ok := threadHeadingKey(lines[i]); ok {
			first = i
			break
		}
	}
	for i := start + 1; i < first; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return start + 1, first, true
		}
	}
	return 0, 0, false
}

// legacyThreadKey names the moved block by its own Updated date, so the thread
// says when it was written, and never reuses the key of a thread that exists.
func legacyThreadKey(lines []string, start, end, ls, le int) string {
	date := "undated"
	for _, l := range lines[ls:le] {
		if m := updatedDate.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			date = m[1]
			break
		}
	}
	key := "legacy-" + date
	for n := 2; ; n++ {
		if s, _ := threadSpan(lines, start, end, key); s < 0 {
			return key
		}
		key = fmt.Sprintf("legacy-%s-%d", date, n)
	}
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
//
// A thread ends at the next MARKED thread heading, so an ordinary `###`
// subheading inside a body is just content. See ThreadPrefix for why the marker
// exists rather than a heuristic.
func threadSpan(lines []string, start, end int, key string) (int, int) {
	for i := start + 1; i < end; i++ {
		if !isThreadHeading(lines[i], key) {
			continue
		}
		for j := i + 1; j < end; j++ {
			if _, ok := threadHeadingKey(lines[j]); ok {
				return i, j
			}
		}
		return i, end
	}
	return -1, -1
}

// threadHeadingKey returns the key a `### ` line declares, if any.
func threadHeadingKey(line string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimRight(line, " \t\r"), ThreadPrefix)
	if !ok {
		return "", false
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	head, _, found := strings.Cut(rest, " ")
	if found {
		return head, true
	}
	return rest, true
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
// A THREAD IS A LINE OF WORK, AND GIT ALREADY NAMES IT: THE BRANCH.
//
// The first version keyed on the worktree. That correlates on one machine and
// decorrelates the moment the work moves to another — and the vault is the SSOT
// across machines, so the same branch continued on a second box must resolve to
// the SAME thread rather than forking one. Every worktree in this repository is
// born on its own branch (measured: three live worktrees, three distinct
// branches, no duplicates), so the branch discriminates at least as well and it
// travels.
//
// THE HOSTNAME ENTERS ONLY WHERE IT DISAMBIGUATES. `main` on two machines is two
// pieces of ambient work, so the default branch is keyed `main@<host>` while
// every feature branch stays clean — the same only-when-it-disambiguates rule
// JournalName applies to its `-main` suffix.
//
// A DETACHED HEAD is named `<worktree>@<host>` and says so, rather than
// collapsing into a plausible "main" that would silently share a thread with
// somebody else's work.
//
// It reads git's own on-disk state through RepoIdentity — no subprocess, no
// naming convention — so a worktree created by Claude Code, Orca, opencode, pi,
// agy, copilot or a bare `git worktree add` all behave identically. A key
// derived from one tool's path pattern resolved every other tool's worktree to
// "main", which reintroduced this very bug for everyone not using our
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
// GIT IS CONSULTED ACROSS THE WHOLE WALK-UP BEFORE THE CONVENTION IS CONSULTED
// AT ALL, and the ordering is the fix for a second reviewer finding on #1279:
// interleaving them let an ordinary directory that merely CONTAINS `-wt-`
// (`vendor-wt-cache`, say) claim its own thread while sitting inside a plain
// checkout whose `.git` one level up said "main". Authoritative answers come
// first; a name-shaped guess is only for a tree git does not know at all.
func ThreadKey(cwd string) string {
	if id, ok := RepoIdentity(cwd); ok {
		switch {
		case id.Branch == "":
			// Detached HEAD: no line of work to name. Say so rather than
			// collapsing into a plausible "main" that would silently share a
			// thread with somebody else's work.
			name := id.Worktree
			if name == "" {
				name = "detached"
			}
			return sanitizeThread(name) + "@" + shortHost()
		case isDefaultBranch(id.Branch):
			return id.Branch + "@" + shortHost()
		default:
			return sanitizeThread(id.Branch)
		}
	}
	// git knows nothing about this path — a test fixture, or a tree copied
	// rather than cloned. Fall back to this repository's own convention.
	for dir := filepath.Clean(cwd); ; {
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

// ThreadKeyForCwd is ThreadKey over the process's working directory.
func ThreadKeyForCwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "main"
	}
	return ThreadKey(wd)
}

// HandoffThread is the thread key handoff-write uses for memoryPath (#1606).
//
// An explicit key is taken as given. Otherwise the key is the cwd's, but only
// when the cwd's repository is the project memoryPath belongs to. A branch names
// a line of work in ITS repository: the vault's `master` says nothing about
// dotfiles, and keying dotfiles' MEMORY.md with it replaced another session's
// `master@<host>` block while reporting success. A MEMORY.md outside the vault's
// `10_projects/<project>/memory/` layout names no project to compare against, so
// there the cwd key stands, as it always did.
func HandoffThread(explicit, memoryPath, cwd string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	key := ThreadKey(cwd)
	project, ok := memoryProject(memoryPath)
	if !ok {
		return key, nil
	}
	id, inRepo := RepoIdentity(cwd)
	if inRepo && id.Project == project {
		return key, nil
	}
	here := "no repository"
	if inRepo {
		here = fmt.Sprintf("repository %q", id.Project)
	}
	return "", fmt.Errorf("%s belongs to project %q, but the current directory is in %s, "+
		"whose branch names no line of work there (it would have keyed %q); "+
		"pass --thread with the branch the work is on", memoryPath, project, here, key)
}

// memoryProject returns <project> for a path ending in
// 10_projects/<project>/memory/MEMORY.md, the vault's project layout.
func memoryProject(memoryPath string) (string, bool) {
	p := filepath.Clean(memoryPath)
	memDir := filepath.Dir(p)
	projDir := filepath.Dir(memDir)
	if filepath.Base(p) != "MEMORY.md" || filepath.Base(memDir) != "memory" ||
		filepath.Base(filepath.Dir(projDir)) != "10_projects" {
		return "", false
	}
	return filepath.Base(projDir), true
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

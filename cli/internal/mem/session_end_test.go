package mem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

// writeMemory drops a MEMORY.md for project "proj" under vault root and returns
// the project cwd the hook payload would carry.
func writeMemory(t *testing.T, vault, content string) {
	t.Helper()
	dir := filepath.Join(vault, "10_projects", "proj", "memory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSessionEnd_NoOps covers the resilience contract: every trivial / missing /
// malformed input is a clean no-op (no file written, no error surfaced).
func TestSessionEnd_NoOps(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		memory  string // MEMORY.md content; "" means do not create the file
	}{
		{"empty stdin", "", ""},
		{"malformed json", "{not json", ""},
		{"missing cwd", `{"session_id":"abc"}`, ""},
		{"absent memory file", `{"cwd":"/x/proj","session_id":"abc"}`, ""},
		{"no handoff heading", `{"cwd":"/x/proj","session_id":"abc"}`, "# Project\n\nsome notes\n"},
		{"whitespace-only block", `{"cwd":"/x/proj","session_id":"abc"}`, "# P\n\n## Session Handoff\n\n   \n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vault := t.TempDir()
			if tc.memory != "" {
				writeMemory(t, vault, tc.memory)
			}
			written, err := SessionEnd([]byte(tc.payload), vault, fixedNow)
			if err != nil {
				t.Fatalf("expected no error on no-op, got %v", err)
			}
			if written != "" {
				t.Fatalf("expected no file written, got %q", written)
			}
		})
	}
}

// TestSessionEnd_HappyPath archives the handoff block into a durable record.
func TestSessionEnd_HappyPath(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "# Project\n\n## Index\n\n- foo\n\n## Session Handoff\n\n**Last task:** shipped PR1.\n**Next:** PR2.\n")

	written, err := SessionEnd([]byte(`{"cwd":"/home/me/proj","session_id":"sid-123"}`), vault, fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(vault, "10_projects", "proj", "sessions", "2026-06-23-proj-claude.md")
	if written != want {
		t.Fatalf("written path = %q, want %q", written, want)
	}
	b, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("reading record: %v", err)
	}
	got := string(b)
	for _, frag := range []string{
		`id: "session-2026-06-23-proj-claude"`,
		"type: session",
		"status: active",
		`created: "2026-06-23"`,
		"owner: manu",
		"session_id: sid-123",
		"agent: claude",
		"project: proj",
		`date: "2026-06-23"`,
		"tags: [session, handoff, proj]",
		"# Session 2026-06-23 — proj (claude)",
		"**Last task:** shipped PR1.",
		"**Next:** PR2.",
	} {
		if !strings.Contains(got, frag) {
			t.Errorf("record missing %q\n--- got ---\n%s", frag, got)
		}
	}
	// The block stops at the next "## " heading — the index must not leak in.
	if strings.Contains(got, "## Index") || strings.Contains(got, "- foo") {
		t.Errorf("record leaked content past the handoff block:\n%s", got)
	}
}

func TestSessionEndWaitsForConcurrentHandoffWrite(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	vault := t.TempDir()
	writeMemory(t, vault, "## Session Handoff\n\n**Next action:** preserve this handoff.\n")
	memory := filepath.Join(vault, "10_projects", "proj", "memory", "MEMORY.md")
	unlock, err := LockHandoffMemory(memory)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		path string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		path, err := SessionEnd([]byte(`{"cwd":"/x/proj","session_id":"sid"}`), vault, fixedNow)
		done <- result{path: path, err: err}
	}()

	select {
	case got := <-done:
		unlock()
		t.Fatalf("SessionEnd read MEMORY.md while a handoff writer held its lock: %+v", got)
	case <-time.After(250 * time.Millisecond):
	}
	unlock()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.path == "" {
			t.Fatal("SessionEnd did not archive after the handoff lock was released")
		}
	case <-time.After(time.Second):
		t.Fatal("SessionEnd did not resume after the handoff lock was released")
	}
}

// TestSessionEnd_UsesLocalCalendarDate pins the CLI-043 contract: the record's
// date is the calendar date of the `now` it is handed, in that value's own
// location — never normalised to UTC. An 18:30 session in a -0600 zone is
// 00:30 the next day in UTC; filing it under tomorrow both misdates the record
// and collides with the following morning's session on one filename.
func TestSessionEnd_UsesLocalCalendarDate(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "## Session Handoff\n\nevening work\n")

	denver := time.FixedZone("MDT", -6*60*60)
	evening := time.Date(2026, 8, 23, 18, 30, 0, 0, denver)
	if evening.UTC().Format("2006-01-02") == evening.Format("2006-01-02") {
		t.Fatal("fixture is not timezone-sensitive; it cannot catch the regression")
	}

	written, err := SessionEnd([]byte(`{"cwd":"/x/proj","session_id":"sid"}`), vault, evening)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(vault, "10_projects", "proj", "sessions", "2026-08-23-proj-claude.md")
	if written != want {
		t.Fatalf("written path = %q, want %q", written, want)
	}
	b, _ := os.ReadFile(written)
	for _, frag := range []string{`date: "2026-08-23"`, `created: "2026-08-23"`, "# Session 2026-08-23 —"} {
		if !strings.Contains(string(b), frag) {
			t.Errorf("record missing %q\n--- got ---\n%s", frag, b)
		}
	}
}

// TestSessionEnd_FrontmatterKeyOrder pins handoff/SKILL.md §1b: id, type and
// status are the first three keys, which is what vault_health validates against.
func TestSessionEnd_FrontmatterKeyOrder(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "## Session Handoff\n\nwork\n")
	written, err := SessionEnd([]byte(`{"cwd":"/x/proj","session_id":"sid"}`), vault, fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, _ := os.ReadFile(written)
	lines := strings.Split(string(b), "\n")
	if len(lines) < 4 {
		t.Fatalf("record too short:\n%s", b)
	}
	for i, prefix := range []string{"---", "id: ", "type: ", "status: "} {
		if !strings.HasPrefix(lines[i], prefix) {
			t.Errorf("line %d = %q, want prefix %q\n--- got ---\n%s", i, lines[i], prefix, b)
		}
	}
	// The Frontmatter Law requires these regardless of position.
	for _, frag := range []string{"created: ", "owner: "} {
		if !strings.Contains(string(b), frag) {
			t.Errorf("record missing %q\n--- got ---\n%s", frag, b)
		}
	}
}

// TestSessionEnd_DefaultsSessionID falls back to "unknown" when the payload omits it.
func TestSessionEnd_DefaultsSessionID(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "## Session Handoff\n\nwork happened\n")
	written, err := SessionEnd([]byte(`{"cwd":"/x/proj"}`), vault, fixedNow)
	if err != nil || written == "" {
		t.Fatalf("expected a record, got written=%q err=%v", written, err)
	}
	b, _ := os.ReadFile(written)
	if !strings.Contains(string(b), "session_id: unknown") {
		t.Errorf("expected session_id: unknown, got:\n%s", b)
	}
}

// TestSessionEnd_EmptyVaultIsNoOp: an unresolved vault ("") never writes.
func TestSessionEnd_EmptyVaultIsNoOp(t *testing.T) {
	written, err := SessionEnd([]byte(`{"cwd":"/x/proj","session_id":"abc"}`), "", fixedNow)
	if err != nil || written != "" {
		t.Fatalf("empty vault must be a no-op, got written=%q err=%v", written, err)
	}
}

// TWO CONCURRENT WORKTREES MUST NOT ARCHIVE OVER EACH OTHER.
//
// The archive filename was built here as `<date>-<project>-claude.md`, hardcoded
// and with no thread, and written with os.WriteFile — a TRUNCATING write to a
// shared name. With several worktrees running, the last SessionEnd destroyed
// every other session's durable record, which is the same defect HARNESS-088
// fixes in the handoff block, in the path that was supposed to be the safe copy.
//
// It also divergedfrom the skill: `dotf mem thread` names the journal with the
// thread while this assembled its own — two files per session, drifting apart.
// Both now come from JournalName.
func TestSessionEndArchivesPerWorktreeRatherThanOverEachOther(t *testing.T) {
	vault := t.TempDir()
	memDir := filepath.Join(vault, "10_projects", "proj", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"),
		[]byte("# M\n\n## Session Handoff\n\n### wt-a\n\nfrom a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Two worktrees, written the way git writes a linked worktree.
	// Two worktrees of the SAME repository on DIFFERENT branches — the real
	// concurrent case. The project comes from the repo, the thread from the
	// branch, and both from the one RepoIdentity.
	a := gitFixture(t, "proj", "wt-a", "feat/a")
	b := gitFixture(t, "proj", "wt-b", "feat/b")

	// Marshalled, never concatenated: a Windows cwd is `C:\Users\...` and every
	// backslash is a JSON escape, so a hand-built payload fails to parse and
	// SessionEnd no-ops SILENTLY. Caught by CI on windows-latest; GOOS=windows
	// go vet cannot see it, because vet is not a test run.
	payload := func(cwd, sid string) []byte {
		b, err := json.Marshal(map[string]string{"cwd": cwd, "session_id": sid})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	wroteA, err := SessionEnd(payload(a, "sa"), vault, fixedNow)
	if err != nil || wroteA == "" {
		t.Fatalf("first archive failed: %v", err)
	}
	wroteB, err := SessionEnd(payload(b, "sb"), vault, fixedNow)
	if err != nil || wroteB == "" {
		t.Fatalf("second archive failed: %v", err)
	}

	if wroteA == wroteB {
		t.Fatalf("both worktrees archived to the same file %q — the second destroyed the first", wroteA)
	}
	for _, p := range []string{wroteA, wroteB} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("archive %s does not exist: %v", p, err)
		}
	}
	// And the name is the one the skill would derive, not a second convention.
	if want := JournalName(fixedNow.Format("2006-01-02"), "proj", "claude", "feat-a"); filepath.Base(wroteA) != want {
		t.Errorf("archive name %q diverges from JournalName %q", filepath.Base(wroteA), want)
	}
}

// TestSessionEndLeavesAnAuthoredJournalByteIdentical is #1620. The hook and the
// /handoff skill name the same file (JournalName, on purpose: one record per
// session), and the hook wrote it with a truncating os.WriteFile. So every
// session that ran /handoff lost its journal to a copy of the MEMORY block when
// it ended, and the result looked like success. The hook is the fallback for a
// session that wrote no journal; it never replaces one. The other half of the
// contract, a session with no journal still getting its record, is
// TestSessionEnd_HappyPath.
func TestSessionEndLeavesAnAuthoredJournalByteIdentical(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "# P\n\n## Session Handoff\n\n**Last task:** shipped.\n")
	sessions := filepath.Join(vault, "10_projects", "proj", "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	cwd := "/home/me/proj"
	journal := filepath.Join(sessions, JournalName(fixedNow.Format("2006-01-02"), "proj", "claude", ThreadKey(cwd)))
	authored := "---\nid: session-2026-06-23-proj-claude\ntype: session\nstatus: active\n---\n\n## Context & Objectives\n\nWritten by /handoff.\n"
	if err := os.WriteFile(journal, []byte(authored), 0o644); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(map[string]string{"cwd": cwd, "session_id": "sid-9"})
	if err != nil {
		t.Fatal(err)
	}
	written, err := SessionEnd(payload, vault, fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != "" {
		t.Errorf("SessionEnd reported writing %q over a journal that already existed", written)
	}
	got, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != authored {
		t.Errorf("the authored journal was changed:\n--- got ---\n%s\n--- want ---\n%s", got, authored)
	}
}

// MEMORY-012 (#1920): the fallback journal is named for one thread, so it holds
// that thread only. It used to copy the whole section, every peer's handoff
// included, into a record named for one of them.
func TestSessionEndArchivesOnlyItsOwnThread(t *testing.T) {
	const memory = "# M\n\n## Session Handoff\n\n" +
		"### thread: feat-a (writer: claude)\n\nfrom a\n\n" +
		"### thread: feat-b (writer: claude)\n\nfrom b\n\n" +
		"### thread: feat-c (writer: pi)\n\nfrom pi on c\n\n" +
		"### thread: feat-c+claude (writer: claude)\n\nfrom claude on c\n\n" +
		"### thread: feat-d (writer: pi)\n\nfrom pi on d\n\n" +
		"## Index\n\n- tail\n"
	for _, tc := range []struct {
		branch, want string
		others       []string
	}{
		{"feat/a", "from a", []string{"from b", "from pi on c", "from claude on c", "- tail"}},
		{"feat/b", "from b", []string{"from a", "from pi on c", "from claude on c"}},
		// Claude's fork of a key pi holds is Claude's thread.
		{"feat/c", "from claude on c", []string{"from pi on c", "from a", "from b"}},
		// A key only another agent wrote is not this session's to archive.
		{"feat/d", "", nil},
		// A thread this session never wrote archives nothing.
		{"feat/none", "", nil},
	} {
		t.Run(tc.branch, func(t *testing.T) {
			vault := t.TempDir()
			writeMemory(t, vault, memory)
			wt := gitFixture(t, "proj", "wt", tc.branch)
			payload, err := json.Marshal(map[string]string{"cwd": wt, "session_id": "s"})
			if err != nil {
				t.Fatal(err)
			}
			written, err := SessionEnd(payload, vault, fixedNow)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if written != "" {
					t.Fatalf("archived %s for a thread this session does not own", written)
				}
				return
			}
			got := readFileT(t, written)
			if !strings.Contains(got, tc.want) {
				t.Errorf("own thread missing (%q):\n%s", tc.want, got)
			}
			for _, o := range tc.others {
				if strings.Contains(got, o) {
					t.Errorf("record carries %q, which is not this thread's:\n%s", o, got)
				}
			}
		})
	}
}

// A second session ending on the same thread and day keeps the first record:
// scoping the block did not loosen O_EXCL's refusal to replace one (#1620).
func TestSessionEndThreadScopedArchiveKeepsNoOverwrite(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "# M\n\n## Session Handoff\n\n### thread: feat-a (writer: claude)\n\nfirst\n")
	wt := gitFixture(t, "proj", "wt", "feat/a")
	payload, _ := json.Marshal(map[string]string{"cwd": wt, "session_id": "s1"})
	first, err := SessionEnd(payload, vault, fixedNow)
	if err != nil || first == "" {
		t.Fatalf("first archive: %q, %v", first, err)
	}
	writeMemory(t, vault, "# M\n\n## Session Handoff\n\n### thread: feat-a (writer: claude)\n\nsecond\n")
	again, err := SessionEnd(payload, vault, fixedNow)
	if err != nil || again != "" {
		t.Fatalf("second archive replaced the first: %q, %v", again, err)
	}
	if got := readFileT(t, first); !strings.Contains(got, "first") || strings.Contains(got, "second") {
		t.Errorf("the first record changed:\n%s", got)
	}
}

func readFileT(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// MEMORY-015 (#1929): the agent names the record, from the caller and never
// from the handoff text. Two agents ending sessions on one thread and day each
// get their own journal, each archiving its own block; with the name hardcoded
// the second was a silent no-op and the first was misattributed to claude.
func TestSessionEndTwoAgentsOnOneThreadAndDayKeepTwoRecords(t *testing.T) {
	const memory = "# M\n\n## Session Handoff\n\n" +
		"### thread: feat-a (writer: claude)\n\nclaude's handoff\n\n" +
		"### thread: feat-a+pi (writer: pi)\n\npi's handoff\n\n" +
		"## Index\n"
	vault := t.TempDir()
	writeMemory(t, vault, memory)
	wt := gitFixture(t, "proj", "wt", "feat/a")
	payload, err := json.Marshal(map[string]string{"cwd": wt, "session_id": "s"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ agent, own, other string }{
		{"claude", "claude's handoff", "pi's handoff"},
		{"pi", "pi's handoff", "claude's handoff"},
	} {
		written, err := SessionEndAs(payload, vault, tc.agent, fixedNow)
		if err != nil {
			t.Fatalf("%s: %v", tc.agent, err)
		}
		if written == "" {
			t.Fatalf("%s: no record; the other agent's journal took its place", tc.agent)
		}
		if want := JournalName(fixedNow.Format("2006-01-02"), "proj", tc.agent, "feat-a"); filepath.Base(written) != want {
			t.Errorf("%s: record %s, want %s", tc.agent, filepath.Base(written), want)
		}
		got := readFileT(t, written)
		for _, want := range []string{tc.own, "agent: " + tc.agent + "\n", "-" + tc.agent + "\"\n", "(" + tc.agent + ")"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: record lacks %q:\n%s", tc.agent, want, got)
			}
		}
		if strings.Contains(got, tc.other) {
			t.Errorf("%s: record carries the other agent's block:\n%s", tc.agent, got)
		}
	}
	entries, err := os.ReadDir(filepath.Join(vault, "10_projects", "proj", "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("want two journals, got %d", len(entries))
	}
}

// The default keeps the hook that predates --agent working as it did.
func TestSessionEndWithoutAnAgentIsClaude(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "# M\n\n## Session Handoff\n\n### thread: feat-a (writer: claude)\n\nx\n")
	wt := gitFixture(t, "proj", "wt", "feat/a")
	payload, _ := json.Marshal(map[string]string{"cwd": wt, "session_id": "s"})
	written, err := SessionEnd(payload, vault, fixedNow)
	if err != nil || written == "" {
		t.Fatalf("SessionEnd = %q, %v", written, err)
	}
	if !strings.Contains(filepath.Base(written), "-claude-") {
		t.Errorf("record %s is not named for claude", filepath.Base(written))
	}
}

// The agent becomes a path component, so it is held to the writer rule.
func TestSessionEndAsRefusesAnAgentThatIsNotOneWord(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "# M\n\n## Session Handoff\n\nx\n")
	payload, _ := json.Marshal(map[string]string{"cwd": t.TempDir(), "session_id": "s"})
	for _, agent := range []string{"", "../x", "Claude", "pi agent"} {
		if _, err := SessionEndAs(payload, vault, agent, fixedNow); err == nil {
			t.Errorf("agent %q accepted", agent)
		}
	}
}

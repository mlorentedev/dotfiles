package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The golden corpus (tests/golden/vault-health/) proves end-to-end byte-parity
// with the shell across all 16 characterized cases. These unit tests cover a
// seam a golden cannot reach: the "scripts dir unresolved" fail-loud path
// (spec-required, but out of scope for the shell to have ever exercised — a
// bash script always has its own $SCRIPT_DIR) and the pure helper functions.

func TestPct(t *testing.T) {
	tests := []struct {
		count, total, want int
	}{
		{0, 0, 0},
		{5, 0, 0},
		{1, 5, 20},
		{30, 100, 30},
		{1, 3, 33}, // integer floor, not rounding
	}
	for _, tt := range tests {
		if got := pct(tt.count, tt.total); got != tt.want {
			t.Errorf("pct(%d, %d) = %d, want %d", tt.count, tt.total, got, tt.want)
		}
	}
}

func TestHasAnyChar(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"one word", "knowledge", true},
		{"whitespace-only line still matches, like grep -q '.'", " ", true},
		{"blank plus a real line", "\nsomething", true},
	}
	for _, tt := range tests {
		if got := hasAnyChar(tt.in); got != tt.want {
			t.Errorf("%s: hasAnyChar(%q) = %v, want %v", tt.name, tt.in, got, tt.want)
		}
	}
}

// The obsidian CLI exits 0 and prints its own errors on stdout, so these are
// the answers a connectivity check that only looks for output would pass.
func TestNamesVault(t *testing.T) {
	tests := []struct {
		name, out string
		want      bool
	}{
		{"real CLI answer (1.14, macOS)", "name\tknowledge\npath\t/Users/x/Projects/knowledge\nfiles\t2631", true},
		{"bare name is not the record", "knowledge", false},
		{"an error that mentions the vault", "Error: Vault knowledge not found.", false},
		{"vault not found", "Vault not found.", false},
		{"flag read as a command", `Error: Command "--no-sandbox" not found. It may require a plugin to be enabled.`, false},
		{"another vault", "name\tknowledge-old", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		if got := namesVault(tt.out, "knowledge"); got != tt.want {
			t.Errorf("%s: namesVault(%q) = %v, want %v", tt.name, tt.out, got, tt.want)
		}
	}
}

// Vault first on every OS; --no-sandbox only off darwin, where the separate
// CLI reads it as a command (measured on obsidian 1.14.4).
func TestObsidianArgsPerOS(t *testing.T) {
	for goos, want := range map[string]string{
		"linux":   "--no-sandbox vault=knowledge unresolved format=json",
		"windows": "--no-sandbox vault=knowledge unresolved format=json",
		"darwin":  "vault=knowledge unresolved format=json",
	} {
		h := &healthRun{opts: HealthOptions{VaultName: "knowledge", GOOS: goos}}
		if got := strings.Join(h.obsidianArgs("unresolved", "format=json"), " "); got != want {
			t.Errorf("%s: argv %q, want %q", goos, got, want)
		}
	}
}

// The golden corpus cannot carry a memory/ directory: GUARD-001 refuses an
// agent-memory path outside the vault. So the directory rule is pinned here.
func TestOrphanExempt(t *testing.T) {
	tests := []struct {
		rel  string
		want bool
	}{
		{"90_archive/old.md", true},
		{"10_projects/p/sessions/2026-10-07.md", true},
		{"10_projects/p/memory/feedback.md", true},
		{"memory/top.md", true},
		{"00_meta/memory.md", false},   // a file named like the directory
		{"00_meta/sessions.md", false}, // likewise
		{"10_projects/memoryless/a.md", false},
		{"00_meta/note.md", false},
	}
	for _, tt := range tests {
		if got := orphanExempt(tt.rel); got != tt.want {
			t.Errorf("orphanExempt(%q) = %v, want %v", tt.rel, got, tt.want)
		}
	}
}

// Each zone of linkExemptions applies to the counts it names and no other;
// memory/ is pinned here for the same GUARD-001 reason as above.
func TestLinkExemptPerCheck(t *testing.T) {
	tests := []struct {
		rel                         string
		orphan, deadEnd, unresolved bool
	}{
		{"90_archive/old.md", true, true, true},
		{"00_meta/templates/t.md", false, true, true},
		{"20_certifications/lfs/n.md", false, true, false},
		{"10_projects/p/sessions/2026-10-09.md", true, true, false},
		{"10_projects/p/memory/feedback.md", true, true, false},
		{"10_projects/p/memory/threads/feat-x.md", true, true, false},
		{"00_meta/templates.md", false, false, false},
		{"50_work/20-products/x.md", false, true, false},
		{"50_work/30-clients/acme/x.md", false, true, false},
		{"50_work/25-applications/x.md", false, false, false},
		{"50_work/20-products.md", false, false, false},
	}
	for _, tt := range tests {
		if got := linkExempt(checkOrphans, tt.rel); got != tt.orphan {
			t.Errorf("orphans: linkExempt(%q) = %v, want %v", tt.rel, got, tt.orphan)
		}
		if got := linkExempt(checkDeadEnds, tt.rel); got != tt.deadEnd {
			t.Errorf("dead-ends: linkExempt(%q) = %v, want %v", tt.rel, got, tt.deadEnd)
		}
		if got := linkExempt(checkUnresolved, tt.rel); got != tt.unresolved {
			t.Errorf("unresolved: linkExempt(%q) = %v, want %v", tt.rel, got, tt.unresolved)
		}
	}
}

func TestExemptZones(t *testing.T) {
	if got, want := exemptZones(checkDeadEnds, "and"), "90_archive/, 00_meta/templates/, 20_certifications/, 50_work/20-products/, 50_work/30-clients/, sessions/ and memory/"; got != want {
		t.Errorf("exemptZones(dead-ends) = %q, want %q", got, want)
	}
	if got, want := exemptZones(checkUnresolved, "or"), "90_archive/ or 00_meta/templates/"; got != want {
		t.Errorf("exemptZones(unresolved) = %q, want %q", got, want)
	}
}

func TestCountNonBlank(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"whitespace-only line does NOT count, unlike hasAnyChar", " ", 0},
		{"three real lines", "a\nb\nc", 3},
		{"blank line among real ones is skipped", "a\n\nb", 2},
	}
	for _, tt := range tests {
		if got := countNonBlank(tt.in); got != tt.want {
			t.Errorf("%s: countNonBlank(%q) = %d, want %d", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestCollectMarkdownFilesExcludesObsidianSubtree(t *testing.T) {
	dir := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	write("a.md")
	write("sub/b.md")
	write(".obsidian/workspace.md")
	write("sub/.obsidian/nested.md")
	write("not-markdown.txt")

	got := collectMarkdownFiles(dir)
	if len(got) != 2 {
		t.Fatalf("collectMarkdownFiles() = %v, want exactly a.md and sub/b.md", got)
	}
	for _, f := range got {
		if strings.Contains(f, ".obsidian") {
			t.Errorf("collectMarkdownFiles() leaked a .obsidian path: %s", f)
		}
	}
}

func TestFrontmatterCountsOneMatchPerFileNotPerLine(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return p
	}
	f1 := write("f1.md", "id: one\nid: duplicate-should-not-double-count\n")
	f2 := write("f2.md", "type: note\n")

	h := &healthRun{mdFiles: []string{f1, f2}}
	counts := h.frontmatterCounts([]string{"id", "type", "status"})

	if counts["id"] != 1 {
		t.Errorf("counts[id] = %d, want 1 (one file, however many matching lines)", counts["id"])
	}
	if counts["type"] != 1 {
		t.Errorf("counts[type] = %d, want 1", counts["type"])
	}
	if counts["status"] != 0 {
		t.Errorf("counts[status] = %d, want 0", counts["status"])
	}
}

// writeTasks writes 10_projects/<project>/11-tasks.md under vaultDir.
func writeTasks(t *testing.T, vaultDir, project, body string) {
	t.Helper()
	dir := filepath.Join(vaultDir, "10_projects", project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "11-tasks.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestSection7BacklogRunsWithoutAShell pins #492's port: section 7 needs no
// dotfiles checkout and no bash. A drifting tasks file is reported with the
// check's own detail, and the run goes on (#1314): every later file is still
// checked, and the stale-merged pass still runs. The shell stopped at the
// first drift, so a second broken file was never named.
func TestSection7BacklogRunsWithoutAShell(t *testing.T) {
	vaultDir := t.TempDir()
	writeTasks(t, vaultDir, "alpha", "- [ ] **A-1** a\n- [x] **A-1** a\n")
	writeTasks(t, vaultDir, "beta", "- [ ] **B-1** b\n- [x] **B-1** b\n")
	// gamma's G-1 is an archived spec whose tick is still open: the advisory pass's finding.
	writeTasks(t, vaultDir, "gamma", "- [ ] **G-1** shipped\n")
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "Projects", "gamma", "specs", "archive", "G-1"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())

	var buf bytes.Buffer
	h := &healthRun{w: &buf, opts: HealthOptions{VaultDir: vaultDir}}
	h.section7Backlog()

	out := buf.String()
	for _, want := range []string{
		"FAIL: Backlog drift in alpha/11-tasks.md",
		"CONTRADICTION: A-1 — 2 entries, marked BOTH open and done",
		"FAIL: Backlog drift in beta/11-tasks.md",
		"CONTRADICTION: B-1 — 2 entries, marked BOTH open and done",
		"WARN: Stale-merged ticks in gamma/11-tasks.md",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
	if h.failed != 2 {
		t.Errorf("failed = %d, want one per drifting file", h.failed)
	}
	if strings.Contains(out, "task file(s) clean") {
		t.Errorf("a clean PASS beside drift\n%s", out)
	}
}

// TestSection7BacklogNamesAnUnreadableTasksFile: the script exited 2 with
// nothing on stdout, which read as drift with no detail. The port names the
// file and the error, and checks the next file instead of stopping (#1314).
func TestSection7BacklogNamesAnUnreadableTasksFile(t *testing.T) {
	vaultDir := t.TempDir()
	// A directory where the file should be: unreadable as a file on every OS.
	if err := os.MkdirAll(filepath.Join(vaultDir, "10_projects", "alpha", "11-tasks.md"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTasks(t, vaultDir, "beta", "- [ ] **B-1** b\n- [x] **B-1** b\n")

	var buf bytes.Buffer
	h := &healthRun{w: &buf, opts: HealthOptions{VaultDir: vaultDir}}
	h.section7Backlog()

	out := buf.String()
	if !strings.Contains(out, "FAIL: Backlog integrity: cannot read alpha/11-tasks.md: ") ||
		!strings.Contains(out, "FAIL: Backlog drift in beta/11-tasks.md") {
		t.Errorf("want the unreadable file named and the next one still checked\n%s", out)
	}
	if h.failed != 2 {
		t.Errorf("failed = %d, want 2", h.failed)
	}
}

// TestSection7BacklogNoTasksSkips: a vault with no 11-tasks.md is the ordinary
// skip, matching every golden case that has no backlog files at all.
func TestSection7BacklogNoTasksSkips(t *testing.T) {
	vaultDir := t.TempDir()

	var buf bytes.Buffer
	h := &healthRun{w: &buf, opts: HealthOptions{VaultDir: vaultDir}}
	h.section7Backlog()

	if h.failed != 0 || h.skipped != 1 {
		t.Errorf("counters = failed:%d skipped:%d, want failed:0 skipped:1", h.failed, h.skipped)
	}
	if !strings.Contains(buf.String(), "SKIP: Backlog integrity - no 10_projects/*/11-tasks.md found") {
		t.Errorf("output = %q, want the no-files skip line", buf.String())
	}
}

// lockDir makes dir unreadable for the rest of the test. filepath.Glob read
// such a directory as holding nothing (#2797).
func lockDir(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions do not deny reads here")
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) //nolint:gosec // restores a test directory so TempDir can remove it
}

// TestSection7BacklogFailsOverAnUnreadableProjectsTree: "no 11-tasks.md found"
// over a permission error is a skip that hides the error; it is a failure.
func TestSection7BacklogFailsOverAnUnreadableProjectsTree(t *testing.T) {
	t.Run("10_projects itself", func(t *testing.T) {
		vaultDir := t.TempDir()
		writeTasks(t, vaultDir, "beta", "- [ ] **B-1** b\n")
		lockDir(t, filepath.Join(vaultDir, "10_projects"))
		var buf bytes.Buffer
		h := &healthRun{w: &buf, opts: HealthOptions{VaultDir: vaultDir}}
		h.section7Backlog()
		if h.failed != 1 || h.skipped != 0 || !strings.Contains(buf.String(), "FAIL: Backlog integrity: cannot read 10_projects: ") {
			t.Errorf("failed:%d skipped:%d, want the unreadable tree failed\n%s", h.failed, h.skipped, buf.String())
		}
	})
	t.Run("one project of it", func(t *testing.T) {
		vaultDir := t.TempDir()
		writeTasks(t, vaultDir, "alpha", "- [ ] **A-1** a\n")
		writeTasks(t, vaultDir, "beta", "- [ ] **B-1** b\n- [x] **B-1** b\n")
		lockDir(t, filepath.Join(vaultDir, "10_projects", "alpha"))
		var buf bytes.Buffer
		h := &healthRun{w: &buf, opts: HealthOptions{VaultDir: vaultDir}}
		h.section7Backlog()
		out := buf.String()
		if !strings.Contains(out, "FAIL: Backlog integrity: cannot read alpha/11-tasks.md: ") ||
			!strings.Contains(out, "FAIL: Backlog drift in beta/11-tasks.md") {
			t.Errorf("want the unreadable project named and the next one still checked\n%s", out)
		}
	})
}

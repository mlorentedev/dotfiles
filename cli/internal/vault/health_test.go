package vault

import (
	"bytes"
	"os"
	"path/filepath"
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

// TestSection7BacklogRunsWithoutAShell pins #492's port: section 7 needs no
// dotfiles checkout and no bash. A vault whose tasks file drifts is reported
// with the check's own detail, and the run stops there as the shell did
// (#1314).
func TestSection7BacklogRunsWithoutAShell(t *testing.T) {
	vaultDir := t.TempDir()
	tasksDir := filepath.Join(vaultDir, "10_projects", "demo")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, "11-tasks.md"), []byte("- [ ] **X-1** a\n- [x] **X-1** a\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("PATH", t.TempDir())

	var buf bytes.Buffer
	h := &healthRun{w: &buf, opts: HealthOptions{VaultDir: vaultDir}}
	code, aborted := h.section7Backlog()

	if !aborted || code != 1 {
		t.Fatalf("section7Backlog() = (%d, %v), want (1, true): drift ends the run", code, aborted)
	}
	if !strings.Contains(buf.String(), "FAIL: Backlog drift in demo/11-tasks.md") ||
		!strings.Contains(buf.String(), "CONTRADICTION: X-1 — 2 entries, marked BOTH open and done") {
		t.Errorf("output = %q, want the drift FAIL and its detail", buf.String())
	}
}

// TestSection7BacklogNamesAnUnreadableTasksFile: the script exited 2 with
// nothing on stdout, which read as drift with no detail. The port names the
// file and the error, and still stops the run with the script's exit 2.
func TestSection7BacklogNamesAnUnreadableTasksFile(t *testing.T) {
	vaultDir := t.TempDir()
	// A directory where the file should be: unreadable as a file on every OS.
	if err := os.MkdirAll(filepath.Join(vaultDir, "10_projects", "demo", "11-tasks.md"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var buf bytes.Buffer
	h := &healthRun{w: &buf, opts: HealthOptions{VaultDir: vaultDir}}
	code, aborted := h.section7Backlog()

	if !aborted || code != 2 {
		t.Fatalf("section7Backlog() = (%d, %v), want (2, true)", code, aborted)
	}
	if !strings.Contains(buf.String(), "FAIL: Backlog integrity: cannot read demo/11-tasks.md: ") {
		t.Errorf("output = %q, want the unreadable file named with its error", buf.String())
	}
}

// TestSection7BacklogNoTasksSkips: a vault with no 11-tasks.md is the ordinary
// skip, matching every golden case that has no backlog files at all.
func TestSection7BacklogNoTasksSkips(t *testing.T) {
	vaultDir := t.TempDir()

	var buf bytes.Buffer
	h := &healthRun{w: &buf, opts: HealthOptions{VaultDir: vaultDir}}
	code, aborted := h.section7Backlog()

	if aborted || code != 0 {
		t.Fatalf("section7Backlog() = (%d, %v), want (0, false)", code, aborted)
	}
	if h.failed != 0 || h.skipped != 1 {
		t.Errorf("counters = failed:%d skipped:%d, want failed:0 skipped:1", h.failed, h.skipped)
	}
	if !strings.Contains(buf.String(), "SKIP: Backlog integrity - no 10_projects/*/11-tasks.md found") {
		t.Errorf("output = %q, want the no-files skip line", buf.String())
	}
}

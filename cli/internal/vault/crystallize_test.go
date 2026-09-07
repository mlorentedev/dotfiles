package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

// The golden corpus (tests/golden/crystallize/) proves end-to-end byte-parity
// with the shell. These unit tests cover the seams a golden cannot isolate: awk's
// record semantics, and the branch each transform picks.

func TestSplitJoinLinesModelsAwkRecords(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		rejoin  string
	}{
		{"empty", "", nil, ""},
		{"single line with newline", "a\n", []string{"a"}, "a\n"},
		// awk normalises: a file whose last line lacks "\n" still gets one on
		// output. The port must normalise identically or byte-parity breaks on
		// any hand-edited MEMORY.md.
		{"single line without newline", "a", []string{"a"}, "a\n"},
		{"trailing blank line", "a\n\n", []string{"a", ""}, "a\n\n"},
		{"multiple", "a\nb\nc\n", []string{"a", "b", "c"}, "a\nb\nc\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitLines(tc.content)
			if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
				t.Fatalf("splitLines(%q) = %q, want %q", tc.content, got, tc.want)
			}
			if rejoined := joinLines(got); rejoined != tc.rejoin {
				t.Fatalf("joinLines(splitLines(%q)) = %q, want %q", tc.content, rejoined, tc.rejoin)
			}
		})
	}
}

func TestDedupCurrentDate(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			// Not "keeps the latest" despite what the shell's docstring says: it
			// removes every block. UpdateCurrentDate re-adds one afterwards.
			name: "removes all blocks, not all-but-one",
			in:   []string{"# currentDate", "Today's date is 2025-01-01.", "x", "# currentDate", "Today's date is 2025-02-02."},
			want: []string{"x"},
		},
		{
			// awk's bare `skip { skip=0 }` has no `next`, so a marker not followed
			// by a date line drops only the marker; the next line survives.
			name: "marker with no date line keeps the following line",
			in:   []string{"# currentDate", "not a date line", "y"},
			want: []string{"not a date line", "y"},
		},
		{
			name: "unrelated content is untouched",
			in:   []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			// The corollary: dropping the apostrophe entirely does NOT match, so
			// the line survives (only the marker above it is removed).
			name: "an apostrophe-less date line does not match and survives",
			in:   []string{"# currentDate", "Todays date is 2025-01-01.", "z"},
			want: []string{"Todays date is 2025-01-01.", "z"},
		},
		{
			// /^Today.s date is / has an UNESCAPED dot: it matches "Today" + ANY
			// single character + "s date is ". So a curly apostrophe matches, and
			// so does any other glyph -- but "Todays date is" does NOT, because
			// the dot consumes the "s" and the literal "s" then has nothing to
			// match. Reproduced deliberately rather than tightened.
			name: "the unescaped dot matches any character in the apostrophe slot",
			in:   []string{"# currentDate", "Today\u2019s date is 2025-01-01.", "z"},
			want: []string{"z"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DedupCurrentDate(tc.in)
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAppendBeforeHandoffKeepsHandoffLast(t *testing.T) {
	tests := []struct {
		name  string
		in    []string
		block string
		want  []string
	}{
		{
			name:  "inserts before the handoff (HARNESS-029)",
			in:    []string{"intro", "## Session Handoff", "body"},
			block: "NEW",
			want:  []string{"intro", "NEW", "## Session Handoff", "body"},
		},
		{
			name:  "appends at EOF when no handoff exists",
			in:    []string{"intro"},
			block: "NEW",
			want:  []string{"intro", "NEW"},
		},
		{
			name:  "multi-line blocks stay in order",
			in:    []string{"## Session Handoff"},
			block: "A\nB",
			want:  []string{"A", "B", "## Session Handoff"},
		},
		{
			name:  "only the FIRST handoff heading is used",
			in:    []string{"## Session Handoff", "## Session Handoff"},
			block: "NEW",
			want:  []string{"NEW", "## Session Handoff", "## Session Handoff"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AppendBeforeHandoff(tc.in, tc.block)
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUpdateCurrentDate(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		want    []string
		wantLog string
	}{
		{
			name:    "rewrites the date line in place",
			in:      []string{"# currentDate", "Today's date is 2025-01-01."},
			want:    []string{"# currentDate", "Today's date is 2026-08-09."},
			wantLog: "[INFO] Updated currentDate to 2026-08-09",
		},
		{
			// ORACLE DEFECT #873: nothing is written, yet success is reported.
			// Pinned so a port cannot "helpfully" fix it without a recapture.
			name:    "marker with no date line writes nothing but still claims success",
			in:      []string{"# currentDate", "", "## Session Handoff"},
			want:    []string{"# currentDate", "", "## Session Handoff"},
			wantLog: "[INFO] Updated currentDate to 2026-08-09",
		},
		{
			name:    "adds the section before the handoff when absent",
			in:      []string{"intro", "## Session Handoff"},
			want:    []string{"intro", "# currentDate", "Today's date is 2026-08-09.", "## Session Handoff"},
			wantLog: "[INFO] Added currentDate section (2026-08-09)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, log := UpdateCurrentDate(tc.in, "2026-08-09")
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("lines = %q, want %q", got, tc.want)
			}
			if log != tc.wantLog {
				t.Fatalf("log = %q, want %q", log, tc.wantLog)
			}
		})
	}
}

func TestStampLastCrystallized(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		want    []string
		wantLog string
	}{
		{
			name:    "replaces an existing stamp",
			in:      []string{"## Last Crystallized: 2025-01-01", "x"},
			want:    []string{"## Last Crystallized: 2026-08-09", "x"},
			wantLog: "[INFO] Updated Last Crystallized to 2026-08-09",
		},
		{
			// awk's print of a string already ending in "\n" emits a BLANK line
			// after the stamp. Easy to miss, and it is load-bearing for parity.
			name:    "inserts stamp plus a blank line before the currentDate marker",
			in:      []string{"intro", "# currentDate", "Today's date is 2025-01-01."},
			want:    []string{"intro", "## Last Crystallized: 2026-08-09", "", "# currentDate", "Today's date is 2025-01-01."},
			wantLog: "[INFO] Added Last Crystallized stamp (2026-08-09)",
		},
		{
			name:    "falls back to insertion before the handoff",
			in:      []string{"intro", "## Session Handoff"},
			want:    []string{"intro", "## Last Crystallized: 2026-08-09", "## Session Handoff"},
			wantLog: "[INFO] Added Last Crystallized stamp (2026-08-09)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, log := StampLastCrystallized(tc.in, "2026-08-09")
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("lines = %q, want %q", got, tc.want)
			}
			if log != tc.wantLog {
				t.Fatalf("log = %q, want %q", log, tc.wantLog)
			}
		})
	}
}

func TestIsYAMLBlockScalar(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		// Indent width is NOT fixed across projects, so detection must be
		// structural. Four spaces (hive) and six (pollex) must both trip it.
		{"four-space indent", "---\ncontent: |\n    # Memory\n", true},
		{"six-space indent", "---\ncontent: |\n      # Memory\n", true},
		{"block scalar with chomp indicator", "---\nbody: |-\n  x\n", true},
		{"plain markdown", "# Memory\n\n## Index\n", false},
		{"front matter without a block scalar", "---\ntitle: x\n---\n# Memory\n", false},
		{"block scalar but no document start", "content: |\n  x\n", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsYAMLBlockScalar(tc.content); got != tc.want {
				t.Fatalf("IsYAMLBlockScalar(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestNewlineCountMatchesWcL(t *testing.T) {
	// `wc -l` counts newline CHARACTERS, so a file whose last line is unterminated
	// counts one fewer. The line-limit warning inherits that.
	tests := []struct {
		content string
		want    int
	}{
		{"", 0},
		{"a\n", 1},
		{"a", 0},
		{"a\nb\n", 2},
		{"a\nb", 1},
	}
	for _, tc := range tests {
		if got := newlineCount(tc.content); got != tc.want {
			t.Fatalf("newlineCount(%q) = %d, want %d", tc.content, got, tc.want)
		}
	}
}

// decodePath must reverse memlink.ClaudeProjectKey for real directories and give
// up quietly for keys that resolve to nothing — the branch that makes --all
// report a project as skipped rather than crashing.
func TestDecodePathRoundTrip(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "Projects", "demo")
	if err := os.MkdirAll(proj, 0o750); err != nil {
		t.Fatal(err)
	}

	key := memlink.ClaudeProjectKey(proj)
	if got := decodePath(home, key); got != proj {
		t.Fatalf("decodePath(%q) = %q, want %q", key, got, proj)
	}

	if got := decodePath(home, "-no-such-project-anywhere"); got != "" {
		t.Fatalf("decodePath of an unresolvable key = %q, want \"\"", got)
	}
}

// A directory whose name contains a dash cannot be recovered by the naive
// `tr '-' '/'` reversal; the shell falls back to scanning, and so must this.
func TestDecodePathHandlesDashInDirectoryName(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "Projects", "my-repo")
	if err := os.MkdirAll(proj, 0o750); err != nil {
		t.Fatal(err)
	}
	key := memlink.ClaudeProjectKey(proj)
	if got := decodePath(home, key); got != proj {
		t.Fatalf("decodePath(%q) = %q, want %q", key, got, proj)
	}
}

func TestProcessProjectRefusesYAMLAndLeavesFileIntact(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "MEMORY.md")
	original := "---\ncontent: |\n    # Memory\n\n    ## Session Handoff\n"
	if err := os.WriteFile(file, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := ProcessProject(&buf, file, "2026-08-09")
	if err == nil {
		t.Fatal("expected a refusal error")
	}
	// Assert the sentinel, not the text: the cmd layer keys on errors.Is to
	// silence cobra's duplicate "Error:" line, so that contract is what matters.
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("error = %v, want it to wrap ErrRefused", err)
	}

	// "Refusing is strictly better than corrupting" is only true if the file is
	// genuinely untouched.
	after, rerr := os.ReadFile(file) //nolint:gosec // test fixture path
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(after) != original {
		t.Fatalf("refused file was modified:\n%q", string(after))
	}
	if n := strings.Count(buf.String(), "[ERROR]"); n != 4 {
		t.Fatalf("got %d [ERROR] lines, want 4 (the shell prints exactly four)", n)
	}
}

// #1553: a dotted repo name (svqtriana.github.io) encodes with the dot mapped to
// '-', exactly as Claude writes it, so memoryFileFor lands on the MEMORY.md Claude
// maintains and decodePath recovers the directory from that key by scanning.
func TestDottedRepoNameEncodesLikeClaudeAndDecodes(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "Projects", "svqtriana.github.io")
	if err := os.MkdirAll(proj, 0o750); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".claude", "projects", memlink.ClaudeProjectKey(proj), "memory", "MEMORY.md")
	if got := memoryFileFor(home, proj); got != want || strings.Contains(filepath.Base(filepath.Dir(filepath.Dir(got))), ".") {
		t.Fatalf("memoryFileFor(%q) = %q, want a dot-free key: %q", proj, got, want)
	}
	key := memlink.ClaudeProjectKey(proj)
	if got := decodePath(home, key); got != proj {
		t.Fatalf("decodePath(%q) = %q, want %q", key, got, proj)
	}
}

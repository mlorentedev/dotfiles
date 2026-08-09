package memshape

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// wrapCommit is the vault commit that wrapped every affected MEMORY.md in one
// bulk edit (2026-05-26 21:17:41). It holds both sides of the transform: the
// wrapped file, and — in its parent — the plain-markdown original.
const wrapCommit = "1c216229"

// wantPairs is the number of wrapped MEMORY.md files that commit produced.
//
// Asserted exactly, not as "more than zero". Without it, a regression in
// IsWrapped that stopped recognising most files would push them down the
// skipped path and leave a single surviving pair to report success — a green
// run proving almost nothing.
//
// It is larger than the 16 files the live migration touched: this is the
// HISTORICAL corpus, and it includes entries later archived or renamed
// (90_archive/yt-intel), plus files since unwrapped by hand (dotfiles' own).
const wantPairs = 23

type vaultRepo struct {
	t    *testing.T
	path string
}

// openVault locates the knowledge vault and verifies it holds wrapCommit,
// skipping the test when it does not. The ground-truth corpus deliberately
// lives OUTSIDE this repository: dotfiles is public and the vault is private,
// so committing those files would publish infrastructure notes and session
// history. The synthetic fixtures in memshape_test.go cover the same shapes and
// are what CI runs.
func openVault(t *testing.T) *vaultRepo {
	t.Helper()

	path := os.Getenv("VAULT_PATH")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home directory — cannot locate the vault")
		}
		path = filepath.Join(home, "Projects", "knowledge")
	}
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		t.Skipf("vault not available at %q — ground-truth corpus is private and out of repo", path)
	}

	v := &vaultRepo{t: t, path: path}
	if _, err := v.git("cat-file", "-e", wrapCommit+"^{commit}"); err != nil {
		t.Skipf("vault has no commit %s — cannot characterize against ground truth", wrapCommit)
	}
	return v
}

func (v *vaultRepo) git(args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", v.path}, args...)...).Output()
	return string(out), err
}

// pairs returns every (wrapped, original) MEMORY.md pair wrapCommit produced.
func (v *vaultRepo) pairs() map[string][2]string {
	v.t.Helper()

	listing, err := v.git("show", "--name-only", "--format=", wrapCommit)
	if err != nil {
		v.t.Fatalf("listing %s: %v", wrapCommit, err)
	}

	found := make(map[string][2]string)
	for _, path := range strings.Split(strings.TrimSpace(listing), "\n") {
		path = strings.TrimSpace(path)
		if !strings.HasSuffix(path, "memory/MEMORY.md") {
			continue
		}
		wrapped, err := v.git("show", wrapCommit+":"+path)
		if err != nil || !IsWrapped(wrapped) {
			continue
		}
		original, err := v.git("show", wrapCommit+"^:"+path)
		if err != nil {
			continue
		}
		found[path] = [2]string{wrapped, original}
	}
	return found
}

// TestUnwrapAgainstVaultGroundTruth characterizes Unwrap against real
// before/after pairs authored by neither this code nor its author.
func TestUnwrapAgainstVaultGroundTruth(t *testing.T) {
	corpus := openVault(t).pairs()

	if len(corpus) != wantPairs {
		t.Fatalf("recognised %d wrapped files in %s, want %d — the corpus or IsWrapped changed; do not read this as a pass",
			len(corpus), wrapCommit, wantPairs)
	}

	for path, pair := range corpus {
		t.Run(path, func(t *testing.T) {
			assertRecoversOriginal(t, pair[0], pair[1])
		})
	}
}

// assertRecoversOriginal checks that de-indenting the wrapped form reproduces
// the pre-wrap original byte-for-byte, for as far as the original survived.
//
// Equality is not the contract, because the 2026-05-26 wrap was lossy in two
// distinct ways — both measured here, neither previously documented:
//
//  1. It dropped each file's trailing `# currentDate` section, and 169 lines
//     from one work project (#865).
//  2. It truncated MID-LINE. kasa-provisioner's wrapped body ends at `|`, a
//     fragment of the table separator `|-------|--------|`.
//
// So the contract asserted is: every COMPLETE recovered line equals its
// counterpart byte-for-byte, and only the final line may be a prefix of its
// counterpart. That is strictly stronger than trimming trailing newlines and
// prefix-matching the whole string, which is what this test did first — that
// version silently accepted the mid-line truncation as if it were clean, and
// would equally have accepted the transform gaining or losing trailing blank
// lines. Blank-line fidelity where nothing was truncated is pinned separately,
// on the synthetic fixtures.
func assertRecoversOriginal(t *testing.T, wrapped, original string) {
	t.Helper()

	got, err := Unwrap(wrapped)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}

	// afterFrontmatter is applied to BOTH sides: four of these files already
	// carried frontmatter before the wrap, so comparing a body against a whole
	// file would fail them for the wrong reason. It is a no-op on the rest.
	gotBody := afterFrontmatter(got)
	want := afterFrontmatter(original)

	if gotBody == want {
		return
	}

	gl, wl := strings.Split(gotBody, "\n"), strings.Split(want, "\n")

	// A trailing newline splits into a final empty element. Drop it so that the
	// last element compared is the last line of actual CONTENT — which is the
	// one the May wrap may have cut mid-way through.
	if n := len(gl); n > 1 && gl[n-1] == "" {
		gl = gl[:n-1]
	}
	if n := len(wl); n > 1 && wl[n-1] == "" {
		wl = wl[:n-1]
	}

	if len(gl) > len(wl) {
		t.Errorf("recovered %d lines, original had only %d — the transform invented content\n%s",
			len(gl), len(wl), firstDiff(want, gotBody))
		return
	}

	for i := 0; i < len(gl)-1; i++ {
		if gl[i] != wl[i] {
			t.Errorf("complete line %d differs — the de-indent is wrong, not merely truncated\n%s",
				i+1, firstDiff(want, gotBody))
			return
		}
	}

	last := len(gl) - 1
	if !strings.HasPrefix(wl[last], gl[last]) {
		t.Errorf("final recovered line is not a prefix of the original's line %d\n  want: %q\n  got : %q",
			last+1, wl[last], gl[last])
		return
	}
	if gl[last] != wl[last] {
		t.Logf("May wrap truncated MID-LINE at line %d: kept %d of %d bytes", last+1, len(gl[last]), len(wl[last]))
	}
	t.Logf("de-indent exact; the May wrap had dropped %d trailing line(s) before we ever saw the file", len(wl)-len(gl))
}

// afterFrontmatter returns everything after the closing `---` of the leading
// frontmatter block, skipping the single blank line that follows it.
func afterFrontmatter(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		return s
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") != "---" {
			continue
		}
		rest := lines[i+1:]
		if len(rest) > 0 && rest[0] == "" {
			rest = rest[1:]
		}
		return strings.Join(rest, "\n")
	}
	return s
}

func firstDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			// Spaces rendered visibly: this transform is entirely about leading
			// and trailing whitespace, so an invisible diff is useless.
			return fmt.Sprintf("line %d:\n  want: %q\n  got : %q",
				i+1, strings.ReplaceAll(a, " ", "·"), strings.ReplaceAll(b, " ", "·"))
		}
	}
	return "(no line differs; length mismatch only)"
}

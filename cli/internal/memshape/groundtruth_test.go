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

// TestUnwrapAgainstVaultGroundTruth characterizes Unwrap against real
// before/after pairs authored by neither this code nor its author.
//
// The corpus deliberately lives OUTSIDE this repository. dotfiles is public and
// the knowledge vault is private, so committing those files as fixtures would
// publish infrastructure notes, hostnames and session history. The shapes are
// covered by the synthetic fixtures in memshape_test.go, which are safe to
// commit; this test adds the ground-truth check on any machine that has the
// vault, and skips cleanly everywhere else — including CI.
//
// Skipping is not a silent pass: the run prints how many pairs it verified, so
// "0 verified" is visible rather than inferred.
func TestUnwrapAgainstVaultGroundTruth(t *testing.T) {
	vault := os.Getenv("VAULT_PATH")
	if vault == "" {
		if home, err := os.UserHomeDir(); err == nil {
			vault = filepath.Join(home, "Projects", "knowledge")
		}
	}
	if _, err := os.Stat(filepath.Join(vault, ".git")); err != nil {
		t.Skipf("vault not available at %q — ground-truth corpus is private and out of repo", vault)
	}

	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", vault}, args...)...)
		out, err := cmd.Output()
		return string(out), err
	}

	if _, err := git("cat-file", "-e", wrapCommit+"^{commit}"); err != nil {
		t.Skipf("vault has no commit %s — cannot characterize against ground truth", wrapCommit)
	}

	listing, err := git("show", "--name-only", "--format=", wrapCommit)
	if err != nil {
		t.Fatalf("listing %s: %v", wrapCommit, err)
	}

	verified, skipped := 0, 0
	for _, path := range strings.Split(strings.TrimSpace(listing), "\n") {
		path = strings.TrimSpace(path)
		if !strings.HasSuffix(path, "memory/MEMORY.md") {
			continue
		}

		wrapped, err := git("show", wrapCommit+":"+path)
		if err != nil {
			continue
		}
		if !IsWrapped(wrapped) {
			// Touched by that commit but not wrapped by it — not our corpus.
			skipped++
			continue
		}
		original, err := git("show", wrapCommit+"^:"+path)
		if err != nil {
			skipped++
			continue
		}

		t.Run(path, func(t *testing.T) {
			got, err := Unwrap(wrapped)
			if err != nil {
				t.Fatalf("Unwrap: %v", err)
			}
			// The original had no frontmatter; the migrated form keeps the
			// frontmatter the wrapper added, which satisfies the vault's
			// Frontmatter Law. So compare the BODY, which is what the transform
			// is responsible for recovering.
			//
			// The assertion is a PREFIX match, not equality, and that is a
			// finding rather than a convenience: the 2026-05-26 wrap was lossy.
			// It dropped the trailing `# currentDate` section from most files —
			// boilerplates' original ends with `# currentDate` + its date line,
			// and the wrapped form simply does not contain them. So the strongest
			// true statement is that de-indenting recovers the surviving content
			// byte-for-byte, in the original's own formatting.
			// afterFrontmatter is applied to BOTH sides: four of these files
			// already carried frontmatter before the wrap, so comparing a body
			// against a whole file would fail them for the wrong reason. It is a
			// no-op on the files that had none.
			gotBody := strings.TrimRight(afterFrontmatter(got), "\n")
			wantFull := strings.TrimRight(afterFrontmatter(original), "\n")

			if gotBody == wantFull {
				return
			}
			if !strings.HasPrefix(wantFull, gotBody) {
				t.Errorf("recovered body is not the original's prefix — the de-indent is wrong, not merely truncated\n%s",
					firstDiff(wantFull, gotBody))
				return
			}
			dropped := strings.Count(wantFull[len(gotBody):], "\n")
			t.Logf("de-indent exact; the May wrap had dropped %d trailing line(s) before we ever saw the file", dropped)
		})
		verified++
	}

	t.Logf("ground-truth pairs verified: %d (skipped %d)", verified, skipped)
	if verified == 0 {
		t.Errorf("no ground-truth pairs verified — the corpus or the commit changed shape; do not read this as a pass")
	}
}

// afterFrontmatter returns everything after the closing `---` of the leading
// frontmatter block, plus the blank line that follows it.
func afterFrontmatter(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		return s
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == "---" {
			rest := lines[i+1:]
			if len(rest) > 0 && rest[0] == "" {
				rest = rest[1:]
			}
			return strings.Join(rest, "\n")
		}
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
			// Spaces are rendered visibly: this transform is entirely about
			// leading and trailing whitespace, so an invisible diff is useless.
			return fmt.Sprintf("line %d:\n  want: %q\n  got : %q",
				i+1, strings.ReplaceAll(a, " ", "·"), strings.ReplaceAll(b, " ", "·"))
		}
	}
	return "(no line differs; length mismatch only)"
}

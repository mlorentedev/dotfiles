package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// gitIn runs git in dir and returns trimmed stdout, failing the test on error.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "init.defaultBranch=main"}, args...)...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// writeCommit writes body to rel in dir and commits it.
func writeCommit(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", rel)
	gitIn(t, dir, "commit", "-q", "-m", "write "+rel)
}

// refreshFixture is the shape #2162 was measured on: an upstream vault at v2,
// a local clone of it one commit behind at v1, and a dotfiles checkout whose
// committed record already holds v2. The record is cut from the vault's
// pattern, so a refresh from the stale clone reverts it. The checkout has two
// targets, so a guard that reads a fixed list instead of the manifest misses
// the second (it missed ai/orca/ORCA.md).
type refreshFixture struct {
	upstream, vault, repo string
}

const refreshPattern = "00_meta/patterns/pattern-x.md"

func refreshPatternBody(v string) string { return "# Rule\n" + v + "\n" }

const refreshManifest = `{"version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [{"id": "demo", "source": "pattern-x.md#rule"}],
  "targets": [{"file": "TARGET.md", "inject": ["demo"]}, {"file": "ai/OTHER.md", "inject": ["demo"]}]}`

const refreshTarget = "intro\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\noutro\n"

func newRefreshFixture(t *testing.T) refreshFixture {
	t.Helper()
	root := t.TempDir()
	f := refreshFixture{
		upstream: filepath.Join(root, "upstream.git"),
		vault:    filepath.Join(root, "vault"),
		repo:     filepath.Join(root, "dotfiles"),
	}
	gitIn(t, root, "init", "-q", "--bare", f.upstream)
	gitIn(t, root, "clone", "-q", f.upstream, f.vault)
	writeCommit(t, f.vault, refreshPattern, refreshPatternBody("v1"))
	gitIn(t, f.vault, "push", "-q", "origin", "HEAD")

	other := filepath.Join(root, "other")
	gitIn(t, root, "clone", "-q", f.upstream, other)
	writeCommit(t, other, refreshPattern, refreshPatternBody("v2"))
	gitIn(t, other, "push", "-q", "origin", "HEAD")

	if err := os.MkdirAll(f.repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, f.repo, "init", "-q")
	writeCommit(t, f.repo, "harness/manifest.json", refreshManifest)
	writeCommit(t, f.repo, "TARGET.md", refreshTarget)
	writeCommit(t, f.repo, "ai/OTHER.md", refreshTarget)
	// The committed state is the v2 render, as the checkout holds it.
	if _, err := harness.Refresh(f.repo, other); err != nil {
		t.Fatal(err)
	}
	gitIn(t, f.repo, "add", "-A")
	gitIn(t, f.repo, "commit", "-q", "-m", "refresh v2")
	return f
}

func (f refreshFixture) run(t *testing.T) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	err = runHarnessRefresh(&out, &errb, f.repo, f.vault)
	return out.String(), errb.String(), err
}

func (f refreshFixture) record(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.repo, "harness", "enforced", "demo.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestHarnessRefresh_ABehindVaultIsFastForwardedBeforeTheRefreshReadsIt(t *testing.T) {
	f := newRefreshFixture(t)

	stdout, stderr, err := f.run(t)

	if err != nil {
		t.Fatalf("refresh: %v\n%s", err, stderr)
	}
	if got := f.record(t); got != "v2\n" {
		t.Errorf("record = %q, want the upstream's v2: the refresh read the stale clone", got)
	}
	if strings.Contains(stderr, "commit them") {
		t.Errorf("a refresh from a current vault reported drift:\n%s", stderr)
	}
	if !strings.Contains(stdout, "fast-forwarded") {
		t.Errorf("stdout does not say the vault moved:\n%s", stdout)
	}
}

func TestHarnessRefresh_ADivergedVaultIsNotRefreshedFromAndStaysUntouched(t *testing.T) {
	f := newRefreshFixture(t)
	writeCommit(t, f.vault, "00_meta/patterns/local.md", "local\n")
	head := gitIn(t, f.vault, "rev-parse", "HEAD")

	_, stderr, err := f.run(t)

	if err != nil {
		t.Fatalf("a skip is not an error: %v", err)
	}
	if !strings.Contains(stderr, "diverged") {
		t.Errorf("the warning does not name the reason:\n%s", stderr)
	}
	if got := f.record(t); got != "v2\n" {
		t.Errorf("record = %q: the refresh ran against a diverged vault", got)
	}
	if got := gitIn(t, f.vault, "rev-parse", "HEAD"); got != head {
		t.Errorf("vault HEAD moved from %s to %s", head, got)
	}
}

func TestHarnessRefresh_AnAheadVaultIsRefreshedFromAndItsChangeAnnounced(t *testing.T) {
	f := newRefreshFixture(t)
	gitIn(t, f.vault, "pull", "-q", "--ff-only")
	writeCommit(t, f.vault, refreshPattern, refreshPatternBody("v3"))

	_, stderr, err := f.run(t)

	if err != nil {
		t.Fatalf("refresh: %v\n%s", err, stderr)
	}
	if got := f.record(t); got != "v3\n" {
		t.Errorf("record = %q, want v3 from the vault's unpushed commit", got)
	}
	for _, want := range []string{"Harness records changed by --refresh", "harness/enforced/demo.md", "chore(harness): refresh records from vault"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("drift report lacks %q:\n%s", want, stderr)
		}
	}
}

func TestHarnessRefresh_WithoutAVaultTheCommittedRecordsStand(t *testing.T) {
	f := newRefreshFixture(t)
	f.vault = filepath.Join(t.TempDir(), "absent")

	stdout, _, err := f.run(t)

	if err != nil || !strings.Contains(stdout, "no vault") {
		t.Fatalf("err = %v, stdout = %q", err, stdout)
	}
	if got := f.record(t); got != "v2\n" {
		t.Errorf("record = %q: something ran without a vault", got)
	}
}

func TestHarnessRefresh_AFailedRenderIsAnErrorAndWritesNothing(t *testing.T) {
	f := newRefreshFixture(t)
	// The second target loses its END marker; the first is fine and comes
	// first, so a refresh that wrote as it went would have rewritten it.
	writeCommit(t, f.repo, "ai/OTHER.md", "intro\n<!-- BEGIN HARNESS GENERATED -->\n")
	before := gitIn(t, f.repo, "rev-parse", "HEAD:TARGET.md")

	_, _, err := f.run(t)

	if err == nil || !strings.Contains(err.Error(), "END") {
		t.Fatalf("err = %v, want the marker error", err)
	}
	if got := f.record(t); got != "v2\n" {
		t.Errorf("record = %q: a failed refresh wrote it", got)
	}
	if dirt := gitIn(t, f.repo, "status", "--porcelain"); dirt != "" {
		t.Errorf("a failed refresh left changes (TARGET.md was %s):\n%s", before, dirt)
	}
}

func TestHarnessRefresh_ADirtyVaultIsNotRefreshedFromAndTheWarningNamesItsPaths(t *testing.T) {
	f := newRefreshFixture(t)
	if err := os.WriteFile(filepath.Join(f.vault, "00_meta", "patterns", "draft.md"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := f.run(t)

	if err != nil {
		t.Fatalf("a skip is not an error: %v", err)
	}
	if !strings.Contains(stderr, "dirty worktree") || !strings.Contains(stderr, "      ?? 00_meta/patterns/draft.md\n") {
		t.Errorf("the warning does not name the dirtying path on its own line:\n%s", stderr)
	}
	if got := f.record(t); got != "v2\n" {
		t.Errorf("record = %q: the refresh ran against a dirty vault", got)
	}
}

func TestHarnessRefresh_UncommittedWorkInTheRecordsIsNeitherOverwrittenNorAnnounced(t *testing.T) {
	f := newRefreshFixture(t)
	if err := os.WriteFile(filepath.Join(f.repo, "harness", "enforced", "demo.md"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := f.run(t)

	if err != nil {
		t.Fatalf("a skip is not an error: %v", err)
	}
	if got := f.record(t); got != "wip\n" {
		t.Errorf("record = %q: the refresh overwrote uncommitted work", got)
	}
	if strings.Contains(stderr, "chore(harness): refresh records from vault") {
		t.Errorf("uncommitted work was announced as the vault's change:\n%s", stderr)
	}
	// The six-space indent, then the porcelain line with its index column
	// intact: a line, not a substring, so a trimmed column fails it (#2177).
	if !slices.Contains(strings.Split(stderr, "\n"), "       M harness/enforced/demo.md") {
		t.Errorf("the warning does not name the uncommitted path in porcelain form:\n%s", stderr)
	}
}

func TestHarnessRefresh_UncommittedWorkInAnyManifestTargetStopsTheRefresh(t *testing.T) {
	f := newRefreshFixture(t)
	other := filepath.Join(f.repo, "ai", "OTHER.md")
	if err := os.WriteFile(other, []byte(refreshTarget+"wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := f.run(t)

	if err != nil {
		t.Fatalf("a skip is not an error: %v", err)
	}
	if b, _ := os.ReadFile(other); !strings.HasSuffix(string(b), "wip\n") {
		t.Errorf("the refresh overwrote uncommitted work in a target the old fixed list did not name:\n%s", b)
	}
	if !slices.Contains(strings.Split(stderr, "\n"), "       M ai/OTHER.md") {
		t.Errorf("the warning does not name the second target:\n%s", stderr)
	}
}

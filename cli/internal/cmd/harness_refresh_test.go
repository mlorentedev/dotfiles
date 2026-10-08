package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
// committed record already holds v2. Its compile-harness.sh stand-in renders
// the record from $VAULT_PATH, so a refresh from the stale clone reverts it.
type refreshFixture struct {
	upstream, vault, repo string
}

const refreshPattern = "00_meta/patterns/pattern-x.md"

func newRefreshFixture(t *testing.T) refreshFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the compile-harness.sh stand-in is a POSIX script")
	}
	root := t.TempDir()
	f := refreshFixture{
		upstream: filepath.Join(root, "upstream.git"),
		vault:    filepath.Join(root, "vault"),
		repo:     filepath.Join(root, "dotfiles"),
	}
	gitIn(t, root, "init", "-q", "--bare", f.upstream)
	gitIn(t, root, "clone", "-q", f.upstream, f.vault)
	writeCommit(t, f.vault, refreshPattern, "v1\n")
	gitIn(t, f.vault, "push", "-q", "origin", "HEAD")

	other := filepath.Join(root, "other")
	gitIn(t, root, "clone", "-q", f.upstream, other)
	writeCommit(t, other, refreshPattern, "v2\n")
	gitIn(t, other, "push", "-q", "origin", "HEAD")

	if err := os.MkdirAll(f.repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, f.repo, "init", "-q")
	writeCommit(t, f.repo, "harness/record.md", "v2\n")
	script := "#!/bin/sh\ncat \"$VAULT_PATH/" + refreshPattern + "\" > harness/record.md\n"
	writeCommit(t, f.repo, "scripts/compile-harness.sh", script)
	if err := os.Chmod(filepath.Join(f.repo, "scripts", "compile-harness.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
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
	b, err := os.ReadFile(filepath.Join(f.repo, "harness", "record.md"))
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
	writeCommit(t, f.vault, refreshPattern, "v3\n")

	_, stderr, err := f.run(t)

	if err != nil {
		t.Fatalf("refresh: %v\n%s", err, stderr)
	}
	if got := f.record(t); got != "v3\n" {
		t.Errorf("record = %q, want v3 from the vault's unpushed commit", got)
	}
	for _, want := range []string{"Harness records changed by --refresh", "harness/record.md", "chore(harness): refresh records from vault"} {
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

func TestHarnessRefresh_AFailedScriptIsAnErrorWithItsOutput(t *testing.T) {
	f := newRefreshFixture(t)
	script := filepath.Join(f.repo, "scripts", "compile-harness.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'marker missing' >&2\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := f.run(t)

	if err == nil {
		t.Fatal("a failed compile-harness run returned nil")
	}
	if !strings.Contains(stderr, "marker missing") {
		t.Errorf("the script's own output was not relayed:\n%s", stderr)
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

package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// fakeGit answers the two questions ScanOrphans asks: whether the checkout is
// shallow, and which paths its history deleted.
func fakeGit(t *testing.T, shallow string, deleted ...string) GitRunner {
	t.Helper()
	return func(_ string, args ...string) (string, error) {
		switch {
		case len(args) > 0 && args[0] == "rev-parse":
			return shallow + "\n", nil
		case len(args) > 0 && args[0] == "log":
			return strings.Join(deleted, "\n") + "\n", nil
		case len(args) > 0 && args[0] == "ls-files":
			return "", nil
		}
		t.Fatalf("unexpected git call: %v", args)
		return "", nil
	}
}

// gitIn runs git in repo with the user's global and system config out of the
// way, so a machine's hooks or commit signing cannot change what a test sees.
func gitIn(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestScanOrphans_SplitsDeletedLeftoversFromFilesGitNeverTracked(t *testing.T) {
	repo, deploy := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(repo, ".zsh", "aliases.zsh"), "alias\n")
	writeFile(t, filepath.Join(deploy, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(deploy, "scripts", "old.sh"), "retired\n")
	writeFile(t, filepath.Join(deploy, "scripts", "mine.sh"), "the user's own\n")
	writeFile(t, filepath.Join(deploy, ".zsh", "gone.zsh"), "retired\n")
	// sensitive/ is outside the pruned trees: env-mapping.conf there is
	// machine-local state (#802), so it is never even scanned.
	writeFile(t, filepath.Join(deploy, "sensitive", "env-mapping.conf"), "local\n")

	got, err := ScanOrphans(repo, deploy, fakeGit(t, "false", "scripts/old.sh", ".zsh/gone.zsh", "sensitive/env-mapping.conf"))
	if err != nil {
		t.Fatal(err)
	}
	want := Orphans{Deleted: []string{".zsh/gone.zsh", "scripts/old.sh"}, Unknown: []string{"scripts/mine.sh"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// A converged deploy dir reads no history: the log is read only to classify
// an orphan, and most runs find none. The ignored-file listing is the one call.
func TestScanOrphans_ReadsNoHistoryWhenNothingIsOrphaned(t *testing.T) {
	repo, deploy := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(deploy, "scripts", "live.sh"), "live\n")

	never := func(_ string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "ls-files" {
			return "", nil
		}
		t.Fatalf("git %v called with no orphan to classify", args)
		return "", nil
	}
	got, err := ScanOrphans(repo, deploy, never)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Deleted)+len(got.Unknown) != 0 || got.Skipped != "" {
		t.Errorf("want nothing, got %+v", got)
	}
}

// A shallow clone's log is missing the commits that deleted things, so an
// empty answer would read as "git never tracked it". Nothing is pruned on it.
func TestScanOrphans_AShallowCloneProvesNothing(t *testing.T) {
	repo, deploy := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(deploy, "scripts", "old.sh"), "retired\n")

	got, err := ScanOrphans(repo, deploy, fakeGit(t, "true", "scripts/old.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Deleted) != 0 || !reflect.DeepEqual(got.Unknown, []string{"scripts/old.sh"}) {
		t.Errorf("a shallow clone must prune nothing: %+v", got)
	}
	if !strings.Contains(got.Skipped, "shallow") {
		t.Errorf("Skipped should say why: %q", got.Skipped)
	}
}

// Without readable history (a tarball checkout, no git) every orphan is
// unknown and the reason is named.
func TestScanOrphans_UnreadableHistoryPrunesNothing(t *testing.T) {
	repo, deploy := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(deploy, "scripts", "old.sh"), "retired\n")

	broken := func(string, ...string) (string, error) { return "", os.ErrNotExist }
	got, err := ScanOrphans(repo, deploy, broken)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Deleted) != 0 || len(got.Unknown) != 1 || got.Skipped == "" {
		t.Errorf("got %+v", got)
	}
}

// A checkout without the tree is the wrong checkout or a broken one; reading
// every deployed file there as an orphan would prune the whole tree.
func TestScanOrphans_SkipsATreeTheCheckoutDoesNotHave(t *testing.T) {
	repo, deploy := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(deploy, "ssh", "config"), "Host *\n")

	got, err := ScanOrphans(repo, deploy, fakeGit(t, "false", "ssh/config"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Deleted)+len(got.Unknown) != 0 {
		t.Errorf("ssh/ is absent from the checkout, so it must not be scanned: %+v", got)
	}
}

// An entry the scan cannot read is named and skipped. It must not fail the
// scan, and with it the whole mirror: the mirror never read the deploy dir
// before it pruned, so one locked directory there would be a new way to fail.
func TestScanOrphans_NamesAnUnreadableDirectoryAndScansTheRest(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions enforced on the test user")
	}
	repo, deploy := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(deploy, "scripts", "old.sh"), "retired\n")
	locked := filepath.Join(deploy, "scripts", "locked")
	writeFile(t, filepath.Join(locked, "x.sh"), "x\n")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	got, err := ScanOrphans(repo, deploy, fakeGit(t, "false", "scripts/old.sh"))
	if err != nil {
		t.Fatalf("an unreadable entry failed the scan: %v", err)
	}
	if !reflect.DeepEqual(got.Unreadable, []string{"scripts/locked"}) {
		t.Errorf("unreadable = %v, want [scripts/locked]", got.Unreadable)
	}
	if !reflect.DeepEqual(got.Deleted, []string{"scripts/old.sh"}) {
		t.Errorf("the rest of the tree was not scanned: %+v", got)
	}
}

func TestPruneOrphans_RemovesTheFilesAndTheDirectoriesTheyEmptied(t *testing.T) {
	deploy := t.TempDir()
	writeFile(t, filepath.Join(deploy, "scripts", "old", "nested", "a.sh"), "x\n")
	writeFile(t, filepath.Join(deploy, "scripts", "keep", "b.sh"), "x\n")
	writeFile(t, filepath.Join(deploy, "scripts", "keep", "c.sh"), "x\n")

	if err := PruneOrphans(deploy, []string{"scripts/old/nested/a.sh", "scripts/keep/b.sh"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(deploy, "scripts", "old")); !os.IsNotExist(err) {
		t.Error("scripts/old held only the pruned file, so it should be gone")
	}
	if _, err := os.Stat(filepath.Join(deploy, "scripts", "keep", "c.sh")); err != nil {
		t.Error("a sibling of a pruned file must survive")
	}
	if _, err := os.Stat(filepath.Join(deploy, "scripts")); err != nil {
		t.Error("the tree root itself is never removed")
	}
}

// A leftover the user cannot remove is reported, and does not shield the
// leftovers after it.
func TestPruneOrphans_ReportsALeftoverItCannotRemoveAndPrunesTheRest(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions enforced on the test user")
	}
	deploy := t.TempDir()
	ro := filepath.Join(deploy, "scripts", "ro")
	writeFile(t, filepath.Join(ro, "old.sh"), "x\n")
	writeFile(t, filepath.Join(deploy, "scripts", "z.sh"), "x\n")
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })

	err := PruneOrphans(deploy, []string{"scripts/ro/old.sh", "scripts/z.sh"})
	if err == nil || !strings.Contains(err.Error(), "scripts/ro/old.sh") {
		t.Errorf("the leftover it could not remove must be named, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(deploy, "scripts", "z.sh")); !os.IsNotExist(err) {
		t.Error("a failure on one leftover stopped the prune of the next")
	}
}

// PruneOrphans refuses a path outside the pruned trees, so a caller that
// passed the wrong list cannot reach sensitive/ or climb out of the deploy dir.
func TestPruneOrphans_RefusesAPathOutsideThePrunedTrees(t *testing.T) {
	deploy := t.TempDir()
	secret := filepath.Join(deploy, "sensitive", "a.secret.age")
	writeFile(t, secret, "x\n")
	for _, rel := range []string{"sensitive/a.secret.age", "scripts/../sensitive/a.secret.age", "../x"} {
		if err := PruneOrphans(deploy, []string{rel}); err == nil {
			t.Errorf("PruneOrphans(%q) succeeded, want a refusal", rel)
		}
	}
	if _, err := os.Stat(secret); err != nil {
		t.Error("the secret was removed")
	}
}

// The real git, end to end: the history query sees a deletion and a rename
// (whose old path git would otherwise report as R, not D), and Mirror prunes
// both leftovers, keeps the user's own file, and prunes nothing on a re-run.
func TestMirror_PrunesLeftoversTheCheckoutDeletedAndARerunPrunesNothing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := mirrorRepo(t)
	git := func(args ...string) { t.Helper(); gitIn(t, repo, args...) }
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(repo, "scripts", "old.sh"), "retired\n")
	writeFile(t, filepath.Join(repo, "scripts", "before.sh"), "renamed\n")
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "one")

	deploy := t.TempDir()
	if _, err := Mirror(repo, deploy); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(deploy, "scripts", "mine.sh"), "the user's own\n")
	git("rm", "-q", "scripts/old.sh")
	git("mv", "scripts/before.sh", "scripts/after.sh")
	git("commit", "-q", "-m", "two")

	res, err := Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"scripts/before.sh", "scripts/old.sh"}; !reflect.DeepEqual(res.Pruned, want) {
		t.Errorf("pruned %v, want %v", res.Pruned, want)
	}
	if want := []string{"scripts/mine.sh"}; !reflect.DeepEqual(res.Unpruned, want) {
		t.Errorf("unpruned %v, want %v", res.Unpruned, want)
	}
	for _, rel := range []string{"scripts/old.sh", "scripts/before.sh"} {
		if _, err := os.Stat(filepath.Join(deploy, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("%s survived the prune", rel)
		}
	}

	again, err := Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Pruned) != 0 || again.Updated != 0 {
		t.Errorf("re-run: want 0 pruned / 0 updated, got %v / %d", again.Pruned, again.Updated)
	}
}

// A file the checkout ignores is local to it (scripts/CLAUDE.md, written by a
// retired memory tool, #2268): the mirror never deploys it, and a copy an
// older mirror deployed is named as an orphan rather than read as part of the
// set.
// git never tracked it, so it is not pruned.
func TestMirror_SkipsAFileTheCheckoutIgnores(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := mirrorRepo(t)
	writeFile(t, filepath.Join(repo, ".gitignore"), "scripts/CLAUDE.md\n")
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "one")
	writeFile(t, filepath.Join(repo, "scripts", "CLAUDE.md"), "local\n")

	deploy := t.TempDir()
	if _, err := Mirror(repo, deploy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(deploy, "scripts", "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("an ignored file was mirrored")
	}
	if _, err := os.Stat(filepath.Join(deploy, "scripts", "live.sh")); err != nil {
		t.Error("a tracked file was not mirrored")
	}

	// The copy an older mirror left is an orphan, named and kept.
	writeFile(t, filepath.Join(deploy, "scripts", "CLAUDE.md"), "local\n")
	res, err := Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Unpruned, []string{"scripts/CLAUDE.md"}) || len(res.Pruned) != 0 {
		t.Errorf("pruned %v, unpruned %v; want the stale copy named and kept", res.Pruned, res.Unpruned)
	}
}

// Without .git (a tarball checkout) nothing counts as ignored, silently, so the
// mirror copies the working tree as it always did. Inside a git checkout a
// failing git ignores nothing too, but says why, so the copy is not silent.
func TestIgnoredInCheckout_FailsOpenAndSaysSoOnlyInAGitCheckout(t *testing.T) {
	broken := func(string, ...string) (string, error) { return "", os.ErrPermission }

	// A tarball is decided before git is asked: an enclosing repository would
	// answer, and its ignore rules are not the checkout's.
	enclosing := func(string, ...string) (string, error) {
		t.Error("git was asked about a checkout without .git")
		return "scripts/live.sh\x00", nil
	}
	tarball := t.TempDir()
	if got, why := IgnoredInCheckout(tarball, enclosing); len(got) != 0 || why != "" {
		t.Errorf("tarball: got %v, %q; want nothing ignored and nothing to report", got, why)
	}

	checkout := t.TempDir()
	writeFile(t, filepath.Join(checkout, ".git"), "gitdir: /nowhere\n")
	got, why := IgnoredInCheckout(checkout, broken)
	if len(got) != 0 || !strings.Contains(why, "so they were mirrored") {
		t.Errorf("git checkout: got %v, %q; want nothing ignored and the failure named", got, why)
	}
}

// A .git that cannot be checked is not a tarball: the mirror would copy the
// ignored files, so the reason is reported, as a failing git's is.
func TestIgnoredInCheckout_ReportsAGitDirItCannotCheck(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a directory the test user cannot search")
	}
	parent := t.TempDir()
	checkout := filepath.Join(parent, "checkout")
	writeFile(t, filepath.Join(checkout, ".git"), "gitdir: /nowhere\n")
	if err := os.Chmod(parent, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	never := func(string, ...string) (string, error) {
		t.Error("git was asked about a checkout whose .git could not be checked")
		return "", nil
	}

	got, why := IgnoredInCheckout(checkout, never)
	if len(got) != 0 || !strings.Contains(why, ".git could not be checked") {
		t.Errorf("got %v, %q; want nothing ignored and the reason named", got, why)
	}
}

// The no-git path end to end, with the real git: a checkout without .git
// mirrors every file and reports nothing, even extracted inside another
// repository. git walks up from such a tree, so asking it would apply the
// enclosing repository's ignore rules, and a catch-all one skips every file.
func TestMirror_CopiesTheWorkingTreeOfATarballCheckout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	// A ceiling in the environment would stop git short of the enclosing
	// repository and hide the defect this test exists for.
	t.Setenv("GIT_CEILING_DIRECTORIES", "")
	repo := mirrorRepo(t)
	enclosing := filepath.Dir(repo)
	gitIn(t, enclosing, "init", "-q")
	writeFile(t, filepath.Join(enclosing, ".gitignore"), "*\n")
	writeFile(t, filepath.Join(repo, "scripts", "CLAUDE.md"), "local\n")

	deploy := t.TempDir()
	res, err := Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"scripts/CLAUDE.md", "harness/skills/handoff/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(deploy, filepath.FromSlash(rel))); err != nil {
			t.Errorf("a tarball checkout's %s was not mirrored: %v", rel, err)
		}
	}
	if res.IgnoreSkipped != "" {
		t.Errorf("a tarball checkout reported %q; it has no git to fail", res.IgnoreSkipped)
	}
}

// The plan names what Mirror would prune and removes nothing, so converge's
// dry run and its probe see a leftover as a pending change.
func TestPlanMirror_NamesALeftoverAndKeepsIt(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := mirrorRepo(t)
	writeFile(t, filepath.Join(repo, "scripts", "old.sh"), "retired\n")
	for _, args := range [][]string{
		{"init", "-q"}, {"add", "-A"}, {"commit", "-q", "-m", "one"}, {"rm", "-q", "scripts/old.sh"}, {"commit", "-q", "-m", "two"},
	} {
		gitIn(t, repo, args...)
	}
	// scripts/ now has no file in the checkout, so keep one there: a tree the
	// checkout lacks is not scanned.
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	deploy := t.TempDir()
	stale := filepath.Join(deploy, "scripts", "old.sh")
	writeFile(t, stale, "retired\n")

	res, err := PlanMirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Pruned, []string{"scripts/old.sh"}) {
		t.Errorf("plan pruned %v", res.Pruned)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Error("a plan must not remove anything")
	}
}

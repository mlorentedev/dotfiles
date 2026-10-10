package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

// A converged deploy dir costs no git call: the history is read only to
// classify an orphan, and most runs find none.
func TestScanOrphans_ReadsNoHistoryWhenNothingIsOrphaned(t *testing.T) {
	repo, deploy := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live\n")
	writeFile(t, filepath.Join(deploy, "scripts", "live.sh"), "live\n")

	never := func(string, ...string) (string, error) {
		t.Fatal("git called with no orphan to classify")
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

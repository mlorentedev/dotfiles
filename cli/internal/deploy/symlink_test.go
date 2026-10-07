package deploy

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// #2054: a destination that is a symlink (the pre-ADR-012 leftover) is an
// alias, not a deployed file, whatever it resolves to.

// linkedDst points the pi config's destination at target, skipping where the
// platform refuses to create links (Windows without the privilege).
func linkedDst(t *testing.T, home, target string) string {
	t.Helper()
	dst := filepath.Join(home, ".pi", "agent", "models.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, dst); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	return dst
}

func writeTarget(t *testing.T, body string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), "models.json")
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return target
}

func assertRegular(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Errorf("%s is %v, want a regular file", path, info.Mode().Type())
	}
}

func TestPlanConfig_ASymlinkWithTheSourceContentIsChanged(t *testing.T) {
	const body = `{"k":"v"}`
	root, home := repoWithSource(t, body), t.TempDir()
	linkedDst(t, home, writeTarget(t, body))

	for _, strategy := range []string{StrategyReplace, StrategyMerge} {
		c := piConfig()
		c.Strategy = strategy
		p, err := PlanConfig(c, root, home, noResolve)
		if err != nil {
			t.Fatal(err)
		}
		if !p.Symlink || !p.Changed {
			t.Errorf("%s: a link resolving to the source's bytes must plan as Symlink and Changed, got %+v", strategy, p)
		}
	}
}

func TestDeploy_ReplacesASymlinkAndKeepsItAsTheBackup(t *testing.T) {
	const body = `{"k":"v"}`
	root, home := repoWithSource(t, body), t.TempDir()
	target := writeTarget(t, body)
	dst := linkedDst(t, home, target)

	res, err := Deploy(piConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Error("replacing a link must report a change")
	}
	assertRegular(t, dst)
	if got, err := os.Readlink(res.BackedUp); err != nil || got != target {
		t.Errorf("backup %q must be a link to %s, got %q, %v", res.BackedUp, target, got, err)
	}
	if got, _ := os.ReadFile(target); string(got) != body {
		t.Errorf("the link's target was modified: %q", got)
	}

	again, err := Deploy(piConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || again.BackedUp != "" {
		t.Errorf("a second deploy must change nothing, got %+v", again)
	}
	if got, _ := os.Readlink(dst + BackupSuffix); got != target {
		t.Errorf("the second deploy touched the backup: now %q", got)
	}
}

func TestDeploy_ReplacesADanglingSymlink(t *testing.T) {
	root, home := repoWithSource(t, `{"k":"v"}`), t.TempDir()
	missing := filepath.Join(t.TempDir(), "gone")
	dst := linkedDst(t, home, missing)

	res, err := Deploy(piConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	assertRegular(t, dst)
	if got, err := os.Readlink(res.BackedUp); err != nil || got != missing {
		t.Errorf("a dangling link must still be kept as a link to %s, got %q, %v", missing, got, err)
	}
}

func TestDeploy_ARenderedConfigBehindASymlinkIsNotInSync(t *testing.T) {
	const body = `{"k":"v"}`
	root, home := repoWithSource(t, body), t.TempDir()
	dst := linkedDst(t, home, writeTarget(t, body))
	c := piConfig()
	c.Render = true

	res, err := Deploy(c, root, home, noResolve, func(string) error { return nil }, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Error("a rendered config behind a link must not read as in sync")
	}
	assertRegular(t, dst)
}

// Where links cannot be created, the backup keeps what the link resolved to,
// with the config's declared mode: a 0644 copy of a credential must not sit
// beside the 0600 file that replaces it.
func TestDeploy_KeepsTheLinkedContentWhereLinksCannotBeCreated(t *testing.T) {
	root, home := repoWithSource(t, `{"k":"new"}`), t.TempDir()
	dst := linkedDst(t, home, writeTarget(t, `{"k":"old"}`))
	symlink = func(string, string) error { return os.ErrPermission }
	t.Cleanup(func() { symlink = os.Symlink })

	res, err := Deploy(piConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	assertRegular(t, dst)
	assertRegular(t, res.BackedUp)
	if got, _ := os.ReadFile(res.BackedUp); string(got) != `{"k":"old"}` {
		t.Errorf("backup must hold the linked content, got %q", got)
	}
	if runtime.GOOS == "windows" {
		return // POSIX permission bits are not meaningful here
	}
	info, err := os.Stat(res.BackedUp)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("backup must take the declared 0600, got %o", info.Mode().Perm())
	}
}

// A dangling link there has no content to keep, and the file it named is
// untouched: the deploy goes ahead without a backup rather than failing.
func TestDeploy_ADanglingLinkWhereLinksCannotBeCreatedNeedsNoBackup(t *testing.T) {
	root, home := repoWithSource(t, `{"k":"v"}`), t.TempDir()
	dst := linkedDst(t, home, filepath.Join(t.TempDir(), "gone"))
	symlink = func(string, string) error { return os.ErrPermission }
	t.Cleanup(func() { symlink = os.Symlink })

	res, err := Deploy(piConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	assertRegular(t, dst)
	if res.BackedUp != "" {
		t.Errorf("a dangling link has nothing to keep, got backup %q", res.BackedUp)
	}
}

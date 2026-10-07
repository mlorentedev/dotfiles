package mem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setRemoteHead records a remote's default branch the way `git clone` and
// `git remote set-head` do: a loose symref, since packed-refs holds no symrefs.
func setRemoteHead(t *testing.T, commonDir, remote, branch string) {
	t.Helper()
	dir := filepath.Join(commonDir, "refs", "remotes", remote)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte("ref: refs/remotes/"+remote+"/"+branch+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// #2089: the default branch is whatever any remote recorded, not only origin's.
// A clone made with `--origin upstream`, or a fork with origin and upstream,
// left a non-main default unqualified, and two machines shared one key: the
// #1921 overwrite on a narrower path. Qualifying a branch that is some
// remote's default is the safe direction: the worst case is two threads where
// one would do, never one thread two machines overwrite.
func TestThreadKeyQualifiesADefaultBranchRecordedByAnyRemote(t *testing.T) {
	host := "@" + shortHost()
	for _, tc := range []struct {
		name, branch string
		heads        map[string]string // remote -> its recorded default
		qualified    bool
	}{
		{"cloned with --origin upstream", "develop", map[string]string{"upstream": "develop"}, true},
		{"a fork: origin main, upstream develop", "develop", map[string]string{"origin": "main", "upstream": "develop"}, true},
		{"a feature branch in that fork", "feat/x", map[string]string{"origin": "main", "upstream": "develop"}, false},
		{"no remote recorded a HEAD", "develop", nil, false},
		{"a remote whose name has a slash", "develop", map[string]string{"foo/bar": "develop"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := mainFixture(t, "proj", tc.branch)
			for remote, head := range tc.heads {
				setRemoteHead(t, filepath.Join(cwd, ".git"), remote, head)
			}
			got := ThreadKey(cwd)
			if qualified := strings.HasSuffix(got, host); qualified != tc.qualified {
				t.Errorf("ThreadKey = %q; host-qualified = %v, want %v", got, qualified, tc.qualified)
			}
		})
	}
}

// A HEAD file that is not a symref into its own remote (a stray file, a
// hand-edited ref) names no default.
func TestRemoteDefaultBranchesIgnoresAHeadThatIsNotARemoteSymref(t *testing.T) {
	common := filepath.Join(t.TempDir(), ".git")
	dir := filepath.Join(common, "refs", "remotes", "origin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte("ref: refs/remotes/other/develop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := remoteDefaultBranches(common); len(got) != 0 {
		t.Errorf("remoteDefaultBranches = %v, want none", got)
	}
}

// #2089, decided: a cwd inside a submodule resolves to the superproject. The
// vault is keyed by project and a submodule rarely has a vault project of its
// own, so resolving to the submodule would move the handoff into a project
// memory that does not exist, where session-end writes nothing. A submodule's
// `.git` file points into `<super>/.git/modules/<name>`, which is not a linked
// worktree, so the walk continues to the superproject.
func TestRepoIdentityResolvesASubmoduleToItsSuperproject(t *testing.T) {
	super := mainFixture(t, "super", "main")
	modDir := filepath.Join(super, ".git", "modules", "sub")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "HEAD"), []byte("ref: refs/heads/feat-sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(super, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, ".git"), []byte("gitdir: ../.git/modules/sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	id, ok := RepoIdentity(sub)
	if !ok || id.Project != "super" || id.Branch != "main" {
		t.Errorf("RepoIdentity(submodule) = %+v, %v; want the superproject super on main", id, ok)
	}
}

// A remote-tracking branch named `<something>/HEAD` is a file called HEAD
// holding a sha, not a symref, and names no default.
func TestRemoteDefaultBranchesIgnoresABranchThatEndsInHead(t *testing.T) {
	common := filepath.Join(t.TempDir(), ".git")
	dir := filepath.Join(common, "refs", "remotes", "origin", "feature")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte("0123456789abcdef0123456789abcdef01234567\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := remoteDefaultBranches(common); len(got) != 0 {
		t.Errorf("remoteDefaultBranches = %v, want none", got)
	}
}

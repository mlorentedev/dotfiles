package lane

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// activateRecord writes the activation record `dotf converge` leaves in the git
// common dir: with it the repository is lane-managed on this machine, whatever
// the working tree carries.
func activateRecord(t *testing.T, repoDir string) {
	t.Helper()
	if err := os.WriteFile(recordPath(t, repoDir), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// initialisedLaneRepo is a repository with the three lanes provisioned and the
// marker in place, the starting state of every audit case that breaks one thing.
// Init no longer writes the marker, so the fixture does.
func initialisedLaneRepo(t *testing.T) (repoDir, wtRoot string) {
	t.Helper()
	repoDir, wtRoot = setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("lane init: %v", err)
	}
	if _, err := writeLanesMarker(repoDir); err != nil {
		t.Fatal(err)
	}
	activateRecord(t, repoDir)
	return repoDir, wtRoot
}

func audit(t *testing.T, repoDir, wtRoot string) Audit {
	t.Helper()
	a, err := AuditLanes(repoDir, wtRoot)
	if err != nil {
		t.Fatalf("AuditLanes: %v", err)
	}
	return a
}

// joined flattens the findings so a case can assert on what they say
// without pinning their order.
func joined(a Audit) string {
	var b strings.Builder
	for _, f := range a.Findings {
		b.WriteString(f.String())
		b.WriteString("\n")
	}
	return b.String()
}

func wantFindings(t *testing.T, a Audit, n int, wants ...string) {
	t.Helper()
	if len(a.Findings) != n {
		t.Fatalf("want %d finding(s), got %d:\n%s", n, len(a.Findings), joined(a))
	}
	for _, w := range wants {
		if !strings.Contains(joined(a), w) {
			t.Errorf("findings do not mention %q:\n%s", w, joined(a))
		}
	}
}

func TestAuditLanes_IsSilentOnACoherentRepository(t *testing.T) {
	repoDir, wtRoot := initialisedLaneRepo(t)
	// A worktree of this repository that is not a lane is not drift: nothing
	// recorded it as one, and a name alone decides nothing (lesson 579).
	runGitLaneTest(t, repoDir, "worktree", "add", "--detach", filepath.Join(wtRoot, "dotfiles", "feat-lane-extra"), "HEAD")

	a := audit(t, repoDir, wtRoot)
	if !a.Managed || !a.Initialised {
		t.Fatalf("want managed and initialised, got %+v", a)
	}
	wantFindings(t, a, 0)
}

// A worktree with no lease is not drift, and the audit warns of both shapes. A
// detached one the guard refuses to let commit while the lanes are active
// (#3436); a branch one it lets through (AC2), but nothing tracks it, so sweep
// can never reap it (#3438). Each warning names its own recovery. Lanes and
// leased worktrees stay out of it.
func TestAuditLanes_WarnsOfLeaselessWorktrees(t *testing.T) {
	repoDir, wtRoot := initialisedLaneRepo(t)
	base := filepath.Join(wtRoot, "dotfiles")
	bare := filepath.Join(base, "feat-scratch")
	leased := filepath.Join(base, "feat-leased")
	onBranch := filepath.Join(base, "feat-branch")
	runGitLaneTest(t, repoDir, "worktree", "add", "--detach", bare, "HEAD")
	runGitLaneTest(t, repoDir, "worktree", "add", "--detach", leased, "HEAD")
	runGitLaneTest(t, repoDir, "worktree", "add", "-b", "feat/on-a-branch", onBranch, "HEAD")
	if err := worktree.SaveMetadata(leased, worktree.Metadata{State: string(worktree.StateActive)}); err != nil {
		t.Fatal(err)
	}

	a := audit(t, repoDir, wtRoot)
	wantFindings(t, a, 0)
	if len(a.Warnings) != 2 {
		t.Fatalf("want two warnings, for %s and %s only, got %d: %+v", bare, onBranch, len(a.Warnings), a.Warnings)
	}
	byPath := map[string]string{}
	for _, w := range a.Warnings {
		byPath[filepath.Base(w.Subject)] = w.Warning()
	}
	for name, wants := range map[string][]string{
		"feat-scratch": {"WARN:", "no lease", "detached", "dotf worktree adopt", "git switch -c"},
		"feat-branch":  {"WARN:", "no lease", "feat/on-a-branch", "dotf worktree adopt"},
	} {
		for _, want := range wants {
			if !strings.Contains(byPath[name], want) {
				t.Errorf("warning for %s does not mention %q:\n%s", name, want, byPath[name])
			}
		}
	}
	if strings.Contains(byPath["feat-branch"], "git switch -c") {
		t.Errorf("a branch worktree is not asked to leave its detached state:\n%s", byPath["feat-branch"])
	}
}

// An expired lease is not drift either: sweep may reap the worktree or park
// the lane under its owner once the lease is past. The audit warns, naming the
// command that ends the work, and leaves alone the leases that cannot expire
// into trouble: a live one, a parked lane's (it carries no expiry) and a
// worktree that has no lease at all.
func TestAuditLanes_WarnsOfAnExpiredLease(t *testing.T) {
	repoDir, wtRoot := initialisedLaneRepo(t)
	base := filepath.Join(wtRoot, "dotfiles")
	stale := filepath.Join(base, "feat-stale")
	live := filepath.Join(base, "feat-live")
	runGitLaneTest(t, repoDir, "worktree", "add", "-b", "feat/stale", stale, "HEAD")
	runGitLaneTest(t, repoDir, "worktree", "add", "-b", "feat/live", live, "HEAD")
	past, future := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	for path, expires := range map[string]time.Time{stale: past, live: future} {
		if err := worktree.SaveMetadata(path, worktree.Metadata{Issue: 1, LeaseExpiresAt: expires, ReapOK: true}); err != nil {
			t.Fatal(err)
		}
	}
	// A lane in use whose lease ran out: the fix is to park it, not to remove it.
	laneName := CanonicalLanes[0]
	lanePath := filepath.Join(base, laneName)
	if err := worktree.SaveMetadata(lanePath, worktree.Metadata{Lane: laneName, State: string(worktree.StateActive), Issue: 2, LeaseExpiresAt: past}); err != nil {
		t.Fatal(err)
	}

	a := audit(t, repoDir, wtRoot)
	wantFindings(t, a, 0)
	var staleWarning, laneWarning Finding
	for _, w := range a.Warnings {
		switch {
		case strings.Contains(w.Subject, "feat-stale"):
			staleWarning = w
		case strings.Contains(w.Subject, laneName):
			laneWarning = w
		default:
			t.Errorf("unexpected warning (live, parked or unleased worktrees stay silent):\n%s", w.Warning())
		}
	}
	// The path git reports can differ in form from the one the test built
	// (slashes, 8.3 names, /var against /private/var): the warning must name
	// this worktree, and its FIX must repeat the path it reports.
	if !samePath(staleWarning.Subject, stale) {
		t.Errorf("warning is about %q, not %q", staleWarning.Subject, stale)
	}
	stalePath, lanePathWarning := staleWarning.Warning(), laneWarning.Warning()
	for _, want := range []string{"WARN:", "lease expired", "FIX: dotf worktree done " + staleWarning.Subject + " "} {
		if !strings.Contains(stalePath, want) {
			t.Errorf("warning for a worktree lacks %q:\n%s", want, stalePath)
		}
	}
	for _, want := range []string{"WARN:", "lease expired", "dotf lane park " + laneName} {
		if !strings.Contains(lanePathWarning, want) {
			t.Errorf("warning for a lane lacks %q:\n%s", want, lanePathWarning)
		}
	}
}

// The guard that runs is the one under the hooks directory git answers with:
// core.hooksPath when set, $GIT_DIR/hooks otherwise. A deploy that lags the
// repository's copy keeps the old rules in force; the warning names the command
// that redeploys. A repository without its own copy has nothing to lag, and a
// hooks directory that is the checkout's own git-hooks is no drift by
// construction.
func TestGuardDrift(t *testing.T) {
	const current, stale = "#!/bin/sh\nexit 0\n", "#!/bin/sh\nexit 1\n"
	writeGuard := func(hooksDir, content string) {
		t.Helper()
		p := filepath.Join(hooksDir, filepath.FromSlash(GuardRelPath))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// where says which directory the deployed copy is written to and which
	// core.hooksPath git is given for it ("" leaves it unset).
	type place struct{ dir, hooksPath string }
	inDeploy := func(repo, deploy string) place { d := filepath.Join(deploy, "git-hooks"); return place{d, d} }
	cases := []struct {
		name     string
		repo     *string
		deployed *string
		where    func(repo, deploy string) place
		want     string
	}{
		{"identical copies", ptr(current), ptr(current), inDeploy, ""},
		{"line endings alone are not drift", ptr(current), ptr("#!/bin/sh\r\nexit 0\r\n"), inDeploy, ""},
		{"deployed copy lags", ptr(current), ptr(stale), inDeploy, "differs"},
		{"guard missing under the hooks path", ptr(current), nil, inDeploy, "not deployed"},
		{"relative hooks path resolves from the repository", ptr(current), ptr(stale), func(repo, deploy string) place {
			d := filepath.Join(deploy, "git-hooks")
			rel, _ := filepath.Rel(repo, d)
			return place{d, rel}
		}, "differs"},
		{"hooks path is the checkout's own git-hooks", ptr(current), nil, func(repo, _ string) place {
			d := filepath.Join(repo, "git-hooks")
			return place{d, d}
		}, ""},
		{"hooks path unset, guard in .git/hooks is current", ptr(current), ptr(current), func(repo, _ string) place {
			return place{filepath.Join(repo, ".git", "hooks"), ""}
		}, ""},
		{"hooks path unset, guard in .git/hooks lags", ptr(current), ptr(stale), func(repo, _ string) place {
			return place{filepath.Join(repo, ".git", "hooks"), ""}
		}, "differs"},
		{"hooks path unset, nothing deployed", ptr(current), nil, func(repo, _ string) place {
			return place{filepath.Join(repo, ".git", "hooks"), ""}
		}, "not deployed"},
		{"repository has no copy of its own", nil, ptr(current), inDeploy, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The machine's own git config must not answer for the test.
			t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			repo, deploy := t.TempDir(), t.TempDir()
			runGitLaneTest(t, repo, "init", "-b", "main")
			if tc.repo != nil {
				writeGuard(filepath.Join(repo, "git-hooks"), *tc.repo)
			}
			w := tc.where(repo, deploy)
			if tc.deployed != nil {
				writeGuard(w.dir, *tc.deployed)
			}
			if w.hooksPath != "" {
				runGitLaneTest(t, repo, "config", "core.hooksPath", w.hooksPath)
			}
			got, err := GuardDrift(repo)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("want no warning, got %+v", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("want one warning, got %+v", got)
			}
			for _, want := range []string{"WARN:", tc.want, "dotf hooks install"} {
				if !strings.Contains(got[0].Warning(), want) {
					t.Errorf("warning lacks %q:\n%s", want, got[0].Warning())
				}
			}
		})
	}
}

// When git cannot say where hooks run from, the answer is an error that names
// the read, not "no guard deployed": a wrong guess would tell the owner to
// redeploy what is already there.
func TestGuardDrift_ReportsAHooksPathGitCannotRead(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	notARepo := t.TempDir()
	p := filepath.Join(notARepo, "git-hooks", filepath.FromSlash(GuardRelPath))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := GuardDrift(notARepo)
	if err == nil || !strings.Contains(err.Error(), "cannot read the hooks path") {
		t.Fatalf("want a hooks path read error, got warnings=%+v err=%v", got, err)
	}
}

func ptr(s string) *string { return &s }

// The caller prints a failure to audit under its own "lane doctor:" prefix, so
// none of the errors AuditLanes returns carries one.
func TestAuditLanes_ErrorsCarryNoCallerPrefix(t *testing.T) {
	t.Run("empty worktree root", func(t *testing.T) {
		repoDir, _ := initialisedLaneRepo(t)
		_, err := AuditLanes(repoDir, " ")
		wantUnprefixed(t, err)
	})
	t.Run("marker unreadable", func(t *testing.T) {
		repoDir, wtRoot := setupLaneTestRepo(t)
		if err := os.MkdirAll(filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile)), 0o755); err != nil {
			t.Fatal(err)
		}
		activateRecord(t, repoDir)
		_, err := AuditLanes(repoDir, wtRoot)
		wantUnprefixed(t, err)
	})
	t.Run("worktree registry unreadable", func(t *testing.T) {
		// A file where the registry directory belongs: ReadDir fails with
		// ENOTDIR here, but Windows reports ERROR_PATH_NOT_FOUND, which Go
		// maps to fs.ErrNotExist, and an absent registry is not an error. No
		// other failure of that read is portable (chmod denies nothing there).
		if runtime.GOOS == "windows" {
			t.Skip("Windows reports a file in place of the registry directory as not-exist, which is not an error")
		}
		repoDir, wtRoot := initialisedLaneRepo(t)
		registry := filepath.Join(repoDir, ".git", "worktrees")
		if err := os.RemoveAll(registry); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(registry, []byte("not a directory"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := AuditLanes(repoDir, wtRoot)
		wantUnprefixed(t, err)
	})
}

func wantUnprefixed(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("want an error")
	}
	if strings.Contains(err.Error(), "lane doctor:") {
		t.Errorf("error carries the caller's prefix: %v", err)
	}
}

func TestAuditLanes_SeparatesNotInitialisedFromDrift(t *testing.T) {
	t.Run("no marker is not lane-managed", func(t *testing.T) {
		repoDir, wtRoot := setupLaneTestRepo(t)
		a := audit(t, repoDir, wtRoot)
		if a.Managed || len(a.Findings) != 0 {
			t.Fatalf("want unmanaged and silent, got %+v", a)
		}
	})
	t.Run("marker with no lane on this machine is not initialised, not drift", func(t *testing.T) {
		repoDir, wtRoot := setupLaneTestRepo(t)
		if _, err := writeLanesMarker(repoDir); err != nil {
			t.Fatal(err)
		}
		activateRecord(t, repoDir)
		a := audit(t, repoDir, wtRoot)
		if !a.Managed || a.Initialised {
			t.Fatalf("want managed and not initialised, got %+v", a)
		}
		wantFindings(t, a, 0)
	})
	t.Run("one lane missing is drift", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		runGitLaneTest(t, repoDir, "worktree", "remove", "--force", filepath.Join(wtRoot, "dotfiles", LaneFeatSecondary))
		a := audit(t, repoDir, wtRoot)
		if !a.Initialised {
			t.Fatalf("want initialised, got %+v", a)
		}
		wantFindings(t, a, 1, LaneFeatSecondary, "absent", "dotf lane init")
	})
}

func TestAuditLanes_ReportsACorruptedMarkerByField(t *testing.T) {
	cases := []struct {
		name, marker string
		wants        []string
	}{
		{"invalid JSON", `{"version": 1,`, []string{LanesMarkerFile, "parse"}},
		{"unknown key", `{"version":1,"lanes":["a"],"third_party_roots":["x"]}`, []string{`"lanes"`}},
		{"wrong version", `{"version":2,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":["x"]}`, []string{"version", "2", "expected 1"}},
		{"empty canonical_lanes", `{"version":1,"canonical_lanes":[],"third_party_roots":["x"]}`, []string{"canonical_lanes", "empty"}},
		{"empty third_party_roots", `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":[]}`, []string{"third_party_roots", "empty"}},
		{"separator in an entry", `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":["a/b"]}`, []string{"third_party_roots", "a/b", "separator"}},
		{"drive prefix in an entry", `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":["C:\\hosts"]}`, []string{"third_party_roots", "C:", "drive"}},
		{"duplicate entry", `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":["x","x"]}`, []string{"third_party_roots", "x", "duplicate"}},
		{"lane set differs from what dotf provisions", `{"version":1,"canonical_lanes":["feat-lane-primary"],"third_party_roots":["x"]}`, []string{"canonical_lanes", "differs", "dotf lane init"}},
		{"dot entry", `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":[".."]}`, []string{"third_party_roots", `".."`, "not a directory name"}},
		{"padded entry", `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":[" x "]}`, []string{"third_party_roots", `" x "`, "whitespace"}},
		{"trailing data after the object", `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":["x"]} {"version":2}`, []string{LanesMarkerFile, "trailing"}},
		{"empty entry", `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":[""]}`, []string{"third_party_roots", "empty entry"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoDir, wtRoot := initialisedLaneRepo(t)
			path := filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))
			if err := os.WriteFile(path, []byte(tc.marker+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			a := audit(t, repoDir, wtRoot)
			if len(a.Findings) == 0 {
				t.Fatal("want at least one finding, got none")
			}
			for _, w := range tc.wants {
				if !strings.Contains(joined(a), w) {
					t.Errorf("findings do not name %q:\n%s", w, joined(a))
				}
			}
			// A broken marker is reported once as the marker's problem, not
			// again as three lanes drifting from a list that is not a list.
			if strings.Contains(joined(a), "absent") {
				t.Errorf("a marker finding must not cascade into lane findings:\n%s", joined(a))
			}
		})
	}
}

func TestAuditLanes_ReportsDriftInBothDirections(t *testing.T) {
	t.Run("a lane moved by hand is registered at a path that no longer exists", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
		renamed := filepath.Join(wtRoot, "dotfiles", "renamed-by-hand")
		if err := os.Rename(lanePath, renamed); err != nil {
			t.Fatal(err)
		}
		a := audit(t, repoDir, wtRoot)
		// Leaf names, not full paths: git records the physical path and the
		// test holds the logical one (macOS /private/var, Windows 8.3 names).
		wantFindings(t, a, 1, LaneFeatPrimary, "no longer exists", "git worktree repair")
		if strings.Contains(joined(a), "dotf lane init") {
			t.Errorf("lane init cannot repair a registered path; git refuses to add there:\n%s", joined(a))
		}
	})
	t.Run("a lane moved with git worktree move is at the wrong path", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFastTrack)
		elsewhere := filepath.Join(wtRoot, "dotfiles", "elsewhere")
		runGitLaneTest(t, repoDir, "worktree", "move", lanePath, elsewhere)
		a := audit(t, repoDir, wtRoot)
		wantFindings(t, a, 1, LaneFastTrack, "elsewhere", "is at ", "git worktree move")
	})
	t.Run("relative gitdir entries (worktree.useRelativePaths) resolve", func(t *testing.T) {
		repoDir, wtRoot := setupLaneTestRepo(t)
		runGitLaneTest(t, repoDir, "config", "worktree.useRelativePaths", "true")
		if _, err := InitLanes(repoDir, wtRoot); err != nil {
			t.Fatal(err)
		}
		if _, err := writeLanesMarker(repoDir); err != nil {
			t.Fatal(err)
		}
		activateRecord(t, repoDir)
		gitDir, _, err := worktreeGitDirs(filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary))
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(gitDir, "gitdir"))
		if err != nil {
			t.Fatal(err)
		}
		if filepath.IsAbs(strings.TrimSpace(string(raw))) {
			t.Skip("this git does not honour worktree.useRelativePaths")
		}
		wantFindings(t, audit(t, repoDir, wtRoot), 0)
	})
	t.Run("an admin dir whose gitdir file is gone is reported, not folded into a missing record", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		gitDir, _, err := worktreeGitDirs(filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(gitDir, "gitdir")); err != nil {
			t.Fatal(err)
		}
		a := audit(t, repoDir, wtRoot)
		wantFindings(t, a, 1, filepath.Base(gitDir), LaneFeatPrimary, "gitdir", "git worktree prune")
		if strings.Contains(joined(a), "no identity record") {
			t.Errorf("the record is intact; only the gitdir file is gone:\n%s", joined(a))
		}
	})
	t.Run("a second registration whose gitdir is gone is not listed as a blank duplicate path", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		twin := filepath.Join(wtRoot, "dotfiles", "twin")
		runGitLaneTest(t, repoDir, "worktree", "add", "--detach", twin, "HEAD")
		twinGitDir, _, err := worktreeGitDirs(twin)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(twinGitDir, LaneIdentityFile), []byte(LaneFeatPrimary+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(twinGitDir, "gitdir")); err != nil {
			t.Fatal(err)
		}
		a := audit(t, repoDir, wtRoot)
		wantFindings(t, a, 1, filepath.Base(twinGitDir), "gitdir")
		if strings.Contains(joined(a), "more than one worktree") {
			t.Errorf("the broken registration is reported on its own; it must not pad the duplicate list:\n%s", joined(a))
		}
	})
	t.Run("an unreadable record is a finding, as the guard fails closed on it", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Getuid() == 0 {
			t.Skip("mode bits do not deny reads here")
		}
		repoDir, wtRoot := initialisedLaneRepo(t)
		gitDir, _, err := worktreeGitDirs(filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary))
		if err != nil {
			t.Fatal(err)
		}
		record := filepath.Join(gitDir, LaneIdentityFile)
		if err := os.Chmod(record, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(record, 0o644) })
		a := audit(t, repoDir, wtRoot)
		if len(a.Findings) == 0 || !strings.Contains(joined(a), "not readable") {
			t.Fatalf("want the unreadable record reported, got:\n%s", joined(a))
		}
	})
	t.Run("an empty worktree root is refused, not resolved to the cwd", func(t *testing.T) {
		repoDir, _ := initialisedLaneRepo(t)
		if _, err := AuditLanes(repoDir, ""); err == nil || !strings.Contains(err.Error(), "WORKTREE_ROOT") {
			t.Fatalf("want an error naming WORKTREE_ROOT, got %v", err)
		}
	})
	t.Run("a worktree recorded as a lane the marker does not declare", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		spike := filepath.Join(wtRoot, "dotfiles", "spike")
		runGitLaneTest(t, repoDir, "worktree", "add", "--detach", spike, "HEAD")
		if _, err := writeIdentity(spike, "spike-lane"); err != nil {
			t.Fatal(err)
		}
		a := audit(t, repoDir, wtRoot)
		wantFindings(t, a, 1, "spike-lane", "spike", "does not declare", LaneIdentityFile)
	})
	t.Run("a lane recorded in more than one worktree", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		twin := filepath.Join(wtRoot, "dotfiles", "twin")
		runGitLaneTest(t, repoDir, "worktree", "add", "--detach", twin, "HEAD")
		if _, err := writeIdentity(twin, LaneFeatPrimary); err != nil {
			t.Fatal(err)
		}
		wantFindings(t, audit(t, repoDir, wtRoot), 1, LaneFeatPrimary, "more than one worktree", "twin", "git worktree remove")
	})
}

func TestAuditLanes_ReportsABrokenLane(t *testing.T) {
	t.Run("no lease", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
		admin, err := worktree.AdminDir(lanePath)
		if err != nil {
			t.Fatal(err)
		}
		for _, lease := range []string{filepath.Join(admin, worktree.LeaseFileName), filepath.Join(lanePath, worktree.MetadataFileName)} {
			if err := os.Remove(lease); err != nil {
				t.Fatal(err)
			}
		}
		wantFindings(t, audit(t, repoDir, wtRoot), 1, LaneFeatPrimary, "no lease", "dotf lane init")
	})
	t.Run("the in-tree mirror gone is not a missing lease", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
		if err := os.Remove(filepath.Join(lanePath, worktree.MetadataFileName)); err != nil {
			t.Fatal(err)
		}
		if a := audit(t, repoDir, wtRoot); len(a.Findings) != 0 {
			t.Fatalf("want no finding with the admin-dir lease in place, got:\n%s", joined(a))
		}
	})
	t.Run("no identity record", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
		gitDir, _, err := worktreeGitDirs(lanePath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(gitDir, LaneIdentityFile)); err != nil {
			t.Fatal(err)
		}
		wantFindings(t, audit(t, repoDir, wtRoot), 1, LaneFeatPrimary, "no identity record", "dotf lane init")
	})
	t.Run("lease names another lane", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
		meta, err := worktree.LoadMetadata(lanePath)
		if err != nil || meta == nil {
			t.Fatalf("lease: %v", err)
		}
		meta.Lane = LaneFastTrack
		if err := worktree.SaveMetadata(lanePath, *meta); err != nil {
			t.Fatal(err)
		}
		wantFindings(t, audit(t, repoDir, wtRoot), 1, LaneFeatPrimary, "lease", LaneFastTrack, "dotf lane init")
	})
	t.Run("lease in a state the lanes do not use", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
		meta, err := worktree.LoadMetadata(lanePath)
		if err != nil || meta == nil {
			t.Fatalf("lease: %v", err)
		}
		meta.State = "REAPED"
		if err := worktree.SaveMetadata(lanePath, *meta); err != nil {
			t.Fatal(err)
		}
		wantFindings(t, audit(t, repoDir, wtRoot), 1, LaneFeatPrimary, `"REAPED"`, "neither ACTIVE nor PARKED", "dotf lane park")
	})
	t.Run("a plain directory at a lane path", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatSecondary)
		runGitLaneTest(t, repoDir, "worktree", "remove", "--force", lanePath)
		if err := os.MkdirAll(lanePath, 0o755); err != nil {
			t.Fatal(err)
		}
		wantFindings(t, audit(t, repoDir, wtRoot), 1, LaneFeatSecondary, "not a git worktree", "git worktree prune")
	})
	// Wiped and recreated by hand: git still registers it, the record is
	// intact, and the directory has no .git file. Measured: dotf lane init
	// refuses the directory, and once it is moved aside git worktree add
	// refuses the registered path too, so the FIX needs git worktree prune.
	t.Run("a plain directory at a lane path git still registers", func(t *testing.T) {
		repoDir, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatSecondary)
		if err := os.RemoveAll(lanePath); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(lanePath, 0o755); err != nil {
			t.Fatal(err)
		}
		a := audit(t, repoDir, wtRoot)
		wantFindings(t, a, 1, LaneFeatSecondary, "not a git worktree", "git worktree prune")
		if strings.Contains(joined(a), "lease") {
			t.Errorf("a plain directory is reported as such, not as a missing lease:\n%s", joined(a))
		}
	})
}

// A worktree of another repository at a lane path is not this repository's
// to move, prune or repair. Measured 2026-11-11 on a scratch pair: from the
// repository whose registration is stale, git worktree prune keeps the entry
// while the other's .git file exists, git worktree remove --force refuses
// ("does not point back"), and git worktree repair REWRITES the other
// repository's .git file to point at this one, hijacking its worktree.
func TestAuditLanes_LeavesAnotherRepositorysWorktreeAlone(t *testing.T) {
	wantForeign := func(t *testing.T, a Audit, n int, wants ...string) {
		t.Helper()
		wantFindings(t, a, n, append(wants, "share a basename", "#3318")...)
		for _, forbidden := range []string{"prune", "repair", "move the directory"} {
			if strings.Contains(joined(a), forbidden) {
				t.Errorf("the FIX names %q against a worktree that is not this repository's:\n%s", forbidden, joined(a))
			}
		}
	}
	t.Run("a fresh clone whose lane paths hold another repository's lanes", func(t *testing.T) {
		owner, wtRoot := initialisedLaneRepo(t)
		clone, _ := setupLaneTestRepo(t)
		if _, err := writeLanesMarker(clone); err != nil {
			t.Fatal(err)
		}
		activateRecord(t, clone)
		a := audit(t, clone, wtRoot)
		if !a.Initialised {
			t.Fatal("Initialised = false: the occupied lane paths read as a clone with nothing to audit, so dotf lane init would be the advice and it refuses")
		}
		wantForeign(t, a, 3, LaneFeatPrimary, LaneFeatSecondary, LaneFastTrack, "different directory name")
		// The named repair works: under another directory name the clone
		// provisions its own lanes, and the owner's stay coherent.
		renamed := clone + "-two"
		if err := os.Rename(clone, renamed); err != nil {
			t.Fatal(err)
		}
		if _, err := InitLanes(renamed, wtRoot); err != nil {
			t.Fatalf("InitLanes after the rename: %v", err)
		}
		wantFindings(t, audit(t, renamed, wtRoot), 0)
		wantFindings(t, audit(t, owner, wtRoot), 0)
	})
	t.Run("this repository's stale registration at a path another repository's worktree took", func(t *testing.T) {
		owner, wtRoot := initialisedLaneRepo(t)
		lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
		if err := os.RemoveAll(lanePath); err != nil {
			t.Fatal(err)
		}
		other, _ := setupLaneTestRepo(t)
		runGitLaneTest(t, other, "worktree", "add", "--detach", lanePath, "HEAD")
		a := audit(t, owner, wtRoot)
		wantForeign(t, a, 1, LaneFeatPrimary, "stale registration", filepath.Join("worktrees", LaneFeatPrimary), "by hand")
	})
}

// The audit gates on what the guard gates on: the activation record, never the
// marker. A marker in the working tree, however it reads, is not a repository the
// guard enforces, so the audit has nothing to compare (#3437).
func TestAuditLanes_IsManagedByTheActivationRecordNotTheMarker(t *testing.T) {
	t.Run("a marker with no record is not managed, and a broken one is not read", func(t *testing.T) {
		repoDir, wtRoot := setupLaneTestRepo(t)
		path := filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{"version": 1,`), 0o644); err != nil {
			t.Fatal(err)
		}
		a := audit(t, repoDir, wtRoot)
		if a.Managed || len(a.Findings) != 0 || len(a.Warnings) != 0 {
			t.Fatalf("want unmanaged and silent, got %+v", a)
		}
	})
	t.Run("a record with the marker is managed", func(t *testing.T) {
		repoDir, wtRoot := setupLaneTestRepo(t)
		if _, err := writeLanesMarker(repoDir); err != nil {
			t.Fatal(err)
		}
		activateRecord(t, repoDir)
		if a := audit(t, repoDir, wtRoot); !a.Managed {
			t.Fatalf("want managed, got %+v", a)
		}
	})
	t.Run("a record with no marker in the main checkout is managed, and says it compared nothing", func(t *testing.T) {
		repoDir, wtRoot := setupLaneTestRepo(t)
		activateRecord(t, repoDir)
		a := audit(t, repoDir, wtRoot)
		if !a.Managed || len(a.Findings) != 0 {
			t.Fatalf("want managed with no finding, got %+v", a)
		}
		if len(a.Warnings) != 1 || !strings.Contains(a.Warnings[0].Warning(), LanesMarkerFile) {
			t.Fatalf("want one warning naming %s, got %+v", LanesMarkerFile, a.Warnings)
		}
	})
}

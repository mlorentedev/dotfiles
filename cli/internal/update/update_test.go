package update

import (
	"errors"
	"strings"
	"testing"
)

// fakeGit maps a joined arg string to a canned stdout, or to an error when the
// key is in fail. Any key absent from both returns "" with no error.
type fakeGit struct {
	out  map[string]string
	fail map[string]bool
}

func (f fakeGit) run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	if f.fail[key] {
		return "", errors.New("git failed: " + key)
	}
	return f.out[key], nil
}

// healthyGit is a repo that is a clean, fetchable, fast-forwardable checkout:
// HEAD (aaaa) != @{u} (bbbb) and the merge-base equals HEAD, so a fast-forward
// is valid. Individual tests mutate one entry to exercise a single branch.
func healthyGit() fakeGit {
	return fakeGit{
		out: map[string]string{
			"rev-parse --git-dir": ".git",
			"status --porcelain":  "",
			"fetch --quiet":       "",
			"rev-parse --abbrev-ref --symbolic-full-name @{u}": "origin/main",
			"rev-parse HEAD":       "aaaa",
			"rev-parse @{u}":       "bbbb",
			"merge-base HEAD @{u}": "aaaa",
			"merge --ff-only @{u}": "",
		},
		fail: map[string]bool{},
	}
}

func TestUpdate(t *testing.T) {
	cfg := Config{Repo: "/repo"}

	tests := []struct {
		name          string
		mutate        func(g fakeGit) // tweak the healthy fake for this branch
		setupErr      error           // Converge result
		wantStatus    string
		wantMsgSubstr string // if set, out.Message must contain it
		wantMsgSuffix string // if set, out.Message must end with it
		wantErr       bool
		setupRuns     bool // whether Converge must be invoked
	}{
		{name: "not a repo", mutate: func(g fakeGit) { g.fail["rev-parse --git-dir"] = true }, wantStatus: "not-a-repo", wantMsgSubstr: "nothing to self-update"},
		// An unreadable status must fail safe (skip, not proceed as if clean) —
		// distinct from the dirty-worktree branch below, which reads fine but
		// reports changes.
		{name: "cannot read git status", mutate: func(g fakeGit) { g.fail["status --porcelain"] = true }, wantStatus: "dirty", wantMsgSubstr: "cannot read git status"},
		// The dirty-skip message must name the offending path so a silently
		// skipping scheduled run stays diagnosable (dotfiles#694).
		// The path list ends the message: a suffix appended after it would land
		// on the last path and garble it.
		{name: "dirty worktree", mutate: func(g fakeGit) { g.out["status --porcelain"] = " M setup.sh" }, wantStatus: "dirty", wantMsgSubstr: "Dirtying paths:\n M setup.sh", wantMsgSuffix: "setup.sh"},
		{name: "fetch offline", mutate: func(g fakeGit) { g.fail["fetch --quiet"] = true }, wantStatus: "offline"},
		{name: "no upstream", mutate: func(g fakeGit) { g.fail["rev-parse --abbrev-ref --symbolic-full-name @{u}"] = true }, wantStatus: "no-upstream"},
		{name: "already current", mutate: func(g fakeGit) { g.out["rev-parse @{u}"] = "aaaa" }, wantStatus: "current"},
		{name: "diverged", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "cccc" }, wantStatus: "diverged"},
		// Ahead is safe for Sync's other caller, not for a deploy: update only
		// ever runs what the upstream holds, so it skips, but under its own tag:
		// nothing has diverged and nothing needs reconciling (#2205).
		{name: "ahead skips without claiming a divergence", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "bbbb" }, wantStatus: StatusAhead, wantMsgSubstr: "ahead of origin/main", wantMsgSuffix: "nothing to deploy"},
		{name: "ff failed", mutate: func(g fakeGit) { g.fail["merge --ff-only @{u}"] = true }, wantStatus: "ff-failed"},
		{name: "clean ff converges", mutate: func(g fakeGit) {}, wantStatus: "updated", setupRuns: true},
		{name: "converge failure is the only error", mutate: func(g fakeGit) {}, setupErr: errors.New("boom"), wantStatus: "converge-failed", wantErr: true, setupRuns: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			tt.mutate(g)
			ranSetup := false
			d := Deps{
				Git: g.run,
				Converge: func() error {
					ranSetup = true
					return tt.setupErr
				},
			}

			out, err := Run(cfg, d)

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if out.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q (msg: %s)", out.Status, tt.wantStatus, out.Message)
			}
			if tt.wantMsgSubstr != "" && !strings.Contains(out.Message, tt.wantMsgSubstr) {
				t.Errorf("message = %q, want it to contain %q", out.Message, tt.wantMsgSubstr)
			}
			if tt.wantMsgSuffix != "" && !strings.HasSuffix(out.Message, tt.wantMsgSuffix) {
				t.Errorf("message = %q, want it to end with %q", out.Message, tt.wantMsgSuffix)
			}
			if ranSetup != tt.setupRuns {
				t.Errorf("Converge invoked = %v, want %v — a skip branch must never converge", ranSetup, tt.setupRuns)
			}
		})
	}
}

// TestSync covers the statuses `dotf harness refresh` branches on that Run
// folds away: ahead is its own status, and a fast-forward is reported, not run
// into a setup.
func TestSync(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(g fakeGit)
		wantStatus string
		wantFF     bool
	}{
		{name: "behind fast-forwards", mutate: func(g fakeGit) {}, wantStatus: StatusFastForwarded, wantFF: true},
		{name: "level", mutate: func(g fakeGit) { g.out["rev-parse @{u}"] = "aaaa" }, wantStatus: StatusCurrent},
		{name: "ahead", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "bbbb" }, wantStatus: StatusAhead},
		{name: "diverged", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "cccc" }, wantStatus: "diverged"},
		{name: "dirty", mutate: func(g fakeGit) { g.out["status --porcelain"] = "?? new.md" }, wantStatus: "dirty"},
		{name: "offline", mutate: func(g fakeGit) { g.fail["fetch --quiet"] = true }, wantStatus: "offline"},
		{name: "no merge-base", mutate: func(g fakeGit) { g.fail["merge-base HEAD @{u}"] = true }, wantStatus: "no-upstream"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			tt.mutate(g)
			merged := false
			git := func(args ...string) (string, error) {
				if strings.Join(args, " ") == "merge --ff-only @{u}" {
					merged = true
				}
				return g.run(args...)
			}

			out := Sync("/vault", git)

			if out.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q (msg: %s)", out.Status, tt.wantStatus, out.Message)
			}
			if merged != tt.wantFF {
				t.Errorf("fast-forward attempted = %v, want %v", merged, tt.wantFF)
			}
		})
	}
}

// Assess is Sync without the merge: a plan calls it, so it must report where
// Sync would fast-forward and never move HEAD.
func TestAssess_ReportsBehindAndNeverMerges(t *testing.T) {
	var calls []string
	g := healthyGit()
	run := func(args ...string) (string, error) {
		calls = append(calls, strings.Join(args, " "))
		return g.run(args...)
	}

	out := Assess("/repo", run)

	if out.Status != StatusBehind || out.Upstream != "origin/main" {
		t.Fatalf("Assess = %+v, want behind origin/main", out)
	}
	for _, c := range calls {
		if strings.HasPrefix(c, "merge ") {
			t.Fatalf("Assess must not merge, calls: %v", calls)
		}
	}
	if got := Sync("/repo", g.run).Status; got != StatusFastForwarded {
		t.Errorf("Sync on the same checkout = %q, want %q", got, StatusFastForwarded)
	}
}

// TestRunUpgradesSystemPackagesOnAQuietDay covers TOOL-036's daily tick: a day
// the checkout did not move still runs the system-upgrade step, once. Every
// other path either converges in full (which includes the step) or skips, so
// the step must not run a second time there.
func TestRunUpgradesSystemPackagesOnAQuietDay(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(g fakeGit)
		running     string // the running dotf; behind the pin installs it
		upgradeErr  error
		wantStatus  string
		wantErr     bool
		wantUpgrade bool
	}{
		{name: "current upgrades", mutate: func(g fakeGit) { g.out["rev-parse @{u}"] = "aaaa" }, wantStatus: StatusCurrent, wantUpgrade: true},
		{name: "current, upgrade fails", mutate: func(g fakeGit) { g.out["rev-parse @{u}"] = "aaaa" }, upgradeErr: errors.New("winget: 1 id failed"), wantStatus: StatusUpgradeFailed, wantErr: true, wantUpgrade: true},
		{name: "fast-forward converges in full instead", mutate: func(fakeGit) {}, wantStatus: "updated"},
		{name: "current, behind the pin: the installer converges", mutate: func(g fakeGit) { g.out["rev-parse @{u}"] = "aaaa" }, running: "0.69.0", wantStatus: "reinstalled"},
		{name: "dirty skips", mutate: func(g fakeGit) { g.out["status --porcelain"] = " M x" }, wantStatus: "dirty"},
		{name: "offline skips", mutate: func(g fakeGit) { g.fail["fetch --quiet"] = true }, wantStatus: "offline"},
		{name: "ahead skips", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "bbbb" }, wantStatus: StatusAhead},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			tt.mutate(g)
			upgrades := 0
			out, err := Run(Config{Repo: "/repo", Version: tt.running}, Deps{
				Git:      g.run,
				Converge: func() error { return nil },
				Pin:      func() (string, error) { return "0.70.0", nil },
				Install:  func(string) error { return nil },
				Upgrade:  func() error { upgrades++; return tt.upgradeErr },
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if out.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q (msg: %s)", out.Status, tt.wantStatus, out.Message)
			}
			if want := map[bool]int{true: 1, false: 0}[tt.wantUpgrade]; upgrades != want {
				t.Errorf("Upgrade calls = %d, want %d", upgrades, want)
			}
			if tt.wantUpgrade && !strings.Contains(out.Message, "system-upgrade") {
				t.Errorf("message = %q, want it to name the system-upgrade step", out.Message)
			}
		})
	}
}

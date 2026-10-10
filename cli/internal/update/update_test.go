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
		setupErr      error           // RunSetup result
		wantStatus    string
		wantMsgSubstr string // if set, out.Message must contain it
		wantMsgSuffix string // if set, out.Message must end with it
		wantErr       bool
		setupRuns     bool // whether RunSetup must be invoked
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
		{name: "dirty worktree", mutate: func(g fakeGit) { g.out["status --porcelain"] = " M setup.sh" }, wantStatus: "dirty", wantMsgSubstr: "Dirtying paths:\nM setup.sh", wantMsgSuffix: "setup.sh"},
		{name: "fetch offline", mutate: func(g fakeGit) { g.fail["fetch --quiet"] = true }, wantStatus: "offline"},
		{name: "no upstream", mutate: func(g fakeGit) { g.fail["rev-parse --abbrev-ref --symbolic-full-name @{u}"] = true }, wantStatus: "no-upstream"},
		{name: "already current", mutate: func(g fakeGit) { g.out["rev-parse @{u}"] = "aaaa" }, wantStatus: "current"},
		{name: "diverged", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "cccc" }, wantStatus: "diverged"},
		// Ahead is safe for Sync's other caller, not for a deploy: update only
		// ever runs what the upstream holds, so it stays a "diverged" skip.
		{name: "ahead is a diverged skip", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "bbbb" }, wantStatus: "diverged", wantMsgSubstr: "diverged from origin/main (non fast-forward)"},
		{name: "ff failed", mutate: func(g fakeGit) { g.fail["merge --ff-only @{u}"] = true }, wantStatus: "ff-failed"},
		{name: "clean ff runs setup", mutate: func(g fakeGit) {}, wantStatus: "updated", setupRuns: true},
		{name: "setup failure is the only error", mutate: func(g fakeGit) {}, setupErr: errors.New("boom"), wantStatus: "setup-failed", wantErr: true, setupRuns: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			tt.mutate(g)
			ranSetup := false
			d := Deps{
				Git: g.run,
				RunSetup: func() error {
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
				t.Errorf("RunSetup invoked = %v, want %v — a skip branch must never re-run setup", ranSetup, tt.setupRuns)
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

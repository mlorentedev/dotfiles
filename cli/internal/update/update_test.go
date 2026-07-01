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
		name       string
		mutate     func(g fakeGit) // tweak the healthy fake for this branch
		setupErr   error           // RunSetup result
		wantStatus string
		wantErr    bool
		setupRuns  bool // whether RunSetup must be invoked
	}{
		{name: "not a repo", mutate: func(g fakeGit) { g.fail["rev-parse --git-dir"] = true }, wantStatus: "not-a-repo"},
		{name: "dirty worktree", mutate: func(g fakeGit) { g.out["status --porcelain"] = " M setup.sh" }, wantStatus: "dirty"},
		{name: "fetch offline", mutate: func(g fakeGit) { g.fail["fetch --quiet"] = true }, wantStatus: "offline"},
		{name: "no upstream", mutate: func(g fakeGit) { g.fail["rev-parse --abbrev-ref --symbolic-full-name @{u}"] = true }, wantStatus: "no-upstream"},
		{name: "already current", mutate: func(g fakeGit) { g.out["rev-parse @{u}"] = "aaaa" }, wantStatus: "current"},
		{name: "diverged", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "cccc" }, wantStatus: "diverged"},
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
			if ranSetup != tt.setupRuns {
				t.Errorf("RunSetup invoked = %v, want %v — a skip branch must never re-run setup", ranSetup, tt.setupRuns)
			}
		})
	}
}

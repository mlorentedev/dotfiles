package converge

import (
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

// lfsFake is a global git config with no lfs filter and a git-lfs that is
// installed when lfs is true; `git lfs install` writes the four keys.
func lfsFake(lfs bool, cfg map[string][]string) (*int, func(string, ...string) ([]byte, error)) {
	installs := 0
	return &installs, func(name string, args ...string) ([]byte, error) {
		cmd := name + " " + strings.Join(args, " ")
		switch {
		case cmd == "git lfs version" && lfs:
			return []byte("git-lfs/3.8.0\n"), nil
		case cmd == "git lfs version":
			return nil, gitExit(1)
		case cmd == "git lfs install --skip-repo":
			installs++
			for _, k := range gitconfig.LFSFilter {
				cfg[k.Key] = []string{k.Value}
			}
			return nil, nil
		case strings.HasPrefix(cmd, "git config --global --get-all "):
			if v, ok := cfg[args[3]]; ok {
				return []byte(strings.Join(v, "\n") + "\n"), nil
			}
			return nil, gitExit(1)
		case strings.HasPrefix(cmd, "git config --system --get-all "):
			return nil, gitExit(1)
		}
		return nil, gitExit(2)
	}
}

// #2201: the step writes the filter once, and the second run changes nothing.
func TestGitLFS_InstallsTheFilterThenIsConverged(t *testing.T) {
	installs, run := lfsFake(true, map[string][]string{})
	r := gitLFS{run: run, has: func(string) bool { return true }}
	plan, err := r.Reconcile(Env{}, true)
	if err != nil || plan.Changes != 1 || *installs != 0 {
		t.Fatalf("plan = %+v, %v, installs %d", plan, err, *installs)
	}
	if res, err := r.Reconcile(Env{}, false); err != nil || res.Changes != 1 || *installs != 1 {
		t.Fatalf("apply = %+v, %v, installs %d", res, err, *installs)
	}
	if err := r.Probe(Env{}); err != nil {
		t.Fatal(err)
	}
	if res, err := r.Reconcile(Env{}, false); err != nil || res.Changes != 0 || *installs != 1 {
		t.Errorf("second run = %+v, %v, installs %d: want changed=0", res, err, *installs)
	}
}

// No git-lfs is a skip naming the catalog entry, not a failure: the tools step
// installs it, and a machine where it cannot is not broken.
func TestGitLFS_SkipsWithoutGitLFS(t *testing.T) {
	installs, run := lfsFake(false, map[string][]string{})
	r := gitLFS{run: run, has: func(string) bool { return true }}
	res, err := r.Reconcile(Env{}, false)
	if err != nil || !strings.Contains(res.Skip, "dotf tools install git-lfs") || *installs != 0 {
		t.Fatalf("res = %+v, %v, installs %d", res, err, *installs)
	}
	if err := r.Probe(Env{}); err != nil {
		t.Errorf("probe on a skip: %v", err)
	}
}

// A filter key someone set to another value is left as it is and said:
// `git lfs install` would refuse it every night.
func TestGitLFS_LeavesADifferentFilter(t *testing.T) {
	cfg := map[string][]string{}
	for _, k := range gitconfig.LFSFilter {
		cfg[k.Key] = []string{k.Value}
	}
	cfg["filter.lfs.required"] = []string{"false"}
	installs, run := lfsFake(true, cfg)
	r := gitLFS{run: run, has: func(string) bool { return true }}
	res, err := r.Reconcile(Env{}, false)
	if err != nil || res.Changes != 0 || *installs != 0 || !strings.Contains(res.Detail, "left as it is") {
		t.Fatalf("res = %+v, %v, installs %d", res, err, *installs)
	}
}

// An absent key beside a differing one installs nothing: git lfs install
// exits 2 on the differing key, so running it would fail every converge.
func TestGitLFS_AnAbsentKeyBesideADifferentOneInstallsNothing(t *testing.T) {
	cfg := map[string][]string{"filter.lfs.smudge": {"custom-smudge %f"}}
	installs, run := lfsFake(true, cfg)
	r := gitLFS{run: run, has: func(string) bool { return true }}
	res, err := r.Reconcile(Env{}, false)
	if err != nil || res.Changes != 0 || *installs != 0 || !strings.Contains(res.Detail, "while a key differs") {
		t.Fatalf("res = %+v, %v, installs %d", res, err, *installs)
	}
	if err := r.Probe(Env{}); err != nil {
		t.Errorf("probe: %v", err)
	}
}

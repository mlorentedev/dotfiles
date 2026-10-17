package gitconfig

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// fakeLFS answers `git config --global|--system --get-all` from two maps and
// records `git lfs install`, which writes the four keys globally, as the real
// one does.
type fakeLFS struct {
	global, system map[string][]string
	installed      bool
	installs       [][]string
	readErr        error
}

func (f *fakeLFS) run(name string, args ...string) ([]byte, error) {
	cmd := name + " " + strings.Join(args, " ")
	switch {
	case cmd == "git lfs version":
		if !f.installed {
			return nil, exitErr(1)
		}
		return []byte("git-lfs/3.8.0\n"), nil
	case strings.HasPrefix(cmd, "git lfs install"):
		f.installs = append(f.installs, args)
		for _, k := range LFSFilter {
			f.global[k.Key] = []string{k.Value}
		}
		return nil, nil
	case strings.HasPrefix(cmd, "git config --global --get-all "), strings.HasPrefix(cmd, "git config --system --get-all "):
		if f.readErr != nil {
			return nil, f.readErr
		}
		cfg := f.global
		if args[1] == "--system" {
			cfg = f.system
		}
		vals, ok := cfg[args[3]]
		if !ok {
			return nil, exitErr(1)
		}
		return []byte(strings.Join(vals, "\n") + "\n"), nil
	}
	return nil, errors.New("unexpected command: " + cmd)
}

func lfsKeys() map[string][]string {
	m := map[string][]string{}
	for _, k := range LFSFilter {
		m[k.Key] = []string{k.Value}
	}
	return m
}

// The four keys are what `git lfs install` writes, measured with git-lfs 3.8.0
// in a scratch HOME.
func TestLFSFilter_IsWhatGitLFSInstallWrites(t *testing.T) {
	want := []string{
		"filter.lfs.clean=git-lfs clean -- %f",
		"filter.lfs.smudge=git-lfs smudge -- %f",
		"filter.lfs.process=git-lfs filter-process",
		"filter.lfs.required=true",
	}
	var got []string
	for _, k := range LFSFilter {
		got = append(got, k.Key+"="+k.Value)
	}
	if !slices.Equal(got, want) {
		t.Errorf("LFSFilter = %v, want %v", got, want)
	}
}

func TestInspectLFS_ByState(t *testing.T) {
	for _, tt := range []struct {
		name           string
		f              *fakeLFS
		notInstalled   bool
		absent, differ int
	}{
		{name: "git-lfs not installed", f: &fakeLFS{global: map[string][]string{}, system: map[string][]string{}}, notInstalled: true},
		{name: "installed, no filter (the Mac, 2026-10-17)", f: &fakeLFS{installed: true, global: map[string][]string{}, system: map[string][]string{}}, absent: 4},
		{name: "global filter in place", f: &fakeLFS{installed: true, global: lfsKeys(), system: map[string][]string{}}},
		{name: "system filter only (Git for Windows)", f: &fakeLFS{installed: true, global: map[string][]string{}, system: lfsKeys()}},
		{name: "one key set to another value", f: &fakeLFS{installed: true, system: map[string][]string{}, global: func() map[string][]string {
			m := lfsKeys()
			m["filter.lfs.smudge"] = []string{"git-lfs smudge --skip -- %f"}
			return m
		}()}, differ: 1},
		{name: "global overrides a differing system value", f: &fakeLFS{installed: true, global: lfsKeys(), system: map[string][]string{"filter.lfs.required": {"false"}}}},
		// install exits 0 over it and the global key wins (measured, pr-agent on #2697).
		{name: "a differing system value with no global one is absent", f: &fakeLFS{installed: true, global: map[string][]string{}, system: map[string][]string{"filter.lfs.smudge": {"admin-smudge %f"}}}, absent: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			st, err := InspectLFS(tt.f.run)
			if err != nil {
				t.Fatal(err)
			}
			if st.NotInstalled != tt.notInstalled || len(st.Absent) != tt.absent || len(st.Differ) != tt.differ {
				t.Errorf("state %+v, want notInstalled=%v absent=%d differ=%d", st, tt.notInstalled, tt.absent, tt.differ)
			}
		})
	}
}

// A config git cannot read is an error, never "no filter".
func TestInspectLFS_AnUnreadableConfigIsAnError(t *testing.T) {
	f := &fakeLFS{installed: true, global: map[string][]string{}, system: map[string][]string{}, readErr: exitErr(3)}
	if _, err := InspectLFS(f.run); err == nil {
		t.Error("an exit 3 read as an absent key")
	}
}

// The install never touches the repository it runs in: without --skip-repo,
// `git lfs install` inside a checkout also writes that repo's hooks, which the
// git-hooks step owns.
func TestInstallLFS_SkipsTheRepository(t *testing.T) {
	f := &fakeLFS{installed: true, global: map[string][]string{}, system: map[string][]string{}}
	if err := InstallLFS(f.run); err != nil {
		t.Fatal(err)
	}
	if len(f.installs) != 1 || !slices.Equal(f.installs[0], []string{"lfs", "install", "--skip-repo"}) {
		t.Fatalf("installs = %v, want one `git lfs install --skip-repo`", f.installs)
	}
	if st, _ := InspectLFS(f.run); !st.Converged() {
		t.Errorf("after the install: %+v", st)
	}
}

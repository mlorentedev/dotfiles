package gitconfig

import "fmt"

// LFSKey is one setting of git's lfs filter.
type LFSKey struct{ Key, Value string }

// LFSFilter is what `git lfs install` writes to the global config: without
// it, a clone of a repository that tracks files with LFS checks out the
// pointers instead of the files (#2201).
var LFSFilter = []LFSKey{
	{"filter.lfs.clean", "git-lfs clean -- %f"},
	{"filter.lfs.smudge", "git-lfs smudge -- %f"},
	{"filter.lfs.process", "git-lfs filter-process"},
	{"filter.lfs.required", "true"},
}

// LFSState is how far the machine's git is from cloning LFS content.
type LFSState struct {
	// NotInstalled: `git lfs version` fails, so git finds no git-lfs; the
	// catalog's git-lfs entry installs it.
	NotInstalled bool
	// Absent are the filter keys set in neither the global nor the system
	// config: `git lfs install` writes them.
	Absent []string
	// Differ are the keys the global config sets to another value. They are
	// left as they are: `git lfs install` refuses to overwrite them without
	// --force, and a different filter is somebody's choice.
	Differ []string
}

// Converged reports whether git clones LFS content with the stock filter.
func (s LFSState) Converged() bool {
	return !s.NotInstalled && len(s.Absent) == 0 && len(s.Differ) == 0
}

// InspectLFS reads the lfs filter from the global config, else the system
// one, which is where Git for Windows sets it. The repository's own config is
// never read: the step is about the user, not the directory it runs in.
func InspectLFS(run Runner) (LFSState, error) {
	var st LFSState
	if _, err := run("git", "lfs", "version"); err != nil {
		st.NotInstalled = true
		return st, nil
	}
	for _, k := range LFSFilter {
		global, err := lastIn(run, "--global", k.Key)
		if err != nil {
			return st, err
		}
		switch global {
		case k.Value:
			continue
		case "":
		default:
			st.Differ = append(st.Differ, fmt.Sprintf("%s is %q, not %q", k.Key, global, k.Value))
			continue
		}
		system, err := lastIn(run, "--system", k.Key)
		if err != nil {
			return st, err
		}
		// A system value other than the stock one is absent, not different:
		// measured with git-lfs 3.8.0, install exits 0 over it and writes the
		// global key, which git reads after the system one (pr-agent on #2697).
		if system != k.Value {
			st.Absent = append(st.Absent, k.Key)
		}
	}
	return st, nil
}

// lastIn is key's last value in the given scope's config, "" when unset.
func lastIn(run Runner, scope, key string) (string, error) {
	vals, err := getAllIn(run, scope, key)
	if err != nil || len(vals) == 0 {
		return "", err
	}
	return vals[len(vals)-1], nil
}

// InstallLFS writes the filter through git-lfs itself. --skip-repo keeps it out
// of the repository it runs in: there it would also write the LFS hooks, which
// the git-hooks step owns.
func InstallLFS(run Runner) error {
	if _, err := run("git", "lfs", "install", "--skip-repo"); err != nil {
		return fmt.Errorf("git lfs install --skip-repo: %w", err)
	}
	return nil
}

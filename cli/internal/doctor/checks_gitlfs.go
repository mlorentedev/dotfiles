package doctor

import (
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

// checkGitLFS reports whether git's lfs filter is in the global config
// (#2201), through the same InspectLFS the converge step applies. Without it a
// clone of a repository that tracks files with LFS checks out the pointers.
// No git-lfs is a WARN, not a SKIP: the catalog installs it on every OS.
func checkGitLFS(sys *System, rep *Report) {
	rep.Section("Git LFS")
	if !sys.has("git") {
		rep.Skip("git not on PATH")
		return
	}
	home := sys.home()
	st, err := gitconfig.InspectLFS(func(name string, args ...string) ([]byte, error) {
		out, err := sys.CommandStdoutDir(home, name, args...)
		return []byte(out), err
	})
	switch {
	case err != nil:
		rep.Warn("cannot read the lfs filter from the global git config: " + err.Error())
	case st.NotInstalled:
		rep.Warn("git-lfs is not installed, so a clone of an LFS repository checks out pointers (run: dotf tools install git-lfs, then dotf converge --only git-lfs)")
	case len(st.Differ) > 0:
		// Before the absent keys: converge installs nothing while one differs.
		rep.Warn("the lfs filter differs from what git lfs install writes, left as it is: " + strings.Join(st.Differ, "; ") +
			" (git lfs install --force resets it, if the value is not yours)")
	case len(st.Absent) > 0:
		rep.Warn("the global git config has no lfs filter (" + strings.Join(st.Absent, ", ") + ") (run: dotf converge --only git-lfs)")
	default:
		rep.Pass("git-lfs filter in the global git config")
	}
}

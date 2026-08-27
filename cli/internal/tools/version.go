package tools

import "os/exec"

// Runner executes name with args and returns its combined output. The seam
// ProbeVersion takes, so the extraction is unit-tested against captured banners
// without the tools installed.
type Runner func(name string, args ...string) ([]byte, error)

// ExecRunner is the production Runner: `<name> <args>` on PATH, stdout and
// stderr merged, because tools disagree about which stream a version goes to.
func ExecRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput() //nolint:gosec // name is an operator-chosen tool
}

// ProbeVersion runs `<name> --version` through run and returns the first
// semver anywhere in its output, or "" when the tool is absent or prints none.
//
// One extraction for every caller. The setup scripts each parsed the version
// as "last whitespace token of the first line" — seven sites across both OSes —
// and on the Windows work box that accepted `locked.` as opencode's version
// from a banner line printed before the number (AI-034, #1294). Output is
// still used when the command exits non-zero: several tools print the version
// and then complain about something unrelated.
func ProbeVersion(name string, run Runner) string {
	if run == nil {
		run = ExecRunner
	}
	out, err := run(name, "--version")
	if err != nil && len(out) == 0 {
		return ""
	}
	if m := semverRE.Find(out); m != nil {
		return string(m)
	}
	return ""
}

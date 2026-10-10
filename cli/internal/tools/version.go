package tools

import (
	"bytes"
	"os/exec"
)

// Runner executes name with args and returns its output. The seam
// ProbeVersion takes, so the extraction is unit-tested against captured banners
// without the tools installed.
type Runner func(name string, args ...string) ([]byte, error)

// ExecRunner is the production Runner: `<name> <args>` on PATH, stdout then
// stderr. Both, because tools disagree about which stream a version goes to;
// stdout first, because a runtime's warning on stderr can carry a version of
// its own and print before the tool's banner (poetry under a Python whose
// urllib3 warns about "OpenSSL 1.1.1+").
func ExecRunner(name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...) //nolint:gosec // name is an operator-chosen tool
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return append(stdout.Bytes(), stderr.Bytes()...), err
}

// ProbeVersion runs `<name> --version` through run and returns the first
// semver in its output (stdout before stderr, see ExecRunner), or "" when the
// tool is absent or prints none.
//
// One extraction for every caller. The setup scripts each parsed the version
// as "last whitespace token of the first line" — seven sites across both OSes —
// and on the Windows work box that accepted `locked.` as opencode's version
// from a banner line printed before the number (AI-034, #1294). Output is
// still used when the command exits non-zero: several tools print the version
// and then complain about something unrelated.
func ProbeVersion(name string, run Runner) string {
	return ProbeVersionArgs(name, run, "--version")
}

// ProbeVersionArgs is ProbeVersion for a tool that reports its version some
// other way than `--version` (kubectl's `version --client`, kubeconform's
// `-v`): the same extraction over `<name> <args>`.
func ProbeVersionArgs(name string, run Runner, args ...string) string {
	if run == nil {
		run = ExecRunner
	}
	out, err := run(name, args...)
	if err != nil && len(out) == 0 {
		return ""
	}
	if m := semverRE.Find(out); m != nil {
		return string(m)
	}
	return ""
}

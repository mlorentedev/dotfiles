package tools

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// ProbeRunner is the Runner version probes use: ExecRunner's stdout then
// stderr, except that a failed run gives its stdout alone. A tool that crashes
// leaves on stderr the warnings and tracebacks of whatever it loaded, and their
// versions are not its own: ansible-lint under a Python its ansible-core does
// not support exited 3 with only `ansible-core 2.21.5 supports Python up to
// 3.14` to read, and was reported as installed at 2.21.5 (BUG-141). Read from
// stdout alone, it does not run, which is what it did.
func ProbeRunner(name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...) //nolint:gosec // name is an operator-chosen tool
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), err
	}
	return append(stdout.Bytes(), stderr.Bytes()...), nil
}

// ProbeVersion runs `<name> --version` through run and returns the first
// semver in its output (stdout before stderr, see ProbeRunner), or "" when the
// tool is absent or prints none.
//
// One extraction for every caller. The setup scripts each parsed the version
// as "last whitespace token of the first line" — seven sites across both OSes —
// and on the Windows work box that accepted `locked.` as opencode's version
// from a banner line printed before the number (AI-034, #1294). stdout is
// still used when the command exits non-zero: several tools print the version
// and then complain about something unrelated.
func ProbeVersion(name string, run Runner) string {
	return ProbeVersionArgs(name, run, "--version")
}

// ProbeTool is the version t answers with on goos, read where its installer
// puts it: the launcher on an OS where t installs itself, `<name>` on PATH
// otherwise. A second copy on PATH, such as a uv one beside hive's own layout
// on Windows, never answers for a self-installed tool (#2991). converge,
// `dotf tools version` and the doctor read it, so they agree on such a machine.
func ProbeTool(t Tool, goos string, getenv func(string) string, run Runner) string {
	if !t.Source.SelfInstalls(goos) {
		return ProbeVersion(t.Name, run)
	}
	launcher := t.Source.SelfInstall.LauncherPath(getenv)
	if launcher == "" {
		return ""
	}
	return ProbeVersion(launcher, run)
}

// ProbeVersionArgs is ProbeVersion for a tool that reports its version some
// other way than `--version` (kubectl's `version --client`, kubeconform's
// `-v`): the same extraction over `<name> <args>`.
func ProbeVersionArgs(name string, run Runner, args ...string) string {
	if run == nil {
		run = ProbeRunner
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

// copiesOnPath lists every file in pathList that would run as name, in PATH
// order, the way exec.LookPath resolves a command: on Windows name plus each
// PATHEXT extension, elsewhere name itself with an execute bit. A directory
// listed twice, even spelled differently, is read once.
func copiesOnPath(name, goos, pathList, pathext string) []string {
	var copies []string
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(pathList) {
		key := filepath.Clean(dir)
		if goos == "windows" {
			key = strings.ToLower(key)
		}
		if dir == "" || seen[key] {
			continue
		}
		seen[key] = true
		for _, n := range commandNames(name, goos, pathext) {
			if p := filepath.Join(dir, n); runsAs(p, goos) {
				copies = append(copies, p)
			}
		}
	}
	return copies
}

// commandNames is the file names a command resolves to: on Windows one per
// PATHEXT extension, in PATHEXT order; elsewhere the name alone.
func commandNames(name, goos, pathext string) []string {
	if goos != "windows" {
		return []string{name}
	}
	if pathext == "" {
		pathext = ".com;.exe;.bat;.cmd"
	}
	var names []string
	for _, ext := range strings.Split(pathext, ";") {
		if ext != "" {
			names = append(names, name+strings.ToLower(ext))
		}
	}
	return names
}

// runsAs reports whether p is a regular file the OS would run: on Windows its
// extension already says so, elsewhere it needs an execute bit.
func runsAs(p, goos string) bool {
	fi, err := os.Stat(p)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	return goos == "windows" || fi.Mode()&0o111 != 0
}

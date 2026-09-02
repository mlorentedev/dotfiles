package doctor

import (
	"fmt"
	"strconv"
	"strings"
)

// checkDotfProvenance answers a question no other check in this file set can:
// "was the dotf on PATH built from this tree?"
//
// WHY VERSION-EQUALITY CANNOT ANSWER IT. checkOptionalTools compares the
// reported semver against the versions.conf pin and warns on drift. For every
// other tool in that list — third-party releases — that is the whole question.
// `dotf` is unlike them in one way that matters: it IS this repository. Between
// two releases the version string is CONSTANT while the source moves on every
// merge, so a matching version proves nothing about the code inside. Measured
// 2026-09-01 (#1158): the deployed binary read 0.52.0, versions.conf read
// 0.52.0, and the binary was two feature merges stale — it predated #1410, so
// `dotf doctor` rendered a clean run with the "Persona skill enforcement"
// SECTION ENTIRELY ABSENT. A report missing a check is indistinguishable from
// one where the check ran and passed.
//
// That matters more here than for any other tool because every gate in this
// repo runs through dotf — `spec review`, `spec archive`, `pr triage-queue`,
// `secrets run`, `harness gate`. A stale binary can enforce a contract that no
// longer exists, or fail to enforce one that does, and report success either way.
//
// WARN, NEVER FAIL. Running a released binary from inside a checkout is
// legitimate and common — it is what every non-CLI session on this machine does.
// The defect is not the drift; it is the drift being INVISIBLE. Failing the
// health command over a normal state would train the reader to ignore it, which
// is the same mistake one layer up.
//
// The comparison is scoped to `cli/`: commits that touch only docs, specs or
// harness records move HEAD without changing what the binary would do, and
// reporting those as staleness would make the check noisy enough to ignore.
func checkDotfProvenance(sys *System, cfg *Config, rep *Report) {
	rep.Section("dotf provenance (deployed binary vs checkout)")

	// C15: a check that cannot answer must say so rather than pass. Outside a
	// checkout there is no HEAD to compare against — that is not a defect, it is
	// most machines.
	if cfg.RepoDir == "" {
		rep.Skip("not inside a dotfiles checkout — no HEAD to compare the deployed binary against")
		return
	}
	binPath, err := sys.LookPath("dotf")
	if err != nil {
		rep.Skip("dotf is not on PATH — nothing deployed to compare")
		return
	}

	head, err := gitIn(sys, cfg.RepoDir, "rev-parse", "HEAD")
	if err != nil {
		rep.Skip(fmt.Sprintf("cannot read HEAD in %s (%v) — provenance not established", cfg.RepoDir, err))
		return
	}

	// A binary built before `--commit` existed answers `unknown flag: --commit`
	// and exits non-zero. That IS the answer for a pre-stamp binary, so the error
	// is reported as provenance rather than swallowed as a probe failure.
	raw, err := sys.CommandOutput(binPath, "version", "--commit")
	if err != nil {
		rep.Warn(fmt.Sprintf(
			"deployed dotf (%s) does not understand `version --commit`, so it predates the build stamp and its provenance CANNOT be established — reinstall from the current release to make this answerable",
			binPath))
		return
	}

	stamp := strings.TrimSpace(raw)
	if stamp == "" {
		// A source build. Deliberate on a dev box and in CI, and the installers
		// refuse to replace one — so this is reported, not scolded.
		rep.Warn(fmt.Sprintf(
			"deployed dotf (%s) is a source build carrying no commit stamp — legitimate on a dev box, but its provenance cannot be checked against HEAD (%s)",
			binPath, shortSHA(head)))
		return
	}

	if _, err := gitIn(sys, cfg.RepoDir, "cat-file", "-e", stamp+"^{commit}"); err != nil {
		rep.Warn(fmt.Sprintf(
			"deployed dotf was built from commit %s, which this checkout does not contain — fetch, or reinstall from a release built off this history",
			shortSHA(stamp)))
		return
	}

	// Not an ancestor: the binary came from a branch, or the checkout is behind
	// the binary. Distinct from "behind" and worth its own message, because the
	// remedy is the opposite one.
	if _, err := gitIn(sys, cfg.RepoDir, "merge-base", "--is-ancestor", stamp, "HEAD"); err != nil {
		rep.Warn(fmt.Sprintf(
			"deployed dotf was built from %s, which is not an ancestor of HEAD (%s) — a branch build, or this checkout is behind the binary",
			shortSHA(stamp), shortSHA(head)))
		return
	}

	// `:(top)cli`, not `cli`. A bare pathspec is resolved relative to git's CWD,
	// so the same argument means `cli/` from the repo root and `cli/cli` — which
	// does not exist — from inside cli/. Measured while verifying this check: the
	// count read 4 from the root and 0 from cli/, and the 0 is SILENT. Relying on
	// RepoDir always being the root would make this guard return a clean answer
	// the day that resolution changes, which is the exact failure mode #1158 is
	// about. The `:(top)` magic prefix is root-relative regardless of CWD.
	out, err := gitIn(sys, cfg.RepoDir, "rev-list", "--count", stamp+"..HEAD", "--", ":(top)cli")
	if err != nil {
		rep.Skip(fmt.Sprintf("cannot count commits between %s and HEAD (%v)", shortSHA(stamp), err))
		return
	}
	behind, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		rep.Skip(fmt.Sprintf("unexpected git rev-list output %q — provenance not established", strings.TrimSpace(out)))
		return
	}

	if behind == 0 {
		rep.Pass(fmt.Sprintf("deployed dotf was built from %s, current with HEAD for cli/", shortSHA(stamp)))
		return
	}
	rep.Warn(fmt.Sprintf(
		"deployed dotf is %d cli/ commit(s) behind HEAD (built %s, HEAD %s) — it may run gates this tree no longer defines, or miss checks it does; reinstall to converge",
		behind, shortSHA(stamp), shortSHA(head)))
}

// gitIn runs git in dir through the System seam so the check is table-testable
// without a real repository.
func gitIn(sys *System, dir string, args ...string) (string, error) {
	return sys.CommandOutputDir(dir, "git", args...)
}

// shortSHA abbreviates for the message only. Every git call above uses the full
// value: an abbreviated hash is ambiguous in principle and does not resolve at
// all in a shallow clone, which is what CI checks out by default.
func shortSHA(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

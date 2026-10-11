package doctor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Options configures a doctor run. System and StartDir are injection seams for
// tests; production leaves them zero (realSystem + os.Getwd).
type Options struct {
	Out      io.Writer
	Fix      bool
	Verbose  bool
	Quick    bool    // env-contract sweep only — fast, for the SessionStart hook (CLI-013)
	Scope    Scope   // ScopeAll (zero) runs every check; ScopeMachine skips the identity ones
	System   *System // nil → realSystem()
	StartDir string  // "" → os.Getwd()
}

// Run executes the full consolidated diagnostic sweep and returns the process
// exit code (0 = all checks passed, 1 = at least one FAIL). It never aborts the
// sweep on a missing/invalid contract — that is surfaced as a FAIL and the
// remaining sections still run, so one command always yields the complete
// picture (the consolidation win over the two separate twins).
//
// The only non-nil error path is a genuinely undiagnosable environment (cannot
// determine the working directory), reported with exit code 2.
func Run(opts Options) (int, error) {
	sys := opts.System
	if sys == nil {
		sys = realSystem()
	}
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	start := opts.StartDir
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return 2, fmt.Errorf("cannot determine working directory: %w", err)
		}
		start = wd
	}

	cfg, err := loadConfig(sys, start)
	if err != nil {
		return 2, err
	}

	// Tee the report into a transcript so Summary() can be followed by a
	// curated Next-steps block (below) without changing Report's shape: it
	// streams messages straight to out and keeps no record of them (CLI-070,
	// #1442 — setup's own "Next steps" never named the FAIL doctor had just
	// printed, e.g. `bw login`, so the reader had to go find it in the
	// scroll). Color detection must run on the real out, not the tee:
	// io.MultiWriter is never an *os.File, so isColorEnabled would otherwise
	// report false on every run.
	color := isColorEnabled(out)
	var transcript bytes.Buffer
	rep := NewReport(io.MultiWriter(out, &transcript), opts.Verbose)
	rep.SetColor(color)
	mode := "check"
	switch {
	case opts.Quick:
		mode = "quick"
	case opts.Fix:
		mode = "fix"
	}
	if opts.Scope == ScopeMachine {
		mode += ", machine scope"
	}
	_, _ = fmt.Fprintf(out, "dotf doctor [%s] — diagnostics for %s\n", mode, cfg.DotfilesDir)

	contract := loadContractSection(sys, cfg, rep)

	// Folded env-contract sweep (the doctor.sh surface). In --quick mode this is
	// the ONLY work: it is the fast, fork-free subset wired into the SessionStart
	// hook (CLI-013), so it must skip the heavy healthcheck sweep below — chiefly
	// the ~2.8s compile-harness drift gate.
	if contract != nil {
		checkContractEnvVars(sys, contract, rep, opts.Fix)
		checkPersistedEnv(sys, cfg, rep)
		checkContractPath(sys, contract, rep)
		checkRequiredBinaries(sys, contract, rep)
	}

	if !opts.Quick {
		for _, c := range sweep(sys, cfg, contract, rep, opts, start) {
			if opts.Scope == ScopeMachine && c.kind == kindIdentity {
				rep.Section(c.section)
				rep.Skip("an identity check, not run under --scope machine: " + c.restore)
				continue
			}
			c.run()
		}
	}

	rep.Summary()
	if steps := nextSteps(transcript.String()); len(steps) > 0 {
		_, _ = fmt.Fprintln(out, "\nNext steps:")
		for _, step := range steps {
			_, _ = fmt.Fprintf(out, "  %s\n", step)
		}
	}
	return rep.ExitCode(), nil
}

// failRemedyRe pulls the backtick-quoted command out of a FAIL line's remedy
// clause. Matched against the handful of verbs FAIL messages actually use to
// introduce one (surveyed across the package: run/re-run cover the large
// majority, recover with/upgrade with the rest) — not every backtick span,
// which would also catch a diagnostic reference like `bw status` that names
// what was checked, not what to do about it.
var failRemedyRe = regexp.MustCompile("(?:run|re-run|recover with|upgrade with) `([^`]+)`")

// nextSteps scans a rendered report transcript for FAIL lines carrying a
// remedy command and returns each one once, in first-seen order. Free text,
// not a structured field on Report: Report streams a message to its writer
// and keeps no record of it, and giving every check a machine-readable hint
// would be a much larger change than one FAIL in the middle of a 30+ section
// sweep (Bitwarden reach) actually needs surfaced at the end.
func nextSteps(transcript string) []string {
	seen := map[string]bool{}
	var steps []string
	for _, line := range strings.Split(transcript, "\n") {
		if !strings.Contains(line, "[FAIL]") {
			continue
		}
		for _, m := range failRemedyRe.FindAllStringSubmatch(line, -1) {
			cmd := m[1]
			if !seen[cmd] {
				seen[cmd] = true
				steps = append(steps, cmd)
			}
		}
	}
	return steps
}

// loadContractSection resolves + parses the contract, reporting its own status
// as a section. A missing or invalid contract is a FAIL (not a fatal abort):
// the env-contract checks are skipped but the rest of the sweep proceeds.
func loadContractSection(sys *System, cfg *Config, rep *Report) *Contract {
	rep.Section("Environment contract")
	if cfg.ContractPath == "" {
		rep.Fail("env-contract.json not found under DOTFILES_DIR or repo root — contract checks skipped")
		return nil
	}
	contract, err := loadContract(cfg.ContractPath)
	if err != nil {
		rep.Fail(fmt.Sprintf("env-contract.json unreadable (%v) — contract checks skipped", err))
		return nil
	}
	rep.Pass("env-contract.json loaded")
	// Provenance (always shown, even in non-verbose where Pass is suppressed) so a
	// stale-deployed-copy read is self-diagnosing rather than a silent contradiction
	// with `dotf env generate` (#697).
	rep.Info("contract: " + cfg.ContractPath)
	return contract
}

// Scope selects the checks a run covers (#2013 D10). Its zero value runs all of
// them, so a plain `dotf doctor` is unchanged.
type Scope int

const (
	ScopeAll     Scope = iota
	ScopeMachine       // what converge produces; for a fresh machine or CI runner
)

// ParseScope reads the --scope flag: "all" (or "") and "machine".
func ParseScope(s string) (Scope, error) {
	switch s {
	case "", "all":
		return ScopeAll, nil
	case "machine":
		return ScopeMachine, nil
	}
	return ScopeAll, fmt.Errorf("doctor: unknown --scope %q (want all or machine)", s)
}

// kind is what a check needs before it can pass. The zero value is invalid:
// every sweep entry names one of the other two, and a test holds it to that,
// so a check added without a decision cannot fall into either scope by
// default.
type kind int

const (
	kindUnset    kind = iota
	kindMachine       // installed and configured by converge
	kindIdentity      // the owner's credentials or vault: restored, never converged
)

// What an identity check needs, as the SKIP line under --scope machine names
// it. The steps are docs/runbooks/guide-new-machine.md's.
const (
	restoreAge   = "restore the age key from the offline backup (docs/runbooks/guide-new-machine.md, step 1)"
	restoreBW    = "run `bw login`, then `dotf secrets unlock` (docs/runbooks/guide-new-machine.md, step 2)"
	restoreGH    = "run `gh auth login` (docs/runbooks/guide-new-machine.md, step 3)"
	restoreVault = "clone the knowledge vault (docs/runbooks/guide-new-machine.md, step 4)"
)

// check is one entry of the full sweep. section and restore are read only for
// an identity check skipped by scope; section is the title its run reports
// under.
type check struct {
	kind    kind
	section string
	restore string
	run     func()
}

// sweep is the full diagnostic sweep (the healthcheck.sh sections and what
// followed them), in report order, each check classified machine or identity.
func sweep(sys *System, cfg *Config, contract *Contract, rep *Report, opts Options, start string) []check {
	return []check{
		{kind: kindMachine, run: func() { checkCoreTools(sys, contract, rep) }},
		{kind: kindMachine, run: func() { checkVersionedPaths(sys, rep) }},
		{kind: kindMachine, run: func() { checkVersionMatch(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkSymlinks(sys, rep) }},
		{kind: kindMachine, run: func() { checkProfileFiles(sys, contract, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkToolHomeEnvVars(sys, rep) }},
		{kind: kindMachine, run: func() { checkOptionalTools(sys, cfg, contract, rep) }},
		{kind: kindMachine, run: func() { checkMiseTools(sys, cfg, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkSystemPackages(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkPython(sys, cfg, rep) }},
		{kind: kindIdentity, section: "Knowledge vault (presence)", restore: restoreVault, run: func() { checkVault(sys, rep) }},
		{kind: kindIdentity, section: "Knowledge vault hooks (secret gate)", restore: restoreVault, run: func() { checkVaultHooks(sys, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkGitConfig(sys, rep, opts.Fix) }},
		{kind: kindIdentity, section: "Auto-memory vault link", restore: restoreVault, run: func() { checkAutoMemoryLink(sys, start, rep, opts.Fix) }},
		{kind: kindIdentity, section: "Auto-memory file shape", restore: restoreVault, run: func() { checkMemoryShape(sys, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkPathFiles(sys, cfg, rep) }},
		{kind: kindIdentity, section: "Secrets integrity", restore: restoreAge, run: func() { checkSecrets(sys, cfg, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkSecretsTooling(sys, cfg, rep) }},
		{kind: kindIdentity, section: "Age identity key", restore: restoreAge, run: func() { checkAgeIdentity(sys, rep) }},
		{kind: kindIdentity, section: "Bitwarden reach (live secrets SSOT)", restore: restoreBW, run: func() { checkBitwardenReach(sys, rep) }},
		{kind: kindIdentity, section: "bw serve daemon (optional local unlock cache)", restore: restoreBW, run: func() { checkBWServeDaemon(sys, cfg, rep) }},
		{kind: kindIdentity, section: "Bitwarden mapping (registry -> vault)", restore: restoreBW, run: func() { checkBWMapping(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkAgentConfigSecrets(sys, rep) }},
		{kind: kindMachine, run: func() { checkHiveBackendCanServe(sys, rep) }},
		{kind: kindMachine, run: func() { checkHiveDaemonAnswers(sys, cfg, rep) }},
		{kind: kindIdentity, section: "Disaster recovery", restore: restoreAge, run: func() { checkDisasterRecovery(sys, cfg, rep) }},
		{kind: kindIdentity, section: "PAT expiry", restore: restoreBW, run: func() { checkPATExpiry(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkGuardHooks(sys, cfg, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkTmux(sys, rep) }},
		{kind: kindMachine, run: func() { checkOpenCode(sys, cfg, rep) }},
		{kind: kindIdentity, section: "Agent configs from secrets", restore: restoreBW, run: func() { checkAgentSecretConfigs(sys, rep) }},
		{kind: kindMachine, run: func() { checkCopilot(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkGolangciLint(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkModelMap(cfg, rep) }},
		{kind: kindMachine, run: func() { checkModelPins(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkModelLimits(sys, cfg, rep) }},
		{kind: kindIdentity, section: "NaN quota", restore: restoreBW, run: func() { checkNaNQuota(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkPiExtensions(sys, cfg, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkPiPackageRequirements(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkHarnessDrift(sys, cfg, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkDeployDrift(sys, cfg, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkHomeDeployDrift(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkDockerEngine(sys, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkColimaSize(sys, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkDockerCompose(sys, rep) }},
		{kind: kindMachine, run: func() { checkDeployManifest(sys, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkAgentPresence(sys, rep) }},
		{kind: kindMachine, run: func() { checkAgentSkillsMigrated(cfg, rep) }},
		{kind: kindMachine, run: func() { checkDotfProvenance(sys, cfg, rep) }},
		{kind: kindMachine, run: func() { checkRepoDirResolves(sys, rep) }},
		{kind: kindIdentity, section: "spec-issue-state", restore: restoreGH, run: func() { checkSpecIssueState(sys, rep) }},
		{kind: kindIdentity, section: "branch-protection", restore: restoreGH, run: func() { checkBranchProtection(sys, rep) }},
		{kind: kindMachine, run: func() { checkAntigravity(sys, rep) }},
		{kind: kindMachine, run: func() { checkOrcaHook(sys, rep, opts.Fix) }},
		{kind: kindMachine, run: func() { checkHookBinding(sys, rep, opts.Fix) }},
	}
}

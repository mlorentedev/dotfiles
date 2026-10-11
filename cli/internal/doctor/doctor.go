package doctor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
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

// sweepEnv carries what the checks of the full sweep read.
type sweepEnv struct {
	sys      *System
	cfg      *Config
	contract *Contract
	rep      *Report
	opts     Options
	start    string
}

// sweep is the full diagnostic sweep (the healthcheck.sh sections and what
// followed them), in report order, each check classified machine or identity.
// The groups only keep each table short; the order across them is the report's.
func sweep(sys *System, cfg *Config, contract *Contract, rep *Report, opts Options, start string) []check {
	e := sweepEnv{sys: sys, cfg: cfg, contract: contract, rep: rep, opts: opts, start: start}
	return slices.Concat(e.toolchain(), e.knowledgeAndSecrets(), e.agents(), e.deployAndHooks())
}

// toolchain: the machine's tools, versions, links and profile files.
func (e sweepEnv) toolchain() []check {
	return []check{
		{kind: kindMachine, run: func() { checkCoreTools(e.sys, e.contract, e.rep) }},
		{kind: kindMachine, run: func() { checkVersionedPaths(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkVersionMatch(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkSymlinks(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkProfileFiles(e.sys, e.contract, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkToolHomeEnvVars(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkOptionalTools(e.sys, e.cfg, e.contract, e.rep) }},
		{kind: kindMachine, run: func() { checkMiseTools(e.sys, e.cfg, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkSystemPackages(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkPython(e.sys, e.cfg, e.rep) }},
	}
}

// knowledgeAndSecrets: the vault, the memory link, secrets, Bitwarden and the hive daemon.
func (e sweepEnv) knowledgeAndSecrets() []check {
	return []check{
		{kind: kindIdentity, section: "Knowledge vault (presence)", restore: restoreVault, run: func() { checkVault(e.sys, e.rep) }},
		{kind: kindIdentity, section: "Knowledge vault hooks (secret gate)", restore: restoreVault, run: func() { checkVaultHooks(e.sys, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkGitConfig(e.sys, e.rep, e.opts.Fix) }},
		{kind: kindIdentity, section: "Auto-memory vault link", restore: restoreVault, run: func() { checkAutoMemoryLink(e.sys, e.start, e.rep, e.opts.Fix) }},
		{kind: kindIdentity, section: "Auto-memory file shape", restore: restoreVault, run: func() { checkMemoryShape(e.sys, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkPathFiles(e.sys, e.cfg, e.rep) }},
		{kind: kindIdentity, section: "Secrets integrity", restore: restoreAge, run: func() { checkSecrets(e.sys, e.cfg, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkSecretsTooling(e.sys, e.cfg, e.rep) }},
		{kind: kindIdentity, section: "Age identity key", restore: restoreAge, run: func() { checkAgeIdentity(e.sys, e.rep) }},
		{kind: kindIdentity, section: "Bitwarden reach (live secrets SSOT)", restore: restoreBW, run: func() { checkBitwardenReach(e.sys, e.rep) }},
		{kind: kindIdentity, section: "bw serve daemon (optional local unlock cache)", restore: restoreBW, run: func() { checkBWServeDaemon(e.sys, e.cfg, e.rep) }},
		{kind: kindIdentity, section: "Bitwarden mapping (registry -> vault)", restore: restoreBW, run: func() { checkBWMapping(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkAgentConfigSecrets(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkHiveBackendCanServe(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkHiveDaemonAnswers(e.sys, e.cfg, e.rep) }},
		{kind: kindIdentity, section: "Disaster recovery", restore: restoreAge, run: func() { checkDisasterRecovery(e.sys, e.cfg, e.rep) }},
		{kind: kindIdentity, section: "PAT expiry", restore: restoreBW, run: func() { checkPATExpiry(e.sys, e.cfg, e.rep) }},
	}
}

// agents: the guard hooks, tmux and the agents with their configs and model pins.
func (e sweepEnv) agents() []check {
	return []check{
		{kind: kindMachine, run: func() { checkGuardHooks(e.sys, e.cfg, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkTmux(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkOpenCode(e.sys, e.cfg, e.rep) }},
		{kind: kindIdentity, section: "Agent configs from secrets", restore: restoreBW, run: func() { checkAgentSecretConfigs(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkCopilot(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkGolangciLint(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkModelMap(e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkModelPins(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkModelLimits(e.sys, e.cfg, e.rep) }},
		{kind: kindIdentity, section: "NaN quota", restore: restoreBW, run: func() { checkNaNQuota(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkPiExtensions(e.sys, e.cfg, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkPiPackageRequirements(e.sys, e.cfg, e.rep) }},
	}
}

// deployAndHooks: drift between repo and deploy, containers, provenance, the repo's GitHub state and the hook bindings.
func (e sweepEnv) deployAndHooks() []check {
	return []check{
		{kind: kindMachine, run: func() { checkHarnessDrift(e.sys, e.cfg, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkDeployDrift(e.sys, e.cfg, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkHomeDeployDrift(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkDockerEngine(e.sys, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkColimaSize(e.sys, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkDockerCompose(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkDeployManifest(e.sys, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkAgentPresence(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkAgentSkillsMigrated(e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkDotfProvenance(e.sys, e.cfg, e.rep) }},
		{kind: kindMachine, run: func() { checkRepoDirResolves(e.sys, e.rep) }},
		{kind: kindIdentity, section: "spec-issue-state", restore: restoreGH, run: func() { checkSpecIssueState(e.sys, e.rep) }},
		{kind: kindIdentity, section: "branch-protection", restore: restoreGH, run: func() { checkBranchProtection(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkAntigravity(e.sys, e.rep) }},
		{kind: kindMachine, run: func() { checkOrcaHook(e.sys, e.rep, e.opts.Fix) }},
		{kind: kindMachine, run: func() { checkHookBinding(e.sys, e.rep, e.opts.Fix) }},
	}
}

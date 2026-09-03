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
		// healthcheck.sh 12-section sweep (minus diff-check + deep vault-health).
		checkCoreTools(sys, contract, rep)
		checkVersionedPaths(sys, rep)
		checkVersionMatch(sys, cfg, rep)
		checkSymlinks(sys, rep)
		checkProfileFiles(sys, contract, rep, opts.Fix)
		checkToolHomeEnvVars(sys, rep)
		checkOptionalTools(sys, cfg, contract, rep)
		checkVault(sys, rep)
		checkVaultHooks(sys, rep, opts.Fix)
		checkAutoMemoryLink(sys, start, rep, opts.Fix)
		checkMemoryShape(sys, rep, opts.Fix)
		checkPathFiles(sys, cfg, rep)
		checkSecrets(sys, cfg, rep)
		checkSecretsTooling(sys, cfg, rep)
		checkBitwardenReach(sys, rep)
		checkBWServeDaemon(sys, cfg, rep)
		checkBWMapping(sys, cfg, rep)
		checkAgentConfigSecrets(sys, rep)
		checkHiveBackendCanServe(sys, rep)
		checkDisasterRecovery(sys, cfg, rep)
		checkPATExpiry(sys, cfg, rep)
		checkGuardHooks(sys, cfg, rep, opts.Fix)
		checkTmux(sys, cfg, rep)
		checkOpenCode(sys, cfg, rep)
		checkCopilot(sys, cfg, rep)
		checkGolangciLint(sys, cfg, rep)
		checkModelMap(cfg, rep)
		checkModelPins(sys, cfg, rep)
		checkPiExtensions(sys, cfg, rep, opts.Fix)
		checkHarnessDrift(sys, cfg, rep, opts.Fix)
		checkDeployDrift(sys, cfg, rep)
		checkHomeDeployDrift(sys, cfg, rep)
		checkDeployManifest(sys, rep)
		checkAgentPresence(sys, rep)
		checkAgentSkillsMigrated(cfg, rep)
		checkDotfProvenance(sys, cfg, rep)
		checkRepoDirResolves(rep)
		checkAntigravity(sys, rep)
		checkOrcaHook(sys, rep, opts.Fix)
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

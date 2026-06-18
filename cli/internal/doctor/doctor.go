package doctor

import (
	"fmt"
	"io"
	"os"
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

	rep := NewReport(out, opts.Verbose)
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
		checkContractPath(sys, contract, rep)
		checkRequiredBinaries(sys, contract, rep)
	}

	if !opts.Quick {
		// healthcheck.sh 12-section sweep (minus diff-check + deep vault-health).
		checkCoreTools(sys, contract, rep)
		checkVersionedPaths(sys, rep)
		checkVersionMatch(sys, cfg, rep)
		checkSymlinks(sys, rep)
		checkToolHomeEnvVars(sys, rep)
		checkOptionalTools(sys, cfg, contract, rep)
		checkVault(sys, rep)
		checkSecrets(sys, cfg, rep)
		checkPATExpiry(sys, cfg, rep)
		checkGuardHooks(sys, cfg, rep, opts.Fix)
		checkTmux(sys, cfg, rep)
		checkOpenCode(sys, cfg, rep)
		checkHarnessDrift(sys, cfg, rep)
		checkAntigravity(sys, rep)

		if opts.Fix {
			runHeals(sys, cfg, rep)
		}
	}

	rep.Summary()
	return rep.ExitCode(), nil
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
	rep.Pass("env-contract.json loaded from " + cfg.ContractPath)
	return contract
}

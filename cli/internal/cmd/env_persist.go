package cmd

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/spf13/cobra"
)

// userEnvStore is the seam tests replace so the command never touches a real
// registry; production resolves the OS store.
var userEnvStore = env.NewUserEnvStore

// newEnvPersistCmd is CLI-058 (#1324): the ADR-025 cascade exists only in the
// rc files, so a process started with no profile — Copilot's `pwsh
// -NoProfile` tool calls, a Scheduled Task — sees none of DOTFILES_REPO_DIR,
// DOTFILES_DIR, VAULT_PATH, SCRIPTS_DIR... `persist` writes the same resolved
// values `generate` renders into the per-user persistent scope, touching only
// what differs, so a second run is a no-op.
func newEnvPersistCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "persist",
		Short: "Write the resolved contract variables into the per-user persistent scope (Windows: HKCU\\Environment)",
		Long: "persist resolves every structural variable exactly as `generate` does\n" +
			"(env-contract.json defaults + machine.json overrides) and writes each one\n" +
			"into the OS's per-user persistent environment, the scope a process\n" +
			"started without a profile inherits. Idempotent: only values that differ\n" +
			"are written. --check reports drift without writing (non-zero when drifted).\n" +
			"The names it wrote are recorded in the store as " + env.ManagedMarker + " (';'-joined);\n" +
			"a name that record lists and the contract no longer names is deleted on the\n" +
			"next run (CLI-065) — a variable dotf never wrote is never touched.\n" +
			"Where the OS has no such scope (Linux, macOS) it is a no-op: the rc files\n" +
			"source paths.sh and unit files carry their own environment.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := userEnvStore()
			if errors.Is(err, env.ErrUserEnvUnsupported) {
				cmd.Printf("nothing to persist on %s: the rc files source the generated path file\n", runtime.GOOS)
				return nil
			}
			if err != nil {
				return err
			}
			vars, err := resolvePersistVars()
			if err != nil {
				return err
			}
			if check {
				return checkPersisted(cmd, vars, store)
			}
			return runPersist(cmd, vars, store)
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "report variables missing or different at user scope without writing")
	return cmd
}

// resolvePersistVars resolves the contract the way `generate` does, for the
// home and OS this process runs under.
func resolvePersistVars() ([]env.ResolvedVar, error) {
	home := env.Home()
	contractPath := env.ResolveContractPath()
	if contractPath == "" {
		return nil, fmt.Errorf("env-contract.json not found: set DOTFILES_DIR or run from the repo")
	}
	return env.ResolveVars(contractPath, env.MachinePath(home), runtime.GOOS, home)
}

// checkPersisted is the read-only mirror of runPersist: it names every write
// a run would make — a missing or different variable, a retired name still in
// the store, an ownership record that lags the contract — and exits non-zero
// while one remains. A clean --check means a run would change nothing.
func checkPersisted(cmd *cobra.Command, vars []env.ResolvedVar, store env.UserEnvReader) error {
	drift, err := env.Drift(vars, store)
	if err != nil {
		return err
	}
	retired, err := env.Retired(store, vars)
	if err != nil {
		return err
	}
	stale, err := env.MarkerStale(store, vars)
	if err != nil {
		return err
	}
	for _, v := range drift {
		cmd.Printf("drift: %s\n", v.Name)
	}
	for _, name := range retired {
		cmd.Printf("retired: %s\n", name)
	}
	if stale {
		cmd.Printf("record: %s does not match the contract\n", env.ManagedMarker)
	}
	switch {
	case len(drift) > 0 && len(retired) > 0:
		return fmt.Errorf("%d variable(s) not persisted and %d retired name(s) still persisted at user scope — run `dotf env persist`", len(drift), len(retired))
	case len(drift) > 0:
		return fmt.Errorf("%d variable(s) not persisted at user scope — run `dotf env persist`", len(drift))
	case len(retired) > 0:
		return fmt.Errorf("%d retired name(s) still persisted at user scope — run `dotf env persist` to sweep them", len(retired))
	case stale:
		return fmt.Errorf("the ownership record (%s) is out of date — run `dotf env persist` to rewrite it", env.ManagedMarker)
	}
	cmd.Printf("ok: %d variable(s) persisted at user scope\n", len(vars))
	return nil
}

// runPersist sweeps, writes and marks, and reports each write on stdout —
// the stream the setup scripts read.
func runPersist(cmd *cobra.Command, vars []env.ResolvedVar, store env.UserEnvStore) error {
	res, err := env.Persist(vars, store)
	if err != nil {
		return err
	}
	changed, removed := 0, 0
	for _, r := range res {
		switch {
		case r.Removed:
			removed++
			cmd.Printf("removed %s (retired from the contract)\n", r.Name)
		case r.Changed:
			changed++
			cmd.Printf("persisted %s\n", r.Name)
		}
	}
	cmd.Printf("user scope: %d changed, %d unchanged, %d removed\n", changed, len(res)-changed-removed, removed)
	return nil
}

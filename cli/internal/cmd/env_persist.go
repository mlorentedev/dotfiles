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
			home := env.Home()
			contractPath := env.ResolveContractPath()
			if contractPath == "" {
				return fmt.Errorf("env-contract.json not found: set DOTFILES_DIR or run from the repo")
			}
			vars, err := env.ResolveVars(contractPath, env.MachinePath(home), runtime.GOOS, home)
			if err != nil {
				return err
			}
			if check {
				drift, err := env.Drift(vars, store)
				if err != nil {
					return err
				}
				if len(drift) > 0 {
					for _, v := range drift {
						cmd.Printf("drift: %s\n", v.Name)
					}
					return fmt.Errorf("%d variable(s) not persisted at user scope — run `dotf env persist`", len(drift))
				}
				cmd.Printf("ok: %d variable(s) persisted at user scope\n", len(vars))
				return nil
			}
			res, err := env.Persist(vars, store)
			if err != nil {
				return err
			}
			changed := 0
			for _, r := range res {
				if r.Changed {
					changed++
					cmd.Printf("persisted %s\n", r.Name)
				}
			}
			cmd.Printf("user scope: %d changed, %d unchanged\n", changed, len(res)-changed)
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "report variables missing or different at user scope without writing")
	return cmd
}

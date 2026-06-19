package cmd

import (
	"fmt"
	"runtime"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/spf13/cobra"
)

// newEnvCmd is the `dotf env` noun: per-machine path resolution and the
// generated path files shells source (ADR-025).
func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Per-machine path resolution (paths.sh / paths.ps1)",
		Long: "env renders the per-machine path file from env-contract.json (defaults)\n" +
			"+ ~/.config/dotfiles/machine.json (overrides). Shells source the result, so a\n" +
			"machine that relocates a repo edits machine.json and re-runs generate (ADR-025).",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newEnvGenerateCmd())
	cmd.AddCommand(newEnvPathCmd())
	return cmd
}

func newEnvPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path <KEY>",
		Short: "Print the cascade-resolved value of one path key",
		Long: "path resolves a single structural path through the ADR-025 cascade\n" +
			"(env -> machine.json -> contract default[GOOS]) and prints it. Used by\n" +
			"setup scripts to provision service environments (e.g. the hive daemon)\n" +
			"that do not source the shell path file. Prints an empty line + exits 0\n" +
			"when the key has no resolved value, so callers can fall back with ${X:-default}.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println(env.ResolvePath(args[0]))
			return nil
		},
	}
}

func newEnvGenerateCmd() *cobra.Command {
	var output string
	var check, stdout bool

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Render paths.sh / paths.ps1 from the contract + machine overrides",
		Long: "generate resolves every structural path (env -> machine.json -> contract\n" +
			"default for this OS) and writes the OS-appropriate path file to\n" +
			"<DOTFILES_DIR>/paths.sh|ps1. --check reports drift without writing.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home := env.Home()
			contractPath := env.ResolveContractPath()
			if contractPath == "" {
				return fmt.Errorf("env-contract.json not found: set DOTFILES_DIR or run from the repo")
			}
			if output == "" {
				output = env.DefaultOutput(runtime.GOOS, env.DotfilesDir(home))
			}
			res, err := env.Generate(env.Options{
				ContractPath: contractPath,
				MachinePath:  env.MachinePath(home),
				GOOS:         runtime.GOOS,
				Home:         home,
				Output:       output,
				Check:        check,
				Stdout:       stdout,
			})
			if err != nil {
				return err
			}
			switch {
			case stdout:
				cmd.Print(res.Content)
			case check:
				if res.Drifted {
					return fmt.Errorf("%s is stale — run `dotf env generate`", res.Output)
				}
				cmd.Printf("ok: %s up to date\n", res.Output)
			case res.Wrote:
				cmd.Printf("wrote %s\n", res.Output)
			default:
				cmd.Printf("unchanged %s\n", res.Output)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "output path (default: <DOTFILES_DIR>/paths.sh|ps1)")
	cmd.Flags().BoolVar(&check, "check", false, "report drift without writing (non-zero exit if stale)")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "print to stdout instead of writing the file")
	return cmd
}

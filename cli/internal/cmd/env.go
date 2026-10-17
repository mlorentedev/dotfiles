package cmd

import (
	"fmt"
	"runtime"
	"strings"

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
			"machine that relocates a repo edits machine.json and re-runs generate.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newEnvGenerateCmd())
	cmd.AddCommand(newEnvPathCmd())
	cmd.AddCommand(newEnvSetCmd())
	cmd.AddCommand(newEnvRoleCmd())
	cmd.AddCommand(newEnvPersistCmd())
	return cmd
}

func newEnvSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <KEY> <VALUE>",
		Short: "Set a per-machine path override in machine.json (write-side of `env path`)",
		Long: "set writes one structural path override into\n" +
			"~/.config/dotfiles/machine.json — the write-side counterpart of `env path`.\n" +
			"KEY must be a var declared in env-contract.json (an unknown key is rejected,\n" +
			"so a typo cannot create a dead override no resolver reads); every other\n" +
			"override is preserved and re-setting the same value is a no-op. First-run\n" +
			"setup uses this to seed DOTFILES_REPO_DIR to the checkout it runs from, so the\n" +
			"path cascade resolves the real repo instead of the contract default.",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			contractPath := env.ResolveContractPath()
			if contractPath == "" {
				return fmt.Errorf("env-contract.json not found: set DOTFILES_DIR or run from the repo")
			}
			machinePath := env.MachinePath(env.Home())
			changed, err := env.SetMachinePath(contractPath, machinePath, key, value)
			if err != nil {
				return err
			}
			if changed {
				cmd.Printf("set %s in %s\n", key, machinePath)
			} else {
				cmd.Printf("unchanged %s (already %s)\n", key, value)
			}
			return nil
		},
	}
}

func newEnvRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role [add|rm <ROLE>]",
		Short: "List, add or remove the roles this machine opts into (machine.json)",
		Long: "role records in ~/.config/dotfiles/machine.json the roles this machine\n" +
			"takes beyond the common converge. A role changes what the machine does\n" +
			"unattended or how it faces the network, so none is on by default and an\n" +
			"unknown name is rejected. With no\n" +
			"argument it lists the declared roles. Known roles: " + strings.Join(env.KnownRoles(), ", ") + ".",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			roles, err := env.LoadMachineRoles(env.MachinePath(env.Home()))
			if err != nil {
				return err
			}
			for _, r := range roles {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), r)
			}
			return nil
		},
	}
	for _, sub := range []struct {
		use, short, verb string
		on               bool
	}{
		{"add <ROLE>", "Add a machine role", "added", true},
		{"rm <ROLE>", "Remove a machine role", "removed", false},
	} {
		verb, on := sub.verb, sub.on
		cmd.AddCommand(&cobra.Command{
			Use:          sub.use,
			Short:        sub.short,
			Args:         cobra.ExactArgs(1),
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				path := env.MachinePath(env.Home())
				changed, err := env.SetMachineRole(path, args[0], on)
				if err != nil {
					return err
				}
				if changed {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s role %s in %s\n", verb, args[0], path)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "unchanged role %s\n", args[0])
				}
				return nil
			},
		})
	}
	return cmd
}

func newEnvPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path <KEY>",
		Short: "Print the cascade-resolved value of one path key",
		Long: "path resolves a single structural path through the path cascade\n" +
			"(env -> machine.json -> contract default[GOOS]) and prints it. Used by\n" +
			"setup scripts to provision service environments (e.g. the hive daemon)\n" +
			"that do not source the shell path file. Prints an empty line + exits 0\n" +
			"when the key has no resolved value, so callers can fall back with ${X:-default}.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Fprintln(OutOrStdout()), never cmd.Println: Cobra's Print* family
			// writes to OutOrStderr(), and this value is read through $(...).
			_, err := fmt.Fprintln(cmd.OutOrStdout(), env.ResolvePath(args[0]))
			return err
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
				if _, werr := fmt.Fprint(cmd.OutOrStdout(), res.Content); werr != nil {
					return werr
				}
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

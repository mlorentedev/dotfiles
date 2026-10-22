package cmd

import (
	"errors"
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/spf13/cobra"
)

// errUndeclared is what `dotf env identity` returns on a machine with no id, so
// a script can branch on the exit status without parsing the output.
var errUndeclared = errors.New("this machine has not declared an identity: dotf env identity set <ID>")

func newEnvIdentityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identity [set <ID>]",
		Short: "Show or declare who this machine is (machine.json)",
		Long: "identity records in ~/.config/dotfiles/machine.json the id this machine\n" +
			"declares. A machine with no id may not dispatch to any non-local pool, so\n" +
			"a rebuilt machine that has not restored this file degrades to no\n" +
			"cross-pool dispatch, never to every pool allowed.\n\n" +
			"The id is a free label: nothing is keyed on its value, only on whether one\n" +
			"is declared. It is never derived from the hostname; you name the machine.\n" +
			"With no argument it prints the declared id, or fails when there is none.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := env.LoadMachinePolicy(env.MachinePath(env.Home()))
			if err != nil {
				return err
			}
			if !p.Identified {
				return errUndeclared
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), p.ID)
			return err
		},
	}
	var replace bool
	set := &cobra.Command{
		Use:          "set <ID>",
		Short:        "Declare this machine's id (declaring the same id again changes nothing)",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := env.MachinePath(env.Home())
			changed, err := env.SetMachineIdentity(path, args[0], replace)
			if err != nil {
				return err
			}
			if changed {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "declared identity %s in %s\n", args[0], path)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "unchanged identity %s\n", args[0])
			}
			return nil
		},
	}
	set.Flags().BoolVar(&replace, "replace", false, "rename a machine that already declares a different id")
	cmd.AddCommand(set)
	return cmd
}

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/vm"
)

func newVMCpCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "cp <source> <target>",
		Short: "Copy a file between the host and a session",
		Long: "cp moves one file through the guest's channel, never a mount or a share. One\n" +
			"side is <session>:<guest path>, the other a host path; write a host path with\n" +
			"a colon as ./a:b. A guest path is literal: no glob, no ~. A directory is\n" +
			"refused by the transfer.\n\n" + sessionVerbs,
		Example:      "  dotf vm linux cp ./build.tar.gz proof:/tmp/build.tar.gz\n  dotf vm linux cp proof:/tmp/report.json ./report.json --json",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return withExitCode(2, errors.New("usage: dotf vm linux cp <source> <target>"))
			}
			if _, err := vm.CopySession(args[0], args[1]); err != nil {
				return withExitCode(2, err)
			}
			if _, guest := vm.GuestSide(args[0]); !guest {
				if _, err := os.Stat(args[0]); err != nil {
					return withExitCode(2, fmt.Errorf("source: %w", err))
				}
			}
			s, err := linuxSessions()
			if err != nil {
				return err
			}
			if asJSON {
				return emitSession(cmd, s.Copy(cmd.Context(), args[0], args[1], io.Discard), true)
			}
			return emitSession(cmd, s.Copy(cmd.Context(), args[0], args[1], cmd.ErrOrStderr()), false)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "write one JSON record to stdout")
	return c
}

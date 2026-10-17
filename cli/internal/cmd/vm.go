package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/vm"
)

func newVMCmd() *cobra.Command {
	v := &cobra.Command{
		Use:   "vm",
		Short: "Verify a converge inside a throwaway virtual machine",
	}
	v.AddCommand(newVMLinuxCmd())
	return v
}

func newVMLinuxCmd() *cobra.Command {
	var ref string
	var keep bool
	cmd := &cobra.Command{
		Use:   "linux",
		Short: "Converge a clean Ubuntu 24.04 Lima guest from a pushed ref, twice",
		Long: "linux boots a throwaway Lima guest (Ubuntu 24.04, systemd) that shares no\n" +
			"directory with the host, checks that, clones --ref from GitHub, runs\n" +
			"./install.sh, then `dotf converge` again: the first run must end with no\n" +
			"failed step and the second must change nothing. The guest is deleted at\n" +
			"the end unless --keep.\n\n" +
			"The guest converges with the dotf release the ref's versions.conf pins,\n" +
			"not a build of this tree. The ref must be pushed: the host working tree\n" +
			"never reaches the guest. GITHUB_TOKEN, when set (run under `dotf secrets\n" +
			"run --` to set it), reaches the guest's converges on stdin, for mise's\n" +
			"downloads, and is never written into the guest.",
		Example:      "  dotf vm linux --ref feat/my-branch\n  dotf secrets run -- dotf vm linux",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := exec.LookPath("limactl"); err != nil {
				return withExitCode(2, errors.New("limactl is not on PATH: run `dotf converge --only tools`"))
			}
			out := cmd.OutOrStdout()
			return vm.Verify(execLimactl, vm.Options{
				Ref:   ref,
				Name:  fmt.Sprintf("dotf-linux-%d", time.Now().Unix()),
				Keep:  keep,
				Token: os.Getenv("GITHUB_TOKEN"),
				Out:   out,
			})
		},
	}
	cmd.Flags().StringVar(&ref, "ref", "main", "pushed branch, tag or commit to converge")
	cmd.Flags().BoolVar(&keep, "keep", false, "leave the guest running for inspection")
	return cmd
}

func execLimactl(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	c := exec.Command("limactl", args...)
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
	return c.Run()
}

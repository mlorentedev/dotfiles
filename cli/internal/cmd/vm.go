package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/vm"
)

func newVMCmd() *cobra.Command {
	v := &cobra.Command{
		Use:   "vm",
		Short: "Verify a converge inside a throwaway virtual machine",
	}
	v.AddCommand(newVMLinuxCmd(), newVMWindowsCmd())
	return v
}

func newVMWindowsCmd() *cobra.Command {
	w := &cobra.Command{
		Use:   "windows",
		Short: "A Windows 11 Arm64 test guest on QEMU",
	}
	w.AddCommand(newVMWindowsBuildCmd())
	return w
}

func newVMWindowsBuildCmd() *cobra.Command {
	var iso string
	var rebuild bool
	var cpus, memory int
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Install Windows unattended into the guest's clean-base",
		Long: "build installs Windows 11 Arm64 unattended into clean-base, the image every\n" +
			"run of the guest starts from, then boots it once more to check that the\n" +
			"guest agent answers and that no sshd is installed. clean-base is read-only\n" +
			"once it is complete, so a second build changes nothing; --rebuild replaces it.\n\n" +
			"The Windows ISO is the English (United States) multi-edition Arm64 ISO from\n" +
			"microsoft.com/software-download/windows11arm64. Microsoft's download API\n" +
			"refuses automation, so pass the file a browser downloaded, or its 24-hour\n" +
			"link, with --iso. Either is verified against the SHA256 Microsoft publishes\n" +
			"before anything boots from it. virtio-win is downloaded and verified the\n" +
			"same way. The guest has no TPM and no credential: a local account with no\n" +
			"password, unactivated.",
		Example:      "  dotf vm windows build --iso ~/Downloads/Win11_25H2_English_Arm64.iso",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
				return withExitCode(2, errors.New("the Windows guest runs on a macOS Arm64 host only"))
			}
			host, firmware, vars, err := vm.FindQEMU()
			if err != nil {
				return withExitCode(2, err)
			}
			state, err := env.StateDir()
			if err != nil {
				return err
			}
			return vm.Build(host, vm.BuildOptions{
				Machine: vm.Machine{
					Layout:   vm.Layout{Dir: filepath.Join(state, "vm", "windows")},
					Firmware: firmware,
					CPUs:     cpus,
					MemoryMB: memory,
				},
				VarsTemplate: vars,
				Windows:      vm.WindowsISO,
				Virtio:       vm.VirtioWin,
				ISO:          expandHome(iso),
				Client:       http.DefaultClient,
				Rebuild:      rebuild,
				DiskSize:     "64G",
				BootKeys:     30 * time.Second,
				KeyEvery:     time.Second,
				Install:      2 * time.Hour,
				Agent:        10 * time.Minute,
				Out:          cmd.OutOrStdout(),
			})
		},
	}
	cmd.Flags().StringVar(&iso, "iso", "", "the Windows ISO: a downloaded file or Microsoft's download link")
	cmd.Flags().BoolVar(&rebuild, "rebuild", false, "replace an existing or unfinished clean-base")
	cmd.Flags().IntVar(&cpus, "cpus", 4, "virtual CPUs")
	cmd.Flags().IntVar(&memory, "memory", 8192, "memory in MiB")
	return cmd
}

// expandHome turns a leading ~/ into the home directory: a flag value quoted
// in the shell reaches dotf unexpanded.
func expandHome(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return p
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

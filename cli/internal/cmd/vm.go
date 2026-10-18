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
	var ref, dotf string
	var keep bool
	var cpus, memory int
	w := &cobra.Command{
		Use:   "windows",
		Short: "Converge a clean Windows 11 Arm64 guest from a committed ref, twice",
		Long: "windows boots a throwaway copy of the guest's clean-base (`dotf vm windows\n" +
			"build` makes it), copies in a git archive of --ref, and runs its install.ps1\n" +
			"as the guest's user, then `dotf converge` again: the first run must end with\n" +
			"no failed step and the second must change nothing. The copy is discarded at\n" +
			"the end unless --keep, so every run starts from the same clean-base.\n\n" +
			"The guest converges with the dotf release the ref's versions.conf pins, not a\n" +
			"build of this tree, unless --dotf names a Windows Arm64 source build to use\n" +
			"instead (install.ps1 keeps a dotf that reports \"dev\"): that tests a change to\n" +
			"dotf before a release carries it. Only committed content and that file reach\n" +
			"the guest: no host directory is shared with it, and no credential is passed in.",
		Example: "  dotf vm windows --ref feat/my-branch\n  dotf vm windows --keep\n" +
			"  GOOS=windows GOARCH=arm64 go build -o /tmp/dotf.exe ./cmd/dotf\n" +
			"  dotf vm windows --ref feat/my-branch --dotf /tmp/dotf.exe",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dotf != "" {
				if fi, err := os.Stat(dotf); err != nil || !fi.Mode().IsRegular() {
					return withExitCode(2, fmt.Errorf("--dotf %s: not a file", dotf))
				}
			}
			host, firmware, _, layout, err := windowsGuest()
			if err != nil {
				return err
			}
			src, err := archiveRef(ref)
			if err != nil {
				return withExitCode(2, err)
			}
			defer func() { _ = os.Remove(src) }()
			return vm.Run(host, vm.RunOptions{
				Machine: vm.Machine{Layout: layout, Firmware: firmware, CPUs: cpus, MemoryMB: memory},
				Source:  src,
				Dotf:    dotf,
				Keep:    keep,
				Agent:   10 * time.Minute,
				Step:    time.Hour,
				Poll:    5 * time.Second,
				Out:     cmd.OutOrStdout(),
			})
		},
	}
	w.Flags().StringVar(&ref, "ref", "HEAD", "committed branch, tag or commit to converge")
	w.Flags().BoolVar(&keep, "keep", false, "leave the guest running for inspection")
	w.Flags().StringVar(&dotf, "dotf", "", "a dotf.exe source build (windows/arm64) to converge with instead of the pinned release")
	w.Flags().IntVar(&cpus, "cpus", 4, "virtual CPUs")
	w.Flags().IntVar(&memory, "memory", 8192, "memory in MiB")
	w.AddCommand(newVMWindowsBuildCmd(), newVMWindowsStopCmd())
	return w
}

func newVMWindowsStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "stop",
		Short:        "Stop a guest that `dotf vm windows --keep` left running",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			host, _, _, layout, err := windowsGuest()
			if err != nil {
				return err
			}
			return vm.Stop(host, layout, cmd.OutOrStdout())
		},
	}
}

// windowsGuest resolves QEMU and the guest's state directory on the one host
// the guest runs on.
func windowsGuest() (host vm.ExecHost, firmware, vars string, l vm.Layout, err error) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return host, "", "", l, withExitCode(2, errors.New("the Windows guest runs on a macOS Arm64 host only"))
	}
	if host, firmware, vars, err = vm.FindQEMU(); err != nil {
		return host, "", "", l, withExitCode(2, err)
	}
	state, err := env.StateDir()
	if err != nil {
		return host, "", "", l, err
	}
	return host, firmware, vars, vm.Layout{Dir: filepath.Join(state, "vm", "windows")}, nil
}

// archiveRef writes a zip of ref's tree, under dotfiles/, from the repository
// of the working directory: committed content only, never the working tree.
func archiveRef(ref string) (string, error) {
	if strings.HasPrefix(ref, "-") {
		return "", fmt.Errorf("ref %q is not a branch, tag or commit name", ref)
	}
	// From a subdirectory, git archive takes only that subdirectory's tree.
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", errors.New("run from the dotfiles repository: it is the archive's source")
	}
	git := func(args ...string) ([]byte, error) {
		c := exec.Command("git", args...)
		c.Dir = strings.TrimSpace(string(top))
		return c.CombinedOutput()
	}
	if out, err := git("cat-file", "-e", ref+":install.ps1"); err != nil {
		return "", fmt.Errorf("%s has no install.ps1 (run from the dotfiles repository, with a committed ref): %s", ref, strings.TrimSpace(string(out)))
	}
	f, err := os.CreateTemp("", "dotf-vm-*.zip")
	if err != nil {
		return "", err
	}
	_ = f.Close()
	if out, err := git("archive", "--format=zip", "--prefix=dotfiles/", "-o", f.Name(), ref); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("git archive %s: %w: %s", ref, err, strings.TrimSpace(string(out)))
	}
	return f.Name(), nil
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
			"link, with --iso. Either is verified against the ISO's pinned SHA256 before\n" +
			"anything boots from it. virtio-win is downloaded and verified the\n" +
			"same way. The guest has no TPM and no credential: a local account with no\n" +
			"password, unactivated.",
		Example:      "  dotf vm windows build --iso ~/Downloads/Windows11_Client_arm64_en-us_26300_9457.iso",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			host, firmware, vars, layout, err := windowsGuest()
			if err != nil {
				return err
			}
			return vm.Build(host, vm.BuildOptions{
				Machine: vm.Machine{
					Layout:   layout,
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

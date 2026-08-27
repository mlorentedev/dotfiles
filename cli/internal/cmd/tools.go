package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
	"github.com/spf13/cobra"
)

// newToolsCmd is the `dotf tools` noun: the declarative cross-OS package catalog
// (packages.json) consumed by dotf instead of per-OS imperative install blocks
// (CLI-029, piloting the ADR-021 / CLI-028 convergence). PR-A ships `list`; the
// installer (`install`) lands in PR-B.
func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Declarative cross-OS package catalog (packages.json)",
		Long: "tools reads packages.json — the tool/install list as data — so a single\n" +
			"catalog feeds every OS instead of duplicated install blocks in setup-linux.sh\n" +
			"and setup-windows.ps1 (CLI-029, piloting ADR-021/CLI-028 with sops).",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newToolsListCmd())
	cmd.AddCommand(newToolsInstallCmd())
	cmd.AddCommand(newToolsVersionCmd())
	return cmd
}

func newToolsInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [name]",
		Short: "Download, verify (sha256), and install catalog tools to ~/.local/bin",
		Long: "install downloads each catalog tool's pinned release binary, verifies its\n" +
			"sha256 against the release checksums, and places it in ~/.local/bin. It is\n" +
			"idempotent: a tool already at or above its pin is skipped, a below-pin one is\n" +
			"upgraded (never downgraded). With no [name] it installs every catalog tool;\n" +
			"with a name it installs just that one.",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := filepath.Join(env.DotfilesDir(env.Home()), "packages.json")
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("packages.json not found at %s — set DOTFILES_DIR or run from the repo", path)
			}
			cat, err := tools.Load(path)
			if err != nil {
				return err
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			in := &tools.Installer{
				Dest: filepath.Join(env.Home(), ".local", "bin"),
				Out:  cmd.OutOrStdout(),
			}
			return runToolsInstall(in, cat, name, cmd.ErrOrStderr())
		},
	}
}

// runToolsInstall selects the requested tools and installs them. Split from the
// cobra wiring so it can be driven in tests with an Installer whose Fetch /
// CurrentVersion seams are faked (no network).
func runToolsInstall(in *tools.Installer, cat tools.Catalog, name string, errOut io.Writer) error {
	selected, err := selectTools(cat, name)
	if err != nil {
		return err
	}
	return installAll(in, selected, errOut)
}

// selectTools returns every tool when name is empty, or the single named tool,
// erroring when the catalog does not list it.
func selectTools(cat tools.Catalog, name string) ([]tools.Tool, error) {
	if name == "" {
		return cat.Tools, nil
	}
	for _, t := range cat.Tools {
		if t.Name == name {
			return []tools.Tool{t}, nil
		}
	}
	return nil, fmt.Errorf("%q is not in the catalog (packages.json)", name)
}

// installAll installs each selected tool best-effort: a failure on one is logged
// to errOut and the run continues, so a single offline or 404'd tool never
// blocks the rest. It returns a non-nil error when ANY tool failed — a human
// running `dotf tools install` sees exit 1, while setup (which wraps this
// best-effort) only warns. This is the multi-tool error-handling policy.
func installAll(in *tools.Installer, selected []tools.Tool, errOut io.Writer) error {
	var failed []string
	for _, t := range selected {
		if _, err := in.Install(t); err != nil {
			_, _ = fmt.Fprintf(errOut, "warning: %v\n", err)
			failed = append(failed, t.Name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to install: %s", strings.Join(failed, ", "))
	}
	return nil
}

func newToolsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List catalog tools with the release asset resolved for this OS/arch",
		Long: "list parses packages.json and prints each tool with the release-asset\n" +
			"filename resolved for the current GOOS/GOARCH. A blank asset means the\n" +
			"tool declares no build for this platform.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := filepath.Join(env.DotfilesDir(env.Home()), "packages.json")
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("packages.json not found at %s — set DOTFILES_DIR or run from the repo", path)
			}
			cat, err := tools.Load(path)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tVERSION\tPROFILE\tASSET ("+runtime.GOOS+"/"+runtime.GOARCH+")")
			for _, t := range cat.Tools {
				asset := t.AssetName(runtime.GOOS, runtime.GOARCH)
				switch {
				case t.Source.Type == "npm":
					asset = "npm:" + t.Source.Package // npm tools are platform-agnostic
				case asset == "":
					asset = "(no build for this platform)"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Name, t.Version, t.Profile, asset)
			}
			return w.Flush()
		},
	}
}

// toolsVersionRunner is the exec seam for `dotf tools version`; tests inject a fake.
var toolsVersionRunner tools.Runner = tools.ExecRunner

func newToolsVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version <name>",
		Short: "Print the semver a tool on PATH reports, or exit 1",
		Long: "version runs `<name> --version` and prints the first semver in its output —\n" +
			"the one extraction every caller shares (ADR-036). The setup scripts used to\n" +
			"parse \"last token of the first line\" in seven places, and on the Windows\n" +
			"work box that accepted `locked.` as opencode's version (AI-034, #1294).\n" +
			"Exits 1 with nothing on stdout when the tool is absent or prints no version,\n" +
			"so a shell caller can test either.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := tools.ProbeVersion(args[0], toolsVersionRunner)
			if v == "" {
				return fmt.Errorf("%s: no version found on PATH", args[0])
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), v)
			return nil
		},
	}
}

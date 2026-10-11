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
			"catalog feeds every OS instead of one install block per OS.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newToolsListCmd())
	cmd.AddCommand(newToolsInstallCmd())
	cmd.AddCommand(newToolsVersionCmd())
	cmd.AddCommand(newToolsSyncCmd())
	return cmd
}

func newToolsInstallCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "install [name]",
		Short: "Download, verify (sha256), and install catalog tools to ~/.local/bin",
		Long: "install downloads each catalog tool's pinned release binary, verifies its\n" +
			"sha256 against the release checksums, and places it in ~/.local/bin. It is\n" +
			"idempotent: a tool already at or above its pin is skipped, a below-pin one is\n" +
			"upgraded (never downgraded). With no [name] it installs every catalog tool;\n" +
			"with a name it installs just that one.\n\n" +
			"--dry-run prints the action install would take for each tool and changes\n" +
			"nothing. It exits non-zero when install would refuse a tool (a release\n" +
			"with no checksums file, a package source naming no package), as install\n" +
			"itself would.",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cat, err := loadToolsCatalog()
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
			if dryRun {
				return planToolsInstall(in, cat, name, cmd.OutOrStdout())
			}
			return runToolsInstall(in, cat, name, cmd.ErrOrStderr())
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print what install would do, and change nothing")
	return c
}

// loadToolsCatalog reads packages.json from the checkout, else the deploy mirror
// (env.ResolveCatalogPath), the file doctor reads too.
func loadToolsCatalog() (tools.Catalog, error) {
	path := env.ResolveCatalogPath()
	if _, err := os.Stat(path); err != nil {
		return tools.Catalog{}, fmt.Errorf("packages.json not found in the checkout or at %s — run from the repo or set DOTFILES_DIR", path)
	}
	return tools.Load(path)
}

// planToolsInstall prints one row per selected tool: what install would do. A
// tool install would refuse fails the dry run too, after every row is printed.
func planToolsInstall(in *tools.Installer, cat tools.Catalog, name string, out io.Writer) error {
	selected, err := selectTools(cat, name)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tINSTALLED\tPIN\tACTION")
	var refused []string
	for _, t := range selected {
		p := in.Plan(t)
		if p.Action == tools.PlanRefused {
			refused = append(refused, p.Name)
		}
		installed := p.Installed
		if installed == "" {
			installed = "absent"
		}
		pin := p.Pin
		if pin == "" {
			pin = "-" // a system package has no pin, and an empty cell would shift the row
		}
		action := string(p.Action)
		if p.Note != "" {
			action += " (" + p.Note + ")"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, installed, pin, action)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if len(refused) > 0 {
		return fmt.Errorf("install would refuse: %s", strings.Join(refused, ", "))
	}
	return nil
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
	// One command for every package that waited on a sudo password, so a fresh
	// machine is one paste, not one per package (#2308).
	if cmd := in.SudoDeferred(); cmd != "" {
		_, _ = fmt.Fprintf(in.Out, "\nInstall the packages that need sudo with one command:\n  %s\n", cmd)
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
			cat, err := loadToolsCatalog()
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tVERSION\tPROFILE\tASSET ("+runtime.GOOS+"/"+runtime.GOARCH+")")
			for _, t := range cat.Tools {
				asset := listAsset(t, runtime.GOOS, runtime.GOARCH)
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Name, t.Version, t.Profile, asset)
			}
			return w.Flush()
		},
	}
}

// listAsset is the ASSET cell of `dotf tools list`: what this OS/arch would
// install, or why nothing would.
func listAsset(t tools.Tool, goos, goarch string) string {
	switch {
	case !t.SupportsOS(goos):
		return "(not in the catalog on this platform)"
	case t.Source.Type == "npm":
		return "npm:" + t.Source.Package
	case t.Source.Type == "uv-tool":
		return "uv:" + t.Source.Package
	case t.Source.Type == "system":
		manager, pkg := t.Source.SystemPackage(goos)
		return manager + ":" + pkg
	}
	if asset := t.AssetName(goos, goarch); asset != "" {
		return asset
	}
	return "(no build for this platform)"
}

// toolsVersionRunner is the exec seam for `dotf tools version`; tests inject a fake.
var toolsVersionRunner tools.Runner = tools.ExecRunner

func newToolsVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version <name>",
		Short: "Print the semver a tool on PATH reports, or exit 1",
		Long: "version runs `<name> --version` and prints the first semver in its output —\n" +
			"the one extraction every caller shares, so no script parses version output\n" +
			"itself (a tool that prints `locked.` is not read as a version).\n" +
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

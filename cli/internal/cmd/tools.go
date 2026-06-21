package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	return cmd
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
			fmt.Fprintln(w, "NAME\tVERSION\tPROFILE\tASSET ("+runtime.GOOS+"/"+runtime.GOARCH+")")
			for _, t := range cat.Tools {
				asset := t.AssetName(runtime.GOOS, runtime.GOARCH)
				if asset == "" {
					asset = "(no build for this platform)"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Name, t.Version, t.Profile, asset)
			}
			return w.Flush()
		},
	}
}

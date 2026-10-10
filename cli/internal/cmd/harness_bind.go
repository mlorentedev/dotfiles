package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessBindCmd is the third manifest mode beside `deploy` and `presence`,
// and the only writer of the `hooks` key in a harness settings file.
//
// WHY IT REPLACES SHELL RATHER THAN JOINING IT. `merge_claude_settings()` in
// setup-linux.sh does `.hooks.SessionStart = $tmpl.hooks.SessionStart` — an
// ASSIGNMENT, not a merge. Simulated against a copy of the deployed file on
// 2026-08-27: SessionStart went from 2 groups to 1, deleting a live third-party
// hook. setup-windows.ps1 carries the identical defect at its own lines. Adding
// a second writer beside them would not have fixed it; the assignment had to go,
// and then the file needs exactly one owner. That is this repository's most
// repeated lesson, now on its seventh surface: THE WRITER TOUCHES ONLY WHAT IT
// OWNS, and ownership is by marker, never by position.
//
// Emission is data-driven from `agents.bind` so adding a harness is a manifest
// edit. A target declaring `emit: false` is skipped visibly rather than
// forgotten, and one naming `requires_command` is skipped when that binary is
// absent — an uninstalled harness is not a failure.
func newHarnessBindCmd() *cobra.Command {
	var (
		harnessName string
		repoRoot    string
		homeDir     string
		dotfPath    string
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "bind",
		Short: "Emit this repository's hooks into each harness's settings file",
		Long: `bind writes the hooks declared in harness/manifest.json's ` + "`agents.bind`" + ` into
each harness's own settings file, merging by marker so a third party's entries
are never reordered, rewritten or removed.

It is the only writer of the ` + "`hooks`" + ` key. A setup script that also assigns to
it will delete whatever it does not know about, which is what this replaced.

Re-running is idempotent: an unchanged file is not rewritten, and a CHANGED
command replaces our entry in place rather than appending a second one.

A target whose format is hooks-json (agy) is a document of NAMED hooks. It is
owned by name rather than by marker, and only that one name is ever written.
A target can also retire a hook it used to emit into another file, so moving a
hook does not leave the old entry firing where it no longer parses.`,
		Example: `  dotf harness bind
  dotf harness bind --harness claude
  dotf harness bind --dry-run`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, home, binary, err := resolveBindInputs(repoRoot, homeDir, dotfPath)
			if err != nil {
				return err
			}
			targets, err := harness.LoadBindTargets(root)
			if err != nil {
				return err
			}
			return bindTargets(cmd.OutOrStdout(), targets, harnessName, home, binary, dryRun)
		},
	}

	cmd.Flags().StringVar(&harnessName, "harness", "", "bind only this harness (default: every declared target)")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "root containing harness/manifest.json")
	cmd.Flags().StringVar(&homeDir, "home", "", "home directory the settings files live under (default: the user's)")
	cmd.Flags().StringVar(&dotfPath, "dotf-path", "", "absolute path emitted into the hook commands (default: resolved)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would change without writing")
	return cmd
}

// resolveBindInputs settles the three ambient inputs — where the manifest is,
// whose home to write under, and what binary path the hooks will name — so RunE
// stays a sequence of named steps rather than a wall of defaulting.
//
// The dotf path stays raw here because each harness owns its command parser.
// bindTargets renders the token once it knows which harness will execute it.
func resolveBindInputs(repoRoot, homeDir, dotfPath string) (root, home, binary string, err error) {
	root = repoRoot
	if root == "" {
		root = env.ResolveHarnessRoot()
	}
	home = homeDir
	if home == "" {
		h, herr := os.UserHomeDir()
		if herr != nil {
			return "", "", "", fmt.Errorf("resolve home: %w", herr)
		}
		home = h
	}
	binary = dotfPath
	if binary == "" {
		binary = harness.ResolveDotfPath(home)
	}
	return root, home, binary, nil
}

// bindTargets binds every selected target and reports one stable status tag
// per line: `skip`, `ok`, `would update` or `bind`. Tests assert those tags, not
// the prose after them.
func bindTargets(out io.Writer, targets []harness.BindTarget, harnessName, home, binary string, dryRun bool) error {
	outcomes, err := harness.Bind(targets, harness.BindOptions{
		Home: home, Binary: binary, GOOS: runtime.GOOS, Only: harnessName, DryRun: dryRun,
		Has: func(name string) bool { _, err := exec.LookPath(name); return err == nil },
	})
	for _, o := range outcomes {
		switch {
		case o.Skip != "":
			_, _ = fmt.Fprintf(out, "skip %s: %s\n", o.Agent, o.Skip)
			continue
		case !o.Changed:
			// Never an error: "already current" is the steady state the
			// idempotence doctrine asks for (changed=0 on a re-run).
			_, _ = fmt.Fprintf(out, "ok   %s: hooks already current\n", o.Agent)
		case dryRun:
			_, _ = fmt.Fprintf(out, "would update %s: %s\n", o.Agent, o.File)
		default:
			_, _ = fmt.Fprintf(out, "bind %s: %s\n", o.Agent, o.File)
		}
		for _, r := range o.Retired {
			verb := "retire"
			if dryRun {
				verb = "would retire"
			}
			_, _ = fmt.Fprintf(out, "%s %s: %s (%s on %s)\n", verb, o.Agent, r.File, r.ID, r.Event)
		}
	}
	return err
}

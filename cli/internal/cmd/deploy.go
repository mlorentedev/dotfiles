package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// The `dotf deploy` report's columns: a verb, the entry, then what the line is
// about. Every step of a bare deploy (the manifest entries, Orca's hooks, Claude
// Code's servers and plugins) writes through deployRow, so the third column
// starts at one place on every line (#1664). A name longer than the column
// would push its own line out; TestDeployRow_EveryShippedNameFits keeps the
// manifest inside it.
const (
	deployVerbWidth = len("would fix mode") // the longest verb
	deployNameWidth = 16                    // the longest manifest name
)

func deployRow(w io.Writer, verb, name, format string, args ...any) {
	_, _ = fmt.Fprintf(w, "%-*s %-*s %s\n", deployVerbWidth, verb, deployNameWidth, name, fmt.Sprintf(format, args...))
}

// deployRenderer is the render seam. Production calls the SAME secrets.Render
// the setup scripts called; a test injects a no-op. Deliberately a call and not
// a reimplementation — a second substitution implementation is the defect this
// command removes.
var deployRenderer = func(path string) error {
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	_, err = secrets.Render(path, reg, secretLoader(), env.Home())
	return err
}

// strictDeployRenderer is deployRenderer for a run nobody watches, converge's
// configs step: a placeholder the secret store could not resolve (locked or
// unreachable, as opposed to a secret this machine never holds) is
// deploy.ErrRenderIncomplete, so the entry keeps its installed file instead of
// trading a resolved value for its placeholder.
func strictDeployRenderer(path string) error {
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	res, err := secrets.Render(path, reg, secretLoader(), env.Home())
	if err != nil {
		return err
	}
	if len(res.Unresolved) > 0 {
		vars := make([]string, 0, len(res.Unresolved))
		for _, u := range res.Unresolved {
			vars = append(vars, u.Var)
		}
		return fmt.Errorf("%w: %s", deploy.ErrRenderIncomplete, strings.Join(vars, ", "))
	}
	return nil
}

// deployCommandAvailable is the PATH seam behind a manifest entry's `requires`
// (AI-039, #1322): production asks the real PATH, a test answers for it.
var deployCommandAvailable = func(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// newDeployCmd installs agent configs from the checkout to their deployed
// locations (CLI-039, #1023).
//
// The behaviour is one implementation for every OS. It used to be two — one in
// setup-linux.sh and one in setup-windows.ps1, per config — which is the twin
// shape ADR-020's strangler-fig rule says to collapse the next time it is
// touched. The setups now call this and get shorter.
func newDeployCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "deploy [name]",
		Short: "Install agent configs from the checkout to their deployed locations",
		Long: "deploy installs the agent configs declared in ai/deploy.json: it stages the\n" +
			"source, substitutes {env:VAR} placeholders through `dotf secrets render` when\n" +
			"the config asks for it, compares against the installed copy, and installs\n" +
			"atomically with the declared permissions.\n\n" +
			"A config already in sync is reported and NOT rewritten, so a setup run does\n" +
			"not churn mtimes and \"did this change?\" stays answerable.\n\n" +
			"An entry with strategy \"merge\" writes only its top-level keys into the\n" +
			"installed JSON and preserves every other key (a file the reading tool also\n" +
			"writes, such as Copilot's settings.json). An entry that `requires` a command\n" +
			"is skipped, and says so, when that command is not on PATH.\n\n" +
			"A bare deploy also re-tunes Orca's generated Copilot hooks (`dotf orca\n" +
			"tune-hooks`), which Orca reverts on every install, registers the MCP\n" +
			"servers in mcp-servers.json with Claude Code, and installs the Claude Code\n" +
			"plugins in ai/claude/plugins.json that the box lacks.\n\n" +
			"  dotf deploy              # every declared config\n" +
			"  dotf deploy pi           # one\n" +
			"  dotf deploy --dry-run    # report what would change, touch nothing",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot := env.RepoDir()
			if repoRoot == "" {
				return fmt.Errorf("cannot locate the dotfiles checkout — set DOTFILES_REPO_DIR or run from inside it")
			}
			manifestPath := filepath.Join(repoRoot, filepath.FromSlash(deploy.ManifestRel))
			raw, err := os.ReadFile(manifestPath) //nolint:gosec // repo-relative, fixed name
			if err != nil {
				return fmt.Errorf("reading %s: %w", deploy.ManifestRel, err)
			}
			man, err := deploy.ParseManifest(raw)
			if err != nil {
				return err
			}

			targets := man.Configs
			if len(args) == 1 {
				c := man.Lookup(args[0])
				if c == nil {
					return fmt.Errorf("%w: %q (declared: %s)", deploy.ErrNoSuchConfig, args[0], names(man))
				}
				targets = []deploy.Config{*c}
			}

			w := cmd.OutOrStdout()
			res, err := deploy.Run(man, targets, deploy.RunOptions{
				RepoRoot:  repoRoot,
				Home:      env.Home(),
				GOOS:      runtime.GOOS,
				Resolve:   env.ResolvePath,
				Render:    deployRenderer,
				Available: deployCommandAvailable,
				DryRun:    dryRun,
			})
			printDeploySteps(w, res, dryRun)
			if err != nil {
				return err
			}
			// A bare deploy converges everything the setups own, including the
			// Orca hooks Orca rewrites on every install (CLI-093, #1953) and the
			// Claude Code MCP servers and plugins (CLI-063, #1339). The two
			// Claude steps are independent, so one failing still runs the other.
			if len(args) == 0 {
				if err := deployOrcaHooks(w, env.Home(), dryRun); err != nil {
					return err
				}
				return errors.Join(
					deployClaudeMCP(w, repoRoot, env.Home(), dryRun),
					deployClaudePlugins(w, repoRoot, env.Home(), dryRun),
				)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report what would change without writing anything")
	return c
}

// printDeploySteps writes one row per entry, then one per directory the run
// narrowed because it holds a private file this run deployed (#2161). Only the
// entries this run deployed count, so `dotf deploy pi` never narrows ~/.ssh as
// a side effect.
func printDeploySteps(w io.Writer, res deploy.RunResult, dryRun bool) {
	for _, s := range res.Steps {
		if s.Skipped != "" {
			deployRow(w, "skipped", s.Name, "(%s)", s.Skipped)
			continue
		}
		o := s.Outcome
		switch {
		case !o.Changed:
			deployRow(w, "in sync", o.Name, "%s", o.Dst)
		case o.ModeFixed && dryRun:
			deployRow(w, "would fix mode", o.Name, "%s", o.Dst)
		case o.ModeFixed:
			// Content was in sync; only the declared mode was missing on
			// the file (CLI-055: an inherited ACL on a 0600).
			deployRow(w, "mode fixed", o.Name, "%s", o.Dst)
		case dryRun:
			deployRow(w, "would deploy", o.Name, "%s", o.Dst)
		default:
			deployRow(w, "deployed", o.Name, "%s", o.Dst)
		}
		if o.BackedUp != "" {
			deployRow(w, "", "", "kept the previous file at %s", o.BackedUp)
		}
	}
	verb := "tightened"
	if dryRun {
		verb = "would tighten"
	}
	for _, d := range res.Tightened {
		deployRow(w, verb, "", "%s from %04o to %04o", d.Dir, d.From, d.To)
	}
}

func names(m *deploy.Manifest) string {
	out := make([]string, 0, len(m.Configs))
	for _, c := range m.Configs {
		out = append(out, c.Name)
	}
	if len(out) == 0 {
		return "none"
	}
	return fmt.Sprint(out)
}

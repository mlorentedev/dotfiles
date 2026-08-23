package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/agent"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newAgentCmd is the RUN side of the orchestration stack, and it takes its own
// noun for a reason ADR-032 §2 states: `dotf harness` is the compile side —
// idempotent and offline — while this spawns processes and consumes quota.
// Folding them into one verb would put those two under one permission and one
// mental model.
func newAgentCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "agent",
		Short: "Run work on another model, pool or harness",
		Long: `agent is the execution layer over harness/model-map.json.

Where ` + "`dotf harness`" + ` renders definitions, ` + "`dotf agent`" + ` dispatches work: it walks
the ordered fallback the map declares for a tier and reports which pool and model
actually answered.`,
		SilenceUsage: true,
		RunE:         func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	c.AddCommand(newAgentRunCmd())
	return c
}

func newAgentRunCmd() *cobra.Command {
	var (
		role, task, tier, cwd, backendName, repoRoot string
		timeout                                      time.Duration
	)

	c := &cobra.Command{
		Use:   "run",
		Short: "Dispatch one task to the first pool in a tier's chain that can serve it",
		Long: `run dispatches ONE task synchronously and writes ONE JSON object to stdout.

Every log line goes to stderr, so stdout is safe to pipe into a parser:

    dotf agent run --role reviewer --task "..." --tier mid --backend dry-run --timeout 2m | jq .status

The record carries status, tier, pool, model, exit, duration_ms, output and the
attempts the walk made. Status is the fine-grained truth and the process exit
code is the coarse class, mirroring ` + "`hive delegate`" + ` so one vocabulary spans the
seam: 0 answered, 1 the task failed, 3 no pool could serve it.

A pool reporting itself unavailable advances to the next entry in the chain; a
task that ran and failed does NOT — retrying a real failure on another model
turns a bad answer into a silent second opinion. The top tier never degrades: if
its pool cannot serve, this escalates and exits non-zero rather than quietly
answering from a weaker model.`,
		Example: `  dotf agent run --role reviewer --task "review this diff" --tier mid --backend dry-run --timeout 2m
  dotf agent run --role architect --task "decide" --tier top --backend dry-run --timeout 5m`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if role == "" || task == "" {
				return fmt.Errorf("--role and --task are both required: a dispatch with no task has nothing to run, " +
					"and one with no role has no way to be scheduled")
			}
			if tier == "" {
				return fmt.Errorf("--tier is required (top|mid|low): the chain to walk is declared per tier in %s, "+
					"and guessing one would route work to a model nobody chose", harness.ModelMapFile)
			}
			if timeout <= 0 {
				return fmt.Errorf("--timeout is required and must be positive: ADR-032 §2 makes a bounded dispatch " +
					"part of the contract, and a backend that cannot be bounded is not eligible")
			}
			backend, err := resolveBackend(backendName)
			if err != nil {
				return err
			}

			root := repoRoot
			if root == "" {
				root = env.ResolveHarnessRoot()
			}
			m, err := harness.LoadModelMap(root)
			if err != nil {
				return err
			}
			chain, err := harness.ResolveChain(m, tier)
			if err != nil {
				return err
			}

			rec := agent.Dispatch(cmd.Context(), agent.Options{
				Tier: tier, Role: role, Task: task, Cwd: cwd,
				Chain: chain, Timeout: timeout,
			}, backend)

			// The record reaches stdout whatever the outcome: its consumer is a
			// dispatcher, and an error path that emits nothing forces that
			// consumer to parse stderr prose to learn what happened.
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetEscapeHTML(false)
			if err := enc.Encode(rec); err != nil {
				return err
			}
			if code := agent.ExitCode(rec.Status); code != 0 {
				return withExitCode(code, summarise(rec))
			}
			return nil
		},
	}

	c.Flags().StringVar(&role, "role", "", "role to run the task as (required)")
	c.Flags().StringVar(&task, "task", "", "the task text handed to the model (required)")
	c.Flags().StringVar(&tier, "tier", "", "neutral tier whose chain is walked: top|mid|low (required)")
	c.Flags().StringVar(&cwd, "cwd", "", "working copy the task runs against (default: the current directory)")
	c.Flags().DurationVar(&timeout, "timeout", 0, "per-dispatch deadline, e.g. 90s or 5m (required)")
	c.Flags().StringVar(&backendName, "backend", "", "backend to dispatch through (required until probing lands)")
	c.Flags().StringVar(&repoRoot, "repo-root", "",
		"root containing harness/model-map.json (default: the checkout, else DOTFILES_DIR)")
	return c
}

// resolveBackend picks the implementation behind the seam.
//
// Probing (ADR-032 §7) selects it automatically once a real backend exists;
// until then the flag is required, and its absence is a loud refusal rather
// than a default. Defaulting to dry-run would be the worst of the options: a
// dispatch that silently ran nothing and exited 0.
func resolveBackend(name string) (agent.Backend, error) {
	switch name {
	case "dry-run":
		return agent.DryRun{}, nil
	case "":
		return nil, fmt.Errorf("--backend is required: the only backend implemented today is `dry-run`, " +
			"which resolves the route without dispatching. The subprocess and hive backends, and the probe " +
			"that would pick one for you, are not built yet")
	default:
		return nil, fmt.Errorf("unknown backend %q: the only backend implemented today is `dry-run`", name)
	}
}

// summarise is the human line on stderr. The JSON already carries the truth; a
// person reading a failing &&-chain should not have to pipe it through a parser
// to learn which pool declined.
func summarise(rec agent.Record) error {
	switch rec.Status {
	case agent.StatusChainExhausted:
		return fmt.Errorf("no pool in the %s chain could serve this dispatch (%d attempted)", rec.Tier, len(rec.Attempts))
	case agent.StatusEscalated:
		return fmt.Errorf("the %s tier could not be served and does not fall back: escalate rather than "+
			"re-run at a lower tier", rec.Tier)
	default:
		return fmt.Errorf("task failed on %s:%s (exit %d)", rec.Pool, rec.Model, rec.Exit)
	}
}

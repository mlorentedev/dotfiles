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
		semaphoreDir                                 string
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
answering from a weaker model.

CONCURRENCY. Dispatches are bounded per pool at the map's declared concurrency
minus its reserve_interactive, and the bound covers dispatches made through this
command and nothing else. The pools are shared with consumers it cannot see — a
hand-run qq, a pi TUI turn, hive embeddings, CI — each of which takes capacity
no semaphore here counts. So the guarantee is exactly this and no wider:

    dotf alone will never be the cause of exhaustion.

The reserve makes starvation of your interactive session unlikely; it cannot
make it impossible, and a dispatch that hits a rate limit is reported as the
pool being unavailable so the chain moves on rather than retrying into the wall.

DENIAL. Which pools a machine may use is declared in its machine.json, and is
evaluated here at dispatch time rather than baked in at config time. A machine
that has not declared an identity is refused outright: an undeclared machine
denies every non-local pool, because defaulting the unknown case to "allowed"
fails silently and in the direction nobody notices.`,
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
			backend, err := agent.ResolveRouter(agent.DefaultBackends(), backendName)
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

			policy, err := env.LoadMachinePolicy(env.MachinePath(env.Home()))
			if err != nil {
				return err
			}
			if !policy.Identified {
				return fmt.Errorf(unidentifiedMachine, env.MachinePath(env.Home()))
			}
			if err := policy.ValidateDeny(declaredPools(m)); err != nil {
				return err
			}

			semDir := semaphoreDir
			if semDir == "" {
				// Not derived from --repo-root: the budget is a property of the
				// MACHINE, and two checkouts of this repo draw on one NaN
				// subscription. A per-checkout counter would bound neither.
				if semDir, err = agent.DefaultSemaphoreDir(); err != nil {
					return err
				}
			}

			rec := agent.Dispatch(cmd.Context(), agent.Options{
				Tier: tier, Role: role, Task: task, Cwd: cwd,
				Chain: chain, Timeout: timeout,
				Denied:    policy.Denies,
				Semaphore: agent.NewSemaphore(semDir),
				Capacity:  declaredCapacity(m),
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
	c.Flags().StringVar(&backendName, "backend", "",
		"force one backend: subprocess|hive|dry-run (default: probe, preferring subprocess where the harness binary is present)")
	c.Flags().StringVar(&semaphoreDir, "semaphore-dir", "",
		"directory holding per-pool slot state (default: the machine's runtime dir; the budget is a machine property, not a checkout's)")
	c.Flags().StringVar(&repoRoot, "repo-root", "",
		"root containing harness/model-map.json (default: the checkout, else DOTFILES_DIR)")
	return c
}

// declaredCapacity answers how many concurrent dispatches a pool admits, and
// whether the map declares a number at all.
//
// The reserve is subtracted here rather than in the semaphore: `concurrency`
// and `reserve_interactive` are both the map's to state, and what the semaphore
// enforces is the difference — with nan's 5 and a reserve of 2, a fan-out
// claims at most 3, leaving the TUI room it never has to ask for.
//
// A pool that declares nothing returns declared=false, not zero. Zero would
// refuse every dispatch to a seat-based pool whose concurrency is a fleet
// property the map cannot honestly state.
func declaredCapacity(m map[string]any) func(string) (int, bool) {
	return func(pool string) (int, bool) {
		b, err := harness.DeclaredBudget(m, pool)
		if err != nil || !b.ConcurrencyDeclared {
			return 0, false
		}
		capacity := b.Concurrency - b.ReserveInteractive
		if capacity < 1 {
			// A reserve that swallows the pool is a declaration error, not a
			// standing refusal: floor at one so the map cannot silently disable
			// dispatch, and let the reserve be wrong loudly elsewhere.
			capacity = 1
		}
		return capacity, true
	}
}

// The backend is no longer chosen by a switch: `agent.DefaultBackends()` is the
// probe order and `agent.ResolveRouter` validates the --backend override
// against it. Routing is per chain ENTRY, because `chains.mid` mixes pools and
// hive serves only one of them.

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

// unidentifiedMachine is the refusal a machine gets when it has not said who it
// is. ADR-032 §8 makes this the required direction rather than a convenience:
// a rebuilt corporate machine that has not restored machine.json probes
// successfully for personal pools, so defaulting to "allowed" would fail
// silently and in the wrong direction.
//
// It names the file, the key and a working value, because a fail-closed
// refusal whose remedy the operator has to go and look up is a fail-closed
// refusal people route around.
const unidentifiedMachine = `this machine has not declared an identity, so every non-local pool is denied (ADR-032 §8)

Declare one in %s:

    {
      "machine": { "id": "<a-name-for-this-machine>" },
      "pools":   { "deny": [] }
    }

Keep any existing "paths" block; this adds to the same file. The deny list is
where a machine forbids pools it must not use — an empty list denies nothing.
The default is denial rather than permission on purpose: a rebuilt machine that
has not restored this file probes successfully for every pool, and allowing them
would be a silent failure in the direction that cannot be noticed`

// declaredPools is the set of pool names the routing map declares — the
// allow-list `pools.deny` entries are checked against, so a typo is refused
// rather than silently leaving the pool it meant to forbid allowed.
func declaredPools(m map[string]any) map[string]bool {
	out := map[string]bool{}
	pools, ok := m["pools"].(map[string]any)
	if !ok {
		return out
	}
	for name := range pools {
		out[name] = true
	}
	return out
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/agent"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newAgentAutoCmd closes the gap HARNESS-120 measured: `dotf harness suggest`
// derives the persona a prompt implies and stops, `dotf agent run` dispatches
// but only if a human supplies the role and the tier by hand — and of 11 745
// gate decisions, 11 168 (95.1%) are `no-role`. The apparatus governs a lane
// almost nothing travels, because entering it costs a deliberate act nobody
// performs. This makes the act one command.
//
// It composes five things that already existed and two that did not: the join
// (ResolveRoles), the narrowing (ResolveOne), the tier the persona declares for
// itself (ResolveTierForPersona — the first reader of Persona.Model), the
// preamble that makes the persona TRAVEL (PersonaPreamble), and the walk
// (agent.Dispatch).
//
// Dispatch stays PROMPTED rather than automatic, which is proposal.md's recorded
// decision and issue #1537's first acceptance criterion. The UserPromptSubmit
// hook cannot be the dispatcher: exit 2 there erases the prompt, so the hook
// must fail open, and a dispatcher that cannot refuse is a dispatcher that
// cannot be trusted with the decision. The hook keeps suggesting; this runs.
func newAgentAutoCmd() *cobra.Command {
	var (
		role, task, tier, cwd, backendName, repoRoot string
		semaphoreDir                                 string
		timeout                                      time.Duration
	)

	c := &cobra.Command{
		Use:   "auto",
		Short: "Dispatch the persona a task implies, on the tier that persona declares",
		Long: `auto dispatches the persona the TASK implies, and sends that persona's own record with it.

It derives what ` + "`dotf harness suggest`" + ` derives — the trigger rules a task matches,
and the personas whose skills those rules name — then narrows to ONE, reads the
tier that persona's record declares for itself, prepends the record to the task,
and walks that tier's chain. One JSON object on stdout, same contract as ` + "`run`" + `.

THE RECORD TRAVELS. This is the half worth more than the routing. ` + "`agent run --role`" + `
labels a dispatch; it does not specialise it — the role reached only the dry-run
backend's echo string, so a generic agent answered and was logged as a reviewer.
Here the persona's mandate, method and boundaries are prepended to the task,
delimited, with the task last. What this can prove is that the record was SENT;
whether the far side honours it is a property of that model.

IT REFUSES RATHER THAN GUESSES, in four places, and each refusal is the
deterministic behaviour rather than a gap in one:

  - two personas match      → names both and dispatches nothing. An advisory
                              layer may print two; a dispatcher runs one process
                              and picking would mean ranking.
  - no rule matches         → a DIFFERENT error, because an ambiguous match is
                              fixed by choosing and an unmatched one cannot be.
  - the persona declares no tier, or one the map has no chain for → refused, never
                              defaulted. A tier chosen here is a route nobody
                              picked, and afterwards indistinguishable from one
                              someone did.
  - the top tier cannot be served → escalates rather than answering from a weaker
                              model (ADR-032 §4). Inherited, and kept visible.

OVERRIDES. --role skips the join entirely and --tier overrides the record. The
output marks each field inferred or dictated, so a caller can tell a derived
route from a supplied one — and so can a reader judging the substring matcher,
which resolves shipper for a task that merely mentions docker.`,
		Example: `  dotf agent auto --task "open a ticket for the bitacora" --backend dry-run --timeout 2m
  dotf agent auto --task "review this diff" --role reviewer --timeout 5m | jq .role`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if task == "" {
				return fmt.Errorf("--task is required: auto derives the persona FROM the task text, " +
					"so a dispatch with no task has neither work to do nor anything to route on")
			}
			if err := requireTimeout(timeout); err != nil {
				return err
			}

			root := dispatchRoot(repoRoot)
			m, err := harness.LoadModelMap(root)
			if err != nil {
				return err
			}

			persona, res, err := resolvePersonaForTask(root, task, role, tier, m)
			if err != nil {
				return err
			}

			// The preamble is composed HERE, before the walk, and not inside
			// Dispatch. The walk retries across pools, and a task composed per
			// attempt could send different bytes to the second pool than to the
			// first — which would make a comparison of the two answers a
			// comparison of two different questions.
			composed, err := harness.PersonaPreamble(persona, task)
			if err != nil {
				return err
			}

			opts, backend, err := prepareDispatch(m, dispatchInputs{
				role: persona.Name, task: composed, tier: res.tier, cwd: cwd,
				backendName: backendName, repoRoot: repoRoot,
				semaphoreDir: semaphoreDir, timeout: timeout,
			})
			if err != nil {
				return err
			}

			rec := agent.Dispatch(cmd.Context(), opts, backend)
			rec.Resolution = &agent.Resolution{
				RoleFrom: res.roleFrom, TierFrom: res.tierFrom, Pattern: res.pattern,
			}
			return emitRecord(cmd, rec)
		},
	}

	c.Flags().StringVar(&task, "task", "", "the work to dispatch; the persona is derived from this text (required)")
	c.Flags().StringVar(&role, "role", "",
		"dispatch as this persona instead of deriving one; skips the join entirely")
	c.Flags().StringVar(&tier, "tier", "",
		"walk this tier's chain instead of the one the persona's record declares")
	c.Flags().StringVar(&cwd, "cwd", "", "working copy the task runs against (default: the current directory)")
	c.Flags().DurationVar(&timeout, "timeout", 0, "per-dispatch deadline, e.g. 90s or 5m (required)")
	c.Flags().StringVar(&backendName, "backend", "",
		"force one backend: subprocess|hive|dry-run (default: probe, preferring subprocess where the harness binary is present)")
	c.Flags().StringVar(&semaphoreDir, "semaphore-dir", "",
		"directory holding per-pool slot state (default: the machine's runtime dir; the budget is a machine property, not a checkout's)")
	c.Flags().StringVar(&repoRoot, "repo-root", "",
		"root containing harness/model-map.json and harness/agents (default: the checkout, else DOTFILES_DIR)")
	return c
}

// route is how the dispatch was arrived at, kept separate from the persona so
// the two refusals below cannot silently produce a persona with no provenance.
type route struct {
	tier               string
	pattern            string
	roleFrom, tierFrom string
}

const (
	inferred = "inferred"
	dictated = "dictated"
)

// resolvePersonaForTask answers WHO runs the task and on WHICH tier, and says
// for each half whether it was derived or supplied.
//
// The roster is loaded from the same root as the map, from ONE resolution, for
// the reason harness_suggest_hook.go:53-58 records: a walk-up that let trigger
// rules come from one root while personas came from another is two halves of a
// join disagreeing about which harness they belong to.
//
// --role skips the join rather than filtering it. That is not an optimisation:
// a task whose text matches no rule must still be dispatchable as a named
// persona, and running the join first would refuse it before the override was
// consulted.
func resolvePersonaForTask(root, task, role, tier string, m map[string]any) (*harness.Persona, route, error) {
	personas, err := harness.LoadPersonas(filepath.Join(root, "harness", "agents"))
	if err != nil {
		return nil, route{}, err
	}

	var r route
	var persona *harness.Persona

	if role != "" {
		r.roleFrom = dictated
		if persona = personaByName(personas, role); persona == nil {
			return nil, route{}, fmt.Errorf(
				"no persona named %q under %s\n\n"+
					"The roster declares: %s. A dispatch as a persona nobody declares would be a "+
					"generic agent wearing a name, which is the state HARNESS-120 exists to end",
				role, filepath.Join(root, "harness", "agents"), personaNames(personas))
		}
	} else {
		r.roleFrom = inferred
		// Read the triggers AT THE RESOLVED ROOT rather than via LoadTriggers,
		// which walks up from the cwd when the explicit root has none. The walk-up
		// is right for an advisory reader and wrong for a dispatcher: it would
		// route work using one checkout's rules and another's personas.
		data, err := os.ReadFile(filepath.Join(root, harness.TriggersFile))
		if err != nil {
			return nil, route{}, fmt.Errorf("read trigger rules at %s to derive a persona: %w", root, err)
		}
		cfg, err := harness.ParseTriggers(data)
		if err != nil {
			return nil, route{}, fmt.Errorf("%s: %w", filepath.Join(root, harness.TriggersFile), err)
		}
		sugg := harness.Suggest(cfg.Triggers, task, nil)
		if persona, r.pattern, err = harness.ResolveOne(sugg, personas); err != nil {
			return nil, route{}, err
		}
	}

	if tier != "" {
		r.tierFrom, r.tier = dictated, tier
		return persona, r, nil
	}
	r.tierFrom = inferred
	if r.tier, err = harness.ResolveTierForPersona(persona, m); err != nil {
		return nil, route{}, err
	}
	return persona, r, nil
}

func personaByName(personas []*harness.Persona, name string) *harness.Persona {
	for _, p := range personas {
		// Only invocable personas, matching ResolveRoles at roles.go:53.
		// `hermes-nan` is kind: autonomous — an externally scheduled steward, not
		// something a session can adopt — so naming it with --role must refuse
		// rather than dispatch a steward as if it were a phase.
		if p != nil && p.Name == name && p.Kind == "invocable" {
			return p
		}
	}
	return nil
}

func personaNames(personas []*harness.Persona) string {
	out := ""
	for _, p := range personas {
		if p == nil || p.Kind != "invocable" {
			continue
		}
		if out != "" {
			out += ", "
		}
		out += p.Name
	}
	return out
}

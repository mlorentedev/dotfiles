package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessResolveSkillsCmd is the third member of the resolve-* family, and it
// exists for the same reason as the other two: the shell renderer must not grow
// a parser of its own.
//
// `compile-harness.sh`'s skill_field reads the INLINE frontmatter form only. Once
// a record declares `skills:` as a block list it returns empty, and the presence
// block renders "MUST consume: none" — enforcement removed, silently, on every
// harness at once. Measured 2026-08-27. `LoadPersona` already reads both forms on
// a real YAML parser, so delegating is strictly cheaper than teaching awk YAML.
//
// TWO CONTRACT DIFFERENCES FROM resolve-capabilities, both deliberate:
//
//   - No `--harness` flag. resolve-capabilities prints a whole frontmatter line
//     because the native field name differs per harness (`tools:` vs
//     `permission:`). A persona's skills reach the harness as PROSE inside a
//     markdown bullet the caller composes, so there is no per-harness field name
//     to hide and no reason to demand one.
//
//   - IDs only, never severity. `enforce:` is what `dotf harness gate` acts on,
//     not what the presence text says — presence asks, bind enforces. Annotating
//     each id would also breach a hard platform limit: the doctrine payload this
//     block lands in is capped at 12000 characters for .gemini/GEMINI.md, and it
//     has been breached twice already, most recently by the roster itself.
//
// The output is byte-identical to skill_field's for every record still in the
// legacy form, which is what makes this substitution provably behaviour-preserving
// rather than merely argued to be.
func newHarnessResolveSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-skills <record>",
		Short: "Print the forced-skill ids one agent record declares, in either frontmatter form",
		Long: `resolve-skills reads one agent record's ` + "`skills:`" + ` frontmatter and prints the
declared skill ids as a YAML flow sequence, accepting both the legacy inline form
and the mapping form that carries per-skill severity:

    skills: [audit, adversarial-review]        # legacy, no severity
    skills:                                    # mapping form
      - id: audit
        enforce: block

Both print the same thing — ` + "`[audit, adversarial-review]`" + ` — because severity is
consumed by ` + "`dotf harness gate`" + `, not by the presence text that names the skills.

A record declaring no skills prints nothing and exits 0; the caller decides what
an empty roster reads as. A ` + "`skills:`" + ` key that is PRESENT but unparseable exits
non-zero and writes nothing to stdout. It never degrades to an empty list: an
unreadable skill list and an empty one produce the same downstream behaviour — a
gate that enforces nothing — and mean opposite things.`,
		Example: `  dotf harness resolve-skills harness/agents/reviewer/AGENT.md
  # [audit, verification-before-completion, adversarial-review, cyclomatic-complexity]`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := harness.LoadPersona(args[0])
			if err != nil {
				return err
			}
			if len(p.Skills) == 0 {
				return nil
			}
			ids := make([]string, 0, len(p.Skills))
			for _, s := range p.Skills {
				ids = append(ids, s.ID)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s]\n", strings.Join(ids, ", "))
			return nil
		},
	}
	return cmd
}

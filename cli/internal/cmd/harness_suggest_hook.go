package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// runSuggestFromHook implements the UserPromptSubmit side of HARNESS-110: read a
// hook payload from stdin, work out which persona the prompt implies, and print
// it as plain stdout, which the runtime adds as context the session can see.
//
// IT RETURNS nil UNCONDITIONALLY, AND THAT IS THE POINT.
//
// On UserPromptSubmit, exit code 2 is documented verbatim as "Blocks prompt
// processing and erases the prompt". A malfunctioning suggester must never be
// able to destroy what the user typed, so fail-open here is not a preference —
// it is the only safe behaviour, and it is strictly stronger than the gate's,
// whose worst case is a refused tool call.
//
// Every failure is therefore a diagnostic on stderr and an empty stdout: a
// missing harness root (any repo that is not this one), an unparseable
// triggers.json, an unreadable persona record, a payload in a shape nobody has
// seen. None of them is worth the price of the user's prompt.
func runSuggestFromHook(cmd *cobra.Command, repoRoot string) error {
	warn := cmd.ErrOrStderr()

	payload, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		_, _ = fmt.Fprintf(warn, "[suggest] read payload: %v\n", err)
		return nil
	}

	prompt, field, ok := harness.PromptFromHookPayload(payload)
	if !ok {
		// Recorded, not swallowed. The field name is undocumented, so a payload
		// shape nobody has seen is a MEASUREMENT worth surfacing — assuming a
		// spelling is exactly how #1434 happened.
		_, _ = fmt.Fprintf(warn, "[suggest] payload unrecognised: no known prompt field (%d bytes)\n", len(payload))
		return nil
	}
	// The arriving spelling is reported every time, so the accepted set stays a
	// measurement rather than a belief.
	_, _ = fmt.Fprintf(warn, "[suggest] prompt arrived as %q\n", field)

	root := harnessRoot(repoRoot)

	// Read the triggers file AT THE RESOLVED ROOT, deliberately not via
	// LoadTriggers: that helper walks up from the cwd when the explicit root has
	// no triggers.json. The hook runs with cwd set to whatever repository the
	// user is in, so the walk-up would let the trigger rules come from one root
	// while the personas come from another — the two halves of a join disagreeing
	// about which harness they belong to. The gate resolves ONE root; so does this.
	triggerData, err := os.ReadFile(filepath.Join(root, harness.TriggersFile))
	if err != nil {
		_, _ = fmt.Fprintf(warn, "[suggest] no triggers at %s: %v\n", root, err)
		return nil
	}
	cfg, err := harness.ParseTriggers(triggerData)
	if err != nil {
		_, _ = fmt.Fprintf(warn, "[suggest] no triggers at %s: %v\n", root, err)
		return nil
	}

	personas, err := harness.LoadPersonas(filepath.Join(root, "harness", "agents"))
	if err != nil {
		_, _ = fmt.Fprintf(warn, "[suggest] no personas at %s: %v\n", root, err)
		return nil
	}

	sugg := harness.Suggest(cfg.Triggers, prompt, nil)
	roles := harness.ResolveRoles(sugg, personas)

	rule := ""
	if len(sugg.Patterns) > 0 {
		rule = sugg.Patterns[0]
	}
	if out := harness.FormatSuggestion(roles, rule, sugg.Skills); out != "" {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), out)
	}
	return nil
}

// harnessRoot resolves where harness/agents and harness/triggers.json live.
//
// The hook does NOT run from the checkout. It runs as a deployed binary with cwd
// set to whatever repository the user happens to be in, so a repo-relative path
// resolves to nothing most of the time. This mirrors loadGatePersona's
// resolution deliberately: one runtime, one answer, and a divergence between the
// gate's root and the suggester's would be invisible until they disagreed.
func harnessRoot(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if dir := os.Getenv("DOTFILES_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(os.Getenv("HOME"), ".dotfiles")
}

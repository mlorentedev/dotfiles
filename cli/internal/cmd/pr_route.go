package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/mlorentedev/dotfiles/cli/internal/prreview"
	"github.com/spf13/cobra"
)

// newPrRouteCmd is `dotf pr route`: which pool member reviews a PR first.
//
// Exit status: 0 routed (the first provider may be `none`), 1 refused (an unknown
// provider override, a NaN preflight that failed on its setup, a pool naming an
// Anthropic model outside the allowlist), 2 usage (no pool source, a pool that
// cannot be read, a draw point outside the total weight). Status 2 is tagged
// with withExitCode; a flag cobra itself rejects still exits 1.
func newPrRouteCmd() *cobra.Command {
	var o routeOpts
	cmd := &cobra.Command{
		Use:   "route",
		Short: "Draw which reviewer pool member reviews a pull request first",
		Long: `Draw the first reviewer of a pull request from the reviewer pool.

The members that answered are drawn by their declared weight. NaN's chain is
what the preflight reported (NAN_OUTCOME, NAN_MODEL, NAN_FALLBACKS,
NAN_PREFLIGHT_EXIT); an Anthropic member is probed first, with the key in
PR_AGENT_ANTHROPIC_API_KEY, and a member that did not answer gives up its
weight. A pull request past the pool's risk line (changed lines, or the label)
goes first to the pool's risk member instead. PR_AGENT_PROVIDER forces a
provider: draw (default), nan, anthropic or nan-only; nan-only never contacts
Anthropic.

The Anthropic models a review may spend on are a fixed list in dotf, each with
its prices and the most one review may cost. A pool naming another model is
refused before any request. --print-allowlist prints that list as JSON.

--pool is required: "embedded" is the pool built into this dotf, anything else is
a file path. A file that cannot be read is an error, never the embedded pool.

Output is key=value lines (first, second, reviewer, route, nan_model,
nan_fallbacks, anthropic_model, anthropic_effort, note), appended to --output or
written to stdout; --json writes one JSON object there instead. Workflow
annotations (::warning::, ::notice::, ::error::) always go to stdout, and the
draw is appended to the job summary when GITHUB_STEP_SUMMARY is set.

Exit status is 0 when routed, 1 when refused and 2 for a usage error.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if o.printAllowlist {
				return writeJSON(c.OutOrStdout(), prreview.Allowlist())
			}
			return o.run(c)
		},
	}
	cmd.Flags().StringVar(&o.pool, "pool", "", "reviewer pool: \"embedded\", or the path of a pool file (required)")
	cmd.Flags().IntVar(&o.changedLines, "changed-lines", 0, "additions plus deletions of the PR (default: PR_CHANGED_LINES)")
	cmd.Flags().StringArrayVar(&o.labels, "labels", nil, "a label of the PR; repeat for several (default: PR_LABELS, a JSON array)")
	cmd.Flags().StringVar(&o.output, "output", "", "append the key=value lines to this file (default: stdout)")
	cmd.Flags().BoolVar(&o.asJSON, "json", false, "write the result as one JSON object instead of key=value lines")
	cmd.Flags().BoolVar(&o.printAllowlist, "print-allowlist", false, "print the Anthropic models a review may spend on, with prices and ceilings, as JSON")
	return cmd
}

type routeOpts struct {
	pool, output   string
	changedLines   int
	labels         []string
	asJSON         bool
	printAllowlist bool
}

// run loads the pool, reads the inputs (flags over environment), routes, and
// reports in the order the workflow expects: annotations while routing, then the
// job summary, the notice and the outputs.
func (o *routeOpts) run(c *cobra.Command) error {
	if o.pool == "" {
		return routeUsage(c, "--pool is required: \"embedded\", or the path of a reviewer pool")
	}
	p, err := prreview.LoadPool(o.pool)
	if err != nil {
		return routeUsage(c, err.Error())
	}
	in := prreview.InputsFromEnv(os.Getenv)
	if c.Flags().Changed("changed-lines") {
		if o.changedLines < 0 {
			return routeUsage(c, fmt.Sprintf("--changed-lines takes a count, got %d", o.changedLines))
		}
		in.ChangedLines = strconv.Itoa(o.changedLines)
	}
	if c.Flags().Changed("labels") {
		raw, _ := json.Marshal(o.labels)
		in.Labels = string(raw)
	}

	res, err := prreview.Route(c.Context(), p, o.pool, in, prreview.Env{Out: c.OutOrStdout()})
	var usage prreview.UsageError
	switch {
	case errors.As(err, &usage):
		return routeUsage(c, usage.Error())
	case errors.Is(err, prreview.ErrRefused):
		return err
	case err != nil:
		c.PrintErrln("pr route:", err)
		return err
	}
	res.Version = dotfVersion
	if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
		if err := appendFile(path, res.Summary()); err != nil {
			c.PrintErrln("pr route: the job summary was not written:", err)
		}
	}
	_, _ = fmt.Fprint(c.OutOrStdout(), res.Notice())

	text := res.KeyValues()
	if o.asJSON {
		js, err := res.JSON()
		if err != nil {
			return err
		}
		text = js + "\n"
	}
	if o.output == "" {
		_, err = fmt.Fprint(c.OutOrStdout(), text)
		return err
	}
	if err := appendFile(o.output, text); err != nil {
		c.PrintErrln("pr route:", err)
		return err
	}
	return nil
}

// routeUsage reports a usage error on stderr and exits 2.
func routeUsage(c *cobra.Command, msg string) error {
	c.PrintErrln(msg)
	return withExitCode(2, errors.New(msg))
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// appendFile appends text to path, creating it: the `>>` the workflow's outputs
// and summary files are written with.
func appendFile(path, text string) (err error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	_, err = io.WriteString(f, text)
	return err
}

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/prreview"
	"github.com/spf13/cobra"
)

// newPrPreflightCmd is `dotf pr preflight`: which model of PR-Agent's declared
// NaN chain NaN answers.
//
// Exit status: 0 a model answered, 1 none did (NaN is out of the review pool for
// this run), 2 usage or setup (no key, model or chain, an output that cannot be
// written). The workflow reads 1 as "NaN is down" and moves the review to the
// pool's other provider, so a setup error must never be 1. Status 2 is tagged
// with withExitCode; a flag cobra itself rejects still exits 1.
func newPrPreflightCmd() *cobra.Command {
	var o preflightOpts
	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Find the first model of PR-Agent's declared NaN chain that NaN answers",
		Long: `Probe each model of PR-Agent's declared NaN chain once, with a small request
and a timeout, and report the first that answers followed by the other models
that answered, in declared order. The chain is --model, then --fallbacks. It
never chooses a model: it only skips dead entries, and a model outside the chain
is never considered.

A model that did not answer becomes a ::warning:: naming the fix for its status
class (a refused key, quota or concurrency, an outage) and a row in the job
summary, appended to GITHUB_STEP_SUMMARY when it is set. If none answers it warns
and exits 1.

The key is read from NAN_API_KEY and sent as one request header; there is no flag
for it. NAN_API_BASE sets the API (default https://api.nan.builders/v1) and
PREFLIGHT_TIMEOUT the seconds per model (default 90).

Output is key=value lines (model, fallbacks), appended to --output or written to
stdout; --json writes one object there instead, with every probe. Annotations go
to stdout.

Exit status is 0 when a model answered, 1 when none did and 2 for a usage or
setup error.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          o.run,
	}
	cmd.Flags().StringVar(&o.model, "model", "", "the declared primary, openai/<id> (required)")
	cmd.Flags().StringVar(&o.fallbacks, "fallbacks", "", "the other declared models, a JSON array of openai/<id> (required)")
	cmd.Flags().StringVar(&o.output, "output", "", "append the key=value lines to this file (default: stdout)")
	cmd.Flags().BoolVar(&o.asJSON, "json", false, "write the result as one JSON object instead of key=value lines")
	return cmd
}

type preflightOpts struct {
	model, fallbacks, output string
	asJSON                   bool
}

// run probes the chain and reports in the order the workflow expects:
// annotations while probing, then the job summary, then the outputs.
func (o *preflightOpts) run(c *cobra.Command, _ []string) error {
	in := prreview.PreflightInputs{Model: o.model, Fallbacks: o.fallbacks, Key: os.Getenv("NAN_API_KEY"),
		APIBase: os.Getenv("NAN_API_BASE"), Timeout: os.Getenv("PREFLIGHT_TIMEOUT")}
	res, err := prreview.Preflight(c.Context(), in, prreview.PreflightEnv{Out: c.OutOrStdout()})
	var setup prreview.PreflightSetupError
	if errors.As(err, &setup) {
		return preflightUsage(c, setup.Error())
	}
	if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
		if serr := appendText(path, res.Summary()); serr != nil {
			c.PrintErrln("pr preflight: the job summary was not written:", serr)
		}
	}
	if err != nil {
		return err
	}
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
	if err := appendText(o.output, text); err != nil {
		return preflightUsage(c, "pr preflight: "+err.Error())
	}
	return nil
}

// preflightUsage reports a usage error on stderr and exits 2.
func preflightUsage(c *cobra.Command, msg string) error {
	c.PrintErrln(msg)
	return withExitCode(2, errors.New(msg))
}

// appendText appends text to path, creating it: the `>>` the workflow's outputs
// and summary files are written with.
func appendText(path, text string) (err error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	_, err = io.WriteString(f, text)
	return err
}

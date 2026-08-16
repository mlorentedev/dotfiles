package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// newSecretsProbeCmd is the sanctioned way to ask what the secrets backend
// actually replied.
//
// It exists because the honest answer to that question used to require `curl`,
// which prints the credential by default. Three credential exposures in this repo
// in one day came from exactly that, each while investigating the previous one,
// and each by someone who had already written the rule against it. The control
// had to stop being prose (CLI-038, #1012).
//
// It is read-only by construction: no unlock, no sync, no write. A probe that
// mutated state could not be run freely while diagnosing, which would defeat it.
// probeClient is the daemon seam for probe, in the same shape as this package's
// other test seams (bwReader, registryPath). Production returns a client on the
// default loopback port; a test points it at an httptest server, which is what
// makes the wiring — flags reaching ShapeProbe, the success predicate, the
// distribution — assertable at all rather than only its refusals.
var probeClient = func() secrets.BWServeClient { return secrets.BWServeClient{} }

func newSecretsProbeCmd() *cobra.Command {
	var raw bool
	var count int
	c := &cobra.Command{
		Use:   "probe <id>",
		Short: "Report what bw serve answered for a secret, without ever printing its value",
		Long: "probe issues the same request the read path issues, through the same client,\n" +
			"and reports what came back:\n\n" +
			"  HTTP status, content-type, byte count\n" +
			"  whether the reply was JSON at all, and the envelope's success/message\n" +
			"  each value's location, length and sha256[:12] fingerprint — never its value\n\n" +
			"It cannot print a secret. --raw echoes a body only for a NON-2xx response,\n" +
			"because a 200 from the item endpoint is the credential itself; that bound is\n" +
			"enforced in the reporting code, not here, so no caller can opt out of it.\n\n" +
			"  dotf secrets probe NAN_API_KEY\n" +
			"  dotf secrets probe NAN_API_KEY --raw       # show a non-2xx body, capped\n" +
			"  dotf secrets probe NAN_API_KEY --count 10  # characterise an intermittent fault",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// A non-positive count would run the loop zero times and exit 0,
			// reporting success for work never done — the failure class this whole
			// ticket is about, reproduced in its own tool. Refused explicitly.
			if count < 1 {
				return fmt.Errorf("--count must be at least 1, got %d", count)
			}
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			s := reg.Lookup(args[0])
			if s == nil {
				return fmt.Errorf("secret %q is not in the registry", args[0])
			}
			if s.Backend != secrets.BackendBW || s.BW == nil || s.BW.Item == "" {
				return fmt.Errorf("secret %q does not resolve through bw serve (backend %q) — nothing to probe",
					s.ID, s.Backend)
			}

			// One client for every iteration. Not a performance choice: --count
			// exists to characterise a fault whose trigger is ORDER, so the
			// sequence of requests has to be a real sequence against one daemon.
			//
			// The trigger is `GET /status`, which poisons this daemon's item-read
			// path for about half a second: 10 item reads before one returned
			// 200x10, and 10 immediately after returned 500x10 (bitwarden/clients
			// #20951, a switchMap/ReplaySubject disposal race).
			//
			// Two earlier explanations of this fault are recorded here because
			// both were wrong and both looked right first. Connection reuse was
			// falsified by DisableKeepAlives over 360 requests (35.0% -> 32.8%
			// failures, i.e. unchanged); concurrency was falsified by 24 requests
			// at each of 1/2/4/8-way parallelism (96/96 clean). What made the fault
			// so hard to see is that THE MEASUREMENT WAS THE CAUSE — every attempt
			// to check the daemon's health called /status and damaged the next
			// read. That is exactly why a probe with a stable, declared request
			// sequence is worth having: an ad-hoc curl loop cannot tell you which
			// of its own steps moved the result.
			client := probeClient()

			itemID, err := client.ProbeItemID(s.BW.Item)
			if err != nil {
				return err
			}
			path := "/object/item/" + url.PathEscape(itemID)

			statuses := map[int]int{}
			for i := 0; i < count; i++ {
				res, probeErr := client.Probe(http.MethodGet, path)
				if probeErr != nil && res.Status == 0 {
					// Transport failure: nothing was answered, so there is no report
					// to shape. Surfaced immediately rather than counted, because a
					// dead daemon is not a data point about the daemon's replies.
					return probeErr
				}
				statuses[res.Status]++
				// Print the full report for a single probe, and — when --raw is
				// set — for every FAILING attempt of a multi-probe run.
				//
				// The earlier shape printed nothing but the distribution once
				// count > 1, which silently made --raw a no-op in the exact
				// combination where it earns its keep: --raw --count N is how you
				// catch an intermittent fault's body without staring at a terminal
				// waiting to get lucky. A flag that quietly does nothing is worse
				// than one that refuses.
				if count == 1 || (raw && !secrets.Is2xx(res.Status)) {
					if count > 1 {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "attempt %d:\n", i+1)
					}
					_, _ = fmt.Fprint(cmd.OutOrStdout(), secrets.ShapeProbe(res, raw).String())
				}
				if probeErr != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "attempt %d: %v\n", i+1, probeErr)
				}
			}

			if count > 1 {
				printDistribution(cmd, statuses, count)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&raw, "raw", false, "echo the response body for NON-2xx replies only, capped")
	c.Flags().IntVar(&count, "count", 1, "probe N times and report the distribution of outcomes")
	return c
}

// printDistribution renders the outcome spread for --count. Statuses only: this
// is the artifact an intermittent backend fault needs and nobody could produce
// safely before.
func printDistribution(cmd *cobra.Command, statuses map[int]int, total int) {
	codes := make([]int, 0, len(statuses))
	for code := range statuses {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d probes, one client, in order:\n", total)
	for _, code := range codes {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  HTTP %d  %d/%d\n", code, statuses[code], total)
	}
	if len(codes) > 1 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(),
			"mixed outcomes for identical requests — the backend is not deterministic here")
	}
}

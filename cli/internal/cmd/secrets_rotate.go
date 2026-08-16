package cmd

import (
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// daemonSyncer is what rotate needs from the bw serve daemon: make it pull the
// current vault state so the read path stops answering from a stale cache. A seam
// so rotate's tests run with no daemon.
type daemonSyncer interface{ Sync() error }

// bwSyncer is the production daemon-sync seam.
var bwSyncer daemonSyncer = &secrets.BWServeDaemon{Client: secrets.BWServeClient{}}

// newSecretsRotateCmd is C7: replace a live credential and prove the replacement
// took, in one command.
//
// It exists because doing this by hand is five steps and two of them are silent
// failures waiting to happen. Rotating a leaked DockerHub PAT on 2026-08-15 hit
// both: the daemon sync was forgotten, so a correct write kept serving the old
// value with no signal; and the liveness probe returned 200 for a token that had
// not actually been replaced, because an unrevoked old credential authenticates
// exactly as well as a new one. Neither is an operator mistake — the sequence
// offers no way to tell those states apart.
//
// So rotate reports a FINGERPRINT change, not just a probe result. "The value
// changed" and "the value works" are different claims and both are required; a
// rotation that writes to the wrong field satisfies the second and fails the
// first, which is precisely the case that looked successful by hand.
func newSecretsRotateCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "rotate <id> [var]",
		Short: "Replace a secret's value and prove the replacement took (write, sync, re-resolve, probe)",
		Long: "rotate replaces the value of a registry secret and verifies the replacement\n" +
			"end to end, which `set` alone cannot do:\n\n" +
			"  1. fingerprint the current value (sha256, first 12 hex — never the value)\n" +
			"  2. read the new value from stdin when piped, else a hidden prompt\n" +
			"  3. refuse a no-op: a new value equal to the current one is a typo, not a rotation\n" +
			"  4. write it through the same idempotent path as `set`\n" +
			"  5. sync the bw serve daemon, so reads stop answering from a stale cache\n" +
			"  6. re-resolve through the normal read path and fingerprint again\n" +
			"  7. run the entry's `validate:` liveness probe when it declares one\n\n" +
			"The fingerprints are the point. A liveness probe cannot tell a rotated\n" +
			"credential from an old one that was never revoked — both authenticate. A\n" +
			"changed fingerprint proves the value was actually replaced.\n\n" +
			"  printf %s \"$new\" | dotf secrets rotate DOCKERHUB_TOKEN\n" +
			"  dotf secrets rotate DOCKERHUB_TOKEN          # prompts (hidden)\n" +
			"  dotf secrets rotate DOCKERHUB_TOKEN --dry-run",
		Args:         cobra.RangeArgs(1, 2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			s := reg.Lookup(args[0])
			if s == nil {
				return fmt.Errorf("unknown secret %q (try `dotf secrets ls`)", args[0])
			}
			varArg := ""
			if len(args) == 2 {
				varArg = args[1]
			}
			item, field, isFile, err := s.BWTarget(varArg)
			if err != nil {
				return err
			}
			return runRotate(cmd, s, item, field, isFile, dryRun)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report the intended rotation and the current fingerprint without writing")
	return c
}

// runRotate is the rotation proper, split out so the command body stays within the
// < 40-line / < 10-complexity budget (AGENTS.md).
func runRotate(cmd *cobra.Command, s *secrets.Secret, item, field string, isFile, dryRun bool) error {
	out := cmd.OutOrStdout()

	before, err := bwRead().Field(item, field)
	if err != nil {
		// Unlike `set`, rotate never creates: rotating something that does not
		// exist is a provisioning action, and conflating the two is how a locked
		// vault turns into a duplicate item (#612).
		return fmt.Errorf("read current value of %s / %s (rotate replaces, it never creates — use `dotf secrets set` to provision): %w", item, field, err)
	}
	beforeFP := secrets.Fingerprint(normalizeValue(before, isFile))
	_, _ = fmt.Fprintf(out, "current  %s / %s  fingerprint %s\n", item, field, beforeFP)

	if dryRun {
		_, _ = fmt.Fprintf(out, "would rotate  %s / %s%s\n", item, field, probeSuffix(s))
		return nil
	}

	value, err := readSecretValue(cmd, isFile)
	if err != nil {
		return err
	}
	value = normalizeValue(value, isFile)
	if value == "" {
		return fmt.Errorf("refusing to write an empty value for %q (a blank secret is a bug, not a clear)", s.ID)
	}
	if secrets.Fingerprint(value) == beforeFP {
		return fmt.Errorf("the new value is identical to the current one — that is not a rotation. Nothing was written")
	}

	if err := bwWrite().SetField(item, field, value); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "written  %s / %s\n", item, field)

	return confirmRotation(cmd, s, item, field, isFile, beforeFP)
}

// confirmRotation performs the half a bare write cannot: make the read path see the
// new value, then prove through that same path that it changed.
func confirmRotation(cmd *cobra.Command, s *secrets.Secret, item, field string, isFile bool, beforeFP string) error {
	out := cmd.OutOrStdout()

	// The daemon answers from its own cache; without this the write is correct and
	// every read still returns the old value, with nothing to indicate a missing
	// step. A sync failure is reported, not fatal — the write did happen, and
	// hiding that would be worse than a noisy success.
	if err := bwSyncer.Sync(); err != nil {
		_, _ = fmt.Fprintf(out, "WARNING  daemon sync failed (%v) — the value was written but reads may still serve the old one until `dotf secrets unlock` or a manual sync\n", err)
	}

	after, err := bwRead().Field(item, field)
	if err != nil {
		return fmt.Errorf("wrote the new value but could not read it back through the normal path: %w", err)
	}
	afterFP := secrets.Fingerprint(normalizeValue(after, isFile))
	if afterFP == beforeFP {
		return fmt.Errorf("the value read back is still the old one (fingerprint %s unchanged) — the write did not take effect on the read path", beforeFP)
	}
	_, _ = fmt.Fprintf(out, "rotated  %s / %s  %s -> %s\n", item, field, beforeFP, afterFP)

	return probeRotated(cmd, s, after)
}

// probeRotated runs the entry's declared liveness check, reusing the mechanism
// `secrets sync ci` already gates uploads with. A secret that declares nothing is
// not probed — liveness cannot be checked generically across providers, and
// inventing a probe would be worse than admitting there is none.
func probeRotated(cmd *cobra.Command, s *secrets.Secret, value string) error {
	if s.Validate != "github-token" {
		if s.Validate != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "note     no probe implemented for validate: %s — liveness unverified\n", s.Validate)
		}
		return nil
	}
	if err := ghTokenValidator.Validate(value); err != nil {
		return fmt.Errorf("the new value was written and read back, but it does not authenticate: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "verified live github token")
	return nil
}

// probeSuffix names the probe a real run would perform, so --dry-run reports the
// whole intended action rather than only the write.
func probeSuffix(s *secrets.Secret) string {
	if s.Validate == "" {
		return " (no liveness probe declared)"
	}
	return " (then probe: " + s.Validate + ")"
}

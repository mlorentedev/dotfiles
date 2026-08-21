package cmd

import (
	"fmt"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// newSecretsMigrateCmd is the C4 cutover: move one secret age→bw, behind a parity gate.
// It composes the primitives — `set`'s write core (applySet), the age resolver, and
// SetBackendBW (the registry flip). The registry flip is the LAST mutation: until it
// lands the secret still resolves via age, so any earlier failure is a safe no-op and
// re-runs are idempotent.
func newSecretsMigrateCmd() *cobra.Command {
	var dryRun, assumeYes bool
	c := &cobra.Command{
		Use:   "migrate <id>",
		Short: "Cut one secret over from the age store to Bitwarden, behind a parity gate",
		Long: "migrate moves one registry secret from the age store to Bitwarden:\n" +
			"  read age → write bw (set's core) → re-read & compare (parity gate) →\n" +
			"  flip the registry entry to backend bw → verify.\n\n" +
			"The registry flip is the last step, so a failure before it leaves the secret\n" +
			"fully working on age (safe rollback); the .secret.age file is KEPT (retire is\n" +
			"separate). Idempotent: an already-bw secret is re-verified and left alone.\n" +
			"--yes creates the absent Bitwarden item; --dry-run reports without writing.\n" +
			"File secrets migrate byte-exact (no newline trimming).\n\n" +
			"Out of scope (refused with a specific reason): a secret sharing its age\n" +
			"source with another (split tokens manually per #321).",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			s := reg.Lookup(id)
			if s == nil {
				return fmt.Errorf("unknown secret %q (try `dotf secrets ls`)", id)
			}
			out := cmd.OutOrStdout()
			loader := secretLoader()

			if s.Backend == "bw" { // idempotent short-circuit: already migrated
				item, field, _, err := s.BWTarget("")
				if err != nil {
					return err
				}
				if vErr := loader.Verify(secrets.Entry{Var: id, Backend: "bw", Item: item, Field: field}); vErr != nil {
					return fmt.Errorf("%q is already bw but does not resolve: %w", id, vErr)
				}
				_, _ = fmt.Fprintf(out, "already bw (verified)  %s\n", id)
				return nil
			}

			if err := migrateGuard(reg, s); err != nil {
				return err
			}
			item, field, isFile, err := s.BWTarget("") // guard ensured a bw: block exists
			if err != nil {
				return err
			}

			value, err := ageValue(reg, loader, s, isFile)
			if err != nil {
				return err
			}

			// Write the age value into bw (idempotent; creates the item with --yes).
			if err := applySet(cmd, item, field, value, s.BW.Folder, isFile, dryRun, assumeYes); err != nil {
				return err
			}
			if dryRun {
				_, _ = fmt.Fprintf(out, "would flip registry: %s → backend bw (%s / %s)\n", id, item, field)
				return nil
			}

			// Parity gate — re-read bw and compare; abort BEFORE the registry flip.
			back, err := bwRead().Field(item, field)
			if err != nil {
				return fmt.Errorf("parity read-back of %s/%s failed (registry NOT flipped): %w", item, field, err)
			}
			if normalizeValue(back, isFile) != value {
				return fmt.Errorf("parity mismatch for %q: bw read-back differs from the age value (age %d bytes, bw %d) — registry NOT flipped",
					id, len(value), len(normalizeValue(back, isFile)))
			}

			// Flip the registry — the last mutation. The target is the entry's
			// pre-declared bw: block (the same item/field parity verified above), so the
			// flip only toggles backend + drops age; it needs no item/field of its own.
			// Write to the checkout's registry (the version-controlled SSOT), failing
			// loud if no checkout is found: writing the deployed copy would be silently
			// reverted on the next redeploy and never reach git (#635).
			wp, err := registryWritePath()
			if err != nil {
				return err
			}
			if err := secrets.FlipRegistryToBW(wp, id); err != nil {
				return err
			}
			if err := loader.Verify(secrets.Entry{Var: id, Backend: "bw", Item: item, Field: field}); err != nil {
				return fmt.Errorf("%q flipped to bw but verify failed: %w", id, err)
			}
			_, _ = fmt.Fprintf(out, "migrated  %s → bw (%s / %s); .secret.age kept (retire separately)\n", id, item, field)
			return nil
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report the intended write + registry flip without writing")
	c.Flags().BoolVarP(&assumeYes, "yes", "y", false, "create the absent Bitwarden item during the write")
	return c
}

// migrateGuard refuses the shapes C4 does not handle, each with a specific reason and no
// side effects. Order matters only for message quality.
func migrateGuard(reg *secrets.Registry, s *secrets.Secret) error {
	if s.Backend == "age-offline" || s.Plane == "floor" {
		return fmt.Errorf("%q is a floor secret — never migrated (needed before Bitwarden is reachable)", s.ID)
	}
	if s.BW == nil || s.BW.Item == "" {
		return fmt.Errorf("%q has no bw: target — declare it in the registry first", s.ID)
	}
	// A ci:* consumer is no longer a barrier: `dotf secrets sync ci` resolves the value
	// backend-agnostically, so a migrated (bw) secret still reaches Actions. The former
	// guard (age-only `ls --pairs` would drop it) was retired with the shell upload path.
	if s.Age != "" {
		shared := 0
		for i := range reg.Secrets {
			if reg.Secrets[i].Age == s.Age {
				shared++
			}
		}
		if shared > 1 {
			return fmt.Errorf("%q shares its age source %q with another entry — split into distinct Bitwarden tokens manually (tracked by #321)", s.ID, s.Age)
		}
	}
	return nil
}

// ageValue resolves the secret's current age plaintext. A file secret is byte-exact —
// zero transformation, matching normalizeValue(value, isFile=true)'s no-trim contract for
// the bw side (file content, e.g. a kubeconfig, must not gain or lose a trailing byte). An
// env secret is trimmed of exactly the trailing newline `age -d` appends — the same trim
// normalizeValue applies on the bw side there — but keeps interior newlines: a multi-line
// secret (BEEHIIV_DNS_RECORDS) must survive the age→bw cutover intact, not get flattened
// to one line (#612 B6). Unlike EnvFor (built for single-line child-process env tokens),
// neither path strips interior newlines.
func ageValue(reg *secrets.Registry, loader *secrets.Loader, s *secrets.Secret, isFile bool) (string, error) {
	target := s.Vars()[0]
	for _, e := range reg.Entries(env.Home()) {
		if e.Var == target {
			plaintext, err := loader.RawResolve(e)
			if err != nil {
				return "", err
			}
			value := string(plaintext)
			if !isFile {
				value = strings.TrimRight(value, "\r\n")
			}
			if value == "" {
				return "", fmt.Errorf("secret %q resolved to an empty value — refusing to migrate", s.ID)
			}
			return value, nil
		}
	}
	return "", fmt.Errorf("no resolvable entry for %q", s.ID)
}

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// bwWriteClient is what `set` needs from the bw write seam: update an existing field
// (BWWriter) and create an absent item (BWCreator). Two small interfaces, one prod
// impl (BWPut) — keeping create separate from update so update-only callers cannot
// create by accident.
type bwWriteClient interface {
	secrets.BWWriter
	secrets.BWCreator
}

// bwWriter is the bw write seam (BWPut in production), overridable so set's tests run
// with no unlocked Bitwarden — the write analog of bwReader.
var bwWriter bwWriteClient = secrets.BWPut{}

// stdinIsTerminal reports whether stdin is an interactive terminal; a var so tests
// force the non-interactive (piped) path.
var stdinIsTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// readPassword reads a value from the terminal without echo. A var because it reads the
// real fd (so it is live/manual-tested, like BWPut's shell-out) and tests never hit it.
var readPassword = func() ([]byte, error) { return term.ReadPassword(int(os.Stdin.Fd())) }

// newSecretsSetCmd is the C3 write primitive: write a value into the Bitwarden item a
// registry secret maps to. Idempotent (read-back compare), value never on the command
// line (stdin or hidden prompt), create-absent gated. `migrate` (C4) and `rotate` (C7)
// compose it.
func newSecretsSetCmd() *cobra.Command {
	var dryRun, assumeYes bool
	c := &cobra.Command{
		Use:   "set <id> [var]",
		Short: "Write a secret value into its Bitwarden item (idempotent; value via stdin or hidden prompt)",
		Long: "set writes a value into the Bitwarden item/field a registry secret maps to\n" +
			"(secrets/registry.yaml). The value is read from stdin when piped, else from a\n" +
			"hidden terminal prompt — never the command line (no shell-history / ps leak):\n" +
			"  printf %s \"$value\" | dotf secrets set openai-api-key\n" +
			"  dotf secrets set openai-api-key            # prompts (hidden)\n\n" +
			"It is idempotent: the current field value is read first and, if it already\n" +
			"equals the new value, nothing is written. A multi-field secret needs [var] to\n" +
			"pick which field; a single-field secret does not. env values are stored with the\n" +
			"trailing newline trimmed; file/notes values byte-exact (multi-line text). When\n" +
			"the item does not exist, set offers to create it (confirm, or --yes) — but only\n" +
			"when Bitwarden reports it genuinely absent; a locked vault fails loud instead of\n" +
			"risking a duplicate. --dry-run reports the action without writing.",
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

			value, err := readSecretValue(cmd, isFile)
			if err != nil {
				return err
			}
			value = normalizeValue(value, isFile)
			if value == "" {
				return fmt.Errorf("refusing to write an empty value for %q (a blank secret is a bug, not a clear)", args[0])
			}
			return applySet(cmd, item, field, value, isFile, dryRun, assumeYes)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report the intended action (update/create/unchanged) without writing")
	c.Flags().BoolVarP(&assumeYes, "yes", "y", false, "create an absent item without prompting (for non-interactive callers like migrate)")
	return c
}

// readSecretValue reads the secret value: all of stdin when piped, else a hidden
// terminal prompt. A file/multi-line secret cannot be entered at the hidden prompt, so
// interactive use of one is rejected with a pipe-it instruction.
func readSecretValue(cmd *cobra.Command, isFile bool) (string, error) {
	if stdinIsTerminal() {
		if isFile {
			return "", fmt.Errorf("this is a multi-line/file secret; pipe the value via stdin, e.g. `dotf secrets set <id> < file`")
		}
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Value (hidden): ")
		b, err := readPassword()
		_, _ = fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", fmt.Errorf("read value from terminal: %w", err)
		}
		return string(b), nil
	}
	b, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", fmt.Errorf("read value from stdin: %w", err)
	}
	return string(b), nil
}

// normalizeValue trims a single-line env token's trailing newline(s); a file/notes
// value is left byte-exact so multi-line text (SSH keys, kubeconfigs) round-trips.
func normalizeValue(value string, isFile bool) string {
	if isFile {
		return value
	}
	return strings.TrimRight(value, "\r\n")
}

// applySet dispatches on the current state of item/field, read back through bwReader:
//
//   - value already current  -> no-op (idempotent), report "unchanged"
//   - field present, differs  -> SetField (update)
//   - item present, field absent (ErrBWFieldNotFound) -> SetField (append the field)
//   - item absent (ErrBWItemNotFound) -> the gated create path
//   - any other read error (locked / unauthenticated vault) -> fail loud, never create
//
// Keeping ErrBWItemNotFound (create) apart from a generic read error (fail) is the
// security crux of #612: a locked vault must never be mistaken for an absent item.
func applySet(cmd *cobra.Command, item, field, value string, isFile, dryRun, assumeYes bool) error {
	out := cmd.OutOrStdout()
	cur, err := bwReader.Field(item, field)
	switch {
	case err == nil:
		if normalizeValue(cur, isFile) == value {
			_, _ = fmt.Fprintf(out, "unchanged  %s / %s\n", item, field)
			return nil
		}
		if dryRun {
			_, _ = fmt.Fprintf(out, "would update  %s / %s\n", item, field)
			return nil
		}
		if err := bwWriter.SetField(item, field, value); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "updated  %s / %s\n", item, field)
		return nil

	case errors.Is(err, secrets.ErrBWFieldNotFound):
		if dryRun {
			_, _ = fmt.Fprintf(out, "would add field  %s / %s\n", item, field)
			return nil
		}
		if err := bwWriter.SetField(item, field, value); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "added field  %s / %s\n", item, field)
		return nil

	case errors.Is(err, secrets.ErrBWItemNotFound):
		return createAbsent(cmd, item, field, value, dryRun, assumeYes)

	default:
		return fmt.Errorf("read current value of %s / %s: %w", item, field, err)
	}
}

// createAbsent handles a genuinely-missing item: refused under --dry-run (reported),
// otherwise confirmed on the TTY or via --yes. Non-interactive callers MUST pass --yes
// — stdin is already consumed by the piped value, so there is no channel to confirm on.
func createAbsent(cmd *cobra.Command, item, field, value string, dryRun, assumeYes bool) error {
	out := cmd.OutOrStdout()
	if dryRun {
		_, _ = fmt.Fprintf(out, "would create item  %s (field %s)\n", item, field)
		return nil
	}
	if !assumeYes {
		if !stdinIsTerminal() {
			return fmt.Errorf("item %q not found; re-run with --yes to create it non-interactively", item)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Bitwarden item %q not found. Create it with field %q? [y/N] ", item, field)
		if !readYes(cmd.InOrStdin()) {
			return fmt.Errorf("aborted: item %q not created", item)
		}
	}
	if err := bwWriter.CreateItem(item, field, value); err != nil {
		return err
	}
	fmt.Fprintf(out, "created item  %s (field %s)\n", item, field)
	return nil
}

// readYes reads one line and reports whether it is an affirmative (y/yes).
func readYes(r io.Reader) bool {
	line, _ := bufio.NewReader(r).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
